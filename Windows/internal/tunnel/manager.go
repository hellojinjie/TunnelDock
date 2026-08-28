package tunnel

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/hellojinjie/TunnelDock/Windows/internal/model"
	"github.com/hellojinjie/TunnelDock/Windows/internal/sshclient"
)

var (
	ErrHostUnavailable         = errors.New("SSH Host is unavailable")
	ErrTunnelNotFound          = errors.New("tunnel not found")
	ErrTunnelNotTemporary      = errors.New("tunnel is not temporary")
	ErrTunnelRunning           = errors.New("running tunnel cannot be deleted")
	ErrApplicationShuttingDown = errors.New("application is shutting down")
	ErrPortUnavailable         = errors.New("local port is unavailable")
)

type DefinitionRepository interface {
	Load() ([]model.TunnelDefinition, error)
	ReplaceAll([]model.TunnelDefinition) error
}

type ManagerPortChecker interface {
	Check(Endpoint) PortStatus
	Reserve(Endpoint) bool
	Release(Endpoint)
}

type ManagedProcess interface {
	Events() <-chan sshclient.ProcessEvent
	IsRunning() bool
	Terminate() error
	Done() <-chan struct{}
}

type ManagedProcessController interface {
	Start(ctx context.Context, runtimeID string, definition model.TunnelDefinition) (ManagedProcess, error)
	WaitUntilReady(ctx context.Context, process ManagedProcess, address string, port uint16, timeout time.Duration) error
}

type ManagerClock interface {
	Sleep(ctx context.Context, duration time.Duration) error
}

type realManagerClock struct{}

func (realManagerClock) Sleep(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

type ManagerOptions struct {
	Repository DefinitionRepository
	Ports      ManagerPortChecker
	Processes  ManagedProcessController
	Clock      ManagerClock
	Now        func() time.Time
}

type Manager struct {
	mu           sync.Mutex
	repository   DefinitionRepository
	ports        ManagerPortChecker
	processes    ManagedProcessController
	clock        ManagerClock
	now          func() time.Time
	runtimes     map[string]*managedRuntime
	savedOrder   []string
	hosts        map[string]bool
	shuttingDown bool
}

type managedRuntime struct {
	definition    model.TunnelDefinition
	temporary     bool
	state         model.TunnelState
	desired       bool
	lastError     string
	log           *LogBuffer
	process       ManagedProcess
	cancel        context.CancelFunc
	ctx           context.Context
	generation    uint64
	connectedOnce bool
	retryAttempt  int
	reserved      bool
}

func NewManager(options ManagerOptions) *Manager {
	clock := options.Clock
	if clock == nil {
		clock = realManagerClock{}
	}
	now := options.Now
	if now == nil {
		now = time.Now
	}
	return &Manager{
		repository: options.Repository, ports: options.Ports, processes: options.Processes,
		clock: clock, now: now, runtimes: make(map[string]*managedRuntime), hosts: make(map[string]bool),
	}
}

func (m *Manager) LoadSavedDefinitions() error {
	definitions, err := m.repository.Load()
	if err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, definition := range definitions {
		m.runtimes[definition.ID] = &managedRuntime{definition: definition, state: model.StateDisconnected, log: NewLogBuffer()}
		m.savedOrder = append(m.savedOrder, definition.ID)
	}
	return nil
}

func (m *Manager) UpdateHosts(hosts []model.SSHHost) {
	available := make(map[string]bool, len(hosts))
	for _, host := range hosts {
		if host.Availability == model.HostAvailable {
			available[host.Alias] = true
		}
	}
	m.mu.Lock()
	m.hosts = available
	m.mu.Unlock()
}

func (m *Manager) ConnectSaved(ctx context.Context, id string) error {
	m.mu.Lock()
	runtime, exists := m.runtimes[id]
	if !exists || runtime.temporary {
		m.mu.Unlock()
		return ErrTunnelNotFound
	}
	if runtime.state == model.StateConnecting || runtime.state == model.StateConnected || runtime.state == model.StateReconnecting {
		m.mu.Unlock()
		return nil
	}
	m.mu.Unlock()
	return m.connectInitial(ctx, id)
}

func (m *Manager) ConnectTemporary(ctx context.Context, definition model.TunnelDefinition) (string, error) {
	if definition.ID == "" {
		id, err := model.NewUUIDv4()
		if err != nil {
			return "", err
		}
		definition.ID = id
	}
	now := m.now()
	if definition.CreatedAt.IsZero() {
		definition.CreatedAt = now
	}
	if definition.UpdatedAt.IsZero() {
		definition.UpdatedAt = now
	}
	if err := definition.Validate(); err != nil {
		return "", err
	}
	m.mu.Lock()
	if _, exists := m.runtimes[definition.ID]; exists {
		m.mu.Unlock()
		return "", fmt.Errorf("duplicate runtime ID %s", definition.ID)
	}
	m.runtimes[definition.ID] = &managedRuntime{definition: definition, temporary: true, state: model.StateDisconnected, log: NewLogBuffer()}
	m.mu.Unlock()
	if err := m.connectInitial(ctx, definition.ID); err != nil {
		m.mu.Lock()
		delete(m.runtimes, definition.ID)
		m.mu.Unlock()
		return "", err
	}
	return definition.ID, nil
}

func (m *Manager) connectInitial(_ context.Context, id string) error {
	m.mu.Lock()
	runtime, exists := m.runtimes[id]
	if !exists {
		m.mu.Unlock()
		return ErrTunnelNotFound
	}
	if m.shuttingDown {
		m.mu.Unlock()
		return ErrApplicationShuttingDown
	}
	if !m.hosts[runtime.definition.HostAlias] {
		m.mu.Unlock()
		return ErrHostUnavailable
	}
	endpoint := endpointFor(runtime.definition)
	if status := m.ports.Check(endpoint); status != PortAvailable {
		m.mu.Unlock()
		return fmt.Errorf("%w: %v", ErrPortUnavailable, status)
	}
	if !m.ports.Reserve(endpoint) {
		m.mu.Unlock()
		return fmt.Errorf("%w: reserved by TunnelDock", ErrPortUnavailable)
	}
	runtime.reserved = true
	runtime.desired = true
	runtime.state = model.StateConnecting
	runtime.lastError = ""
	runtime.generation++
	generation := runtime.generation
	runtime.ctx, runtime.cancel = context.WithCancel(context.Background())
	runtime.log.AddAt(m.now(), "Connecting...")
	m.mu.Unlock()

	if err := m.launch(id, generation, true); err != nil {
		return err
	}
	return nil
}

func (m *Manager) launch(id string, generation uint64, initial bool) error {
	m.mu.Lock()
	runtime, exists := m.runtimes[id]
	if !exists || runtime.generation != generation || !runtime.desired {
		m.mu.Unlock()
		return context.Canceled
	}
	ctx := runtime.ctx
	definition := runtime.definition
	m.mu.Unlock()

	process, err := m.processes.Start(ctx, id, definition)
	if err == nil {
		m.mu.Lock()
		if current, ok := m.runtimes[id]; ok && current.generation == generation && current.desired {
			current.process = process
		} else {
			m.mu.Unlock()
			_ = process.Terminate()
			return context.Canceled
		}
		m.mu.Unlock()
		err = m.processes.WaitUntilReady(ctx, process, definition.LocalAddress, definition.LocalPort, 5*time.Second)
	}
	if err != nil {
		stderr := startupStderr(process)
		if process != nil {
			_ = process.Terminate()
		}
		return m.launchFailed(id, generation, initial, sshclient.NewConnectionFailure(stderr, err))
	}

	now := m.now()
	m.mu.Lock()
	runtime, exists = m.runtimes[id]
	if !exists || runtime.generation != generation || !runtime.desired {
		m.mu.Unlock()
		_ = process.Terminate()
		return context.Canceled
	}
	runtime.process = process
	runtime.state = model.StateConnected
	runtime.connectedOnce = true
	runtime.retryAttempt = 0
	runtime.lastError = ""
	runtime.definition.LastConnectedAt = &now
	runtime.log.AddAt(now, "Connected.")
	temporary := runtime.temporary
	m.mu.Unlock()
	if !temporary {
		_ = m.persistDefinitions()
	}
	go m.monitor(id, generation, process)
	return nil
}

// startupStderr drains output emitted before the process reaches the normal
// monitor. In particular, ssh.exe may exit during readiness checks; retaining
// these lines turns "process exited" into the actual OpenSSH explanation.
func startupStderr(process ManagedProcess) string {
	if process == nil {
		return ""
	}
	var lines []string
	for {
		select {
		case event, open := <-process.Events():
			if !open {
				return strings.Join(lines, "\n")
			}
			if event.Kind == sshclient.ProcessEventStderr && strings.TrimSpace(event.Data) != "" {
				lines = append(lines, event.Data)
			}
		default:
			return strings.Join(lines, "\n")
		}
	}
}

func (m *Manager) launchFailed(id string, generation uint64, initial bool, launchErr error) error {
	m.mu.Lock()
	runtime, exists := m.runtimes[id]
	if !exists || runtime.generation != generation {
		m.mu.Unlock()
		return launchErr
	}
	runtime.process = nil
	runtime.lastError = launchErr.Error()
	runtime.log.AddAt(m.now(), launchErr.Error())
	if initial {
		runtime.state = model.StateFailed
		runtime.desired = false
		if runtime.cancel != nil {
			runtime.cancel()
		}
		if runtime.reserved {
			m.ports.Release(endpointFor(runtime.definition))
			runtime.reserved = false
		}
	} else {
		runtime.state = model.StateReconnecting
	}
	m.mu.Unlock()
	return launchErr
}

func (m *Manager) monitor(id string, generation uint64, process ManagedProcess) {
	exited := false
	for event := range process.Events() {
		m.mu.Lock()
		runtime, exists := m.runtimes[id]
		if !exists || runtime.generation != generation || runtime.process != process {
			m.mu.Unlock()
			return
		}
		switch event.Kind {
		case sshclient.ProcessEventStdout, sshclient.ProcessEventStderr:
			runtime.log.Add(event.Data)
		case sshclient.ProcessEventReadError:
			if event.Err != nil {
				runtime.log.Add(event.Err.Error())
			}
		case sshclient.ProcessEventExited:
			runtime.log.AddAt(m.now(), fmt.Sprintf("SSH process exited: %d", event.ExitCode))
			exited = true
		}
		m.mu.Unlock()
		if exited {
			break
		}
	}
	if exited {
		m.processExited(id, generation, process)
	}
}

func (m *Manager) processExited(id string, generation uint64, process ManagedProcess) {
	m.mu.Lock()
	runtime, exists := m.runtimes[id]
	if !exists || runtime.generation != generation || runtime.process != process {
		m.mu.Unlock()
		return
	}
	runtime.process = nil
	if !runtime.desired {
		m.mu.Unlock()
		return
	}
	if !m.hosts[runtime.definition.HostAlias] {
		runtime.state = model.StateFailed
		runtime.desired = false
		runtime.lastError = ErrHostUnavailable.Error()
		if runtime.cancel != nil {
			runtime.cancel()
		}
		if runtime.reserved {
			m.ports.Release(endpointFor(runtime.definition))
			runtime.reserved = false
		}
		m.mu.Unlock()
		return
	}
	runtime.state = model.StateReconnecting
	ctx := runtime.ctx
	m.mu.Unlock()
	go m.reconnectLoop(ctx, id, generation)
}

func (m *Manager) reconnectLoop(ctx context.Context, id string, generation uint64) {
	for {
		m.mu.Lock()
		runtime, exists := m.runtimes[id]
		if !exists || runtime.generation != generation || !runtime.desired {
			m.mu.Unlock()
			return
		}
		delay := reconnectDelay(runtime.retryAttempt)
		runtime.retryAttempt++
		runtime.log.AddAt(m.now(), fmt.Sprintf("Reconnecting in %s...", delay))
		m.mu.Unlock()
		if err := m.clock.Sleep(ctx, delay); err != nil {
			return
		}

		m.mu.Lock()
		runtime, exists = m.runtimes[id]
		if !exists || runtime.generation != generation || !runtime.desired {
			m.mu.Unlock()
			return
		}
		if !m.hosts[runtime.definition.HostAlias] {
			runtime.state = model.StateFailed
			runtime.desired = false
			runtime.lastError = ErrHostUnavailable.Error()
			if runtime.reserved {
				m.ports.Release(endpointFor(runtime.definition))
				runtime.reserved = false
			}
			m.mu.Unlock()
			return
		}
		m.mu.Unlock()
		if err := m.launch(id, generation, false); err == nil {
			return
		}
	}
}

func reconnectDelay(attempt int) time.Duration {
	delays := [...]time.Duration{time.Second, 2 * time.Second, 5 * time.Second, 10 * time.Second, 30 * time.Second}
	if attempt >= len(delays) {
		return delays[len(delays)-1]
	}
	return delays[attempt]
}

func (m *Manager) Disconnect(id string) error {
	m.mu.Lock()
	runtime, exists := m.runtimes[id]
	if !exists {
		m.mu.Unlock()
		return ErrTunnelNotFound
	}
	runtime.desired = false
	runtime.generation++
	if runtime.cancel != nil {
		runtime.cancel()
	}
	process := runtime.process
	runtime.process = nil
	runtime.state = model.StateDisconnected
	if runtime.reserved {
		m.ports.Release(endpointFor(runtime.definition))
		runtime.reserved = false
	}
	temporary := runtime.temporary
	if temporary {
		delete(m.runtimes, id)
	}
	m.mu.Unlock()
	if process != nil {
		_ = process.Terminate()
	}
	return nil
}

// Shutdown prevents new connections and reconnects, then terminates every
// managed SSH process. The application closes its Job Object afterwards as the
// final process-ownership safety net.
func (m *Manager) Shutdown() error {
	m.mu.Lock()
	if m.shuttingDown {
		m.mu.Unlock()
		return nil
	}
	m.shuttingDown = true
	processes := make([]ManagedProcess, 0, len(m.runtimes))
	for _, runtime := range m.runtimes {
		runtime.desired = false
		runtime.generation++
		if runtime.cancel != nil {
			runtime.cancel()
		}
		if runtime.process != nil {
			processes = append(processes, runtime.process)
			runtime.process = nil
		}
		if runtime.reserved {
			m.ports.Release(endpointFor(runtime.definition))
			runtime.reserved = false
		}
		runtime.state = model.StateDisconnected
	}
	m.mu.Unlock()
	for _, process := range processes {
		_ = process.Terminate()
	}
	for _, process := range processes {
		<-process.Done()
	}
	return nil
}

func (m *Manager) SaveTemporary(id string) (model.TunnelDefinition, error) {
	m.mu.Lock()
	runtime, exists := m.runtimes[id]
	if !exists {
		m.mu.Unlock()
		return model.TunnelDefinition{}, ErrTunnelNotFound
	}
	if !runtime.temporary {
		m.mu.Unlock()
		return model.TunnelDefinition{}, ErrTunnelNotTemporary
	}
	runtime.temporary = false
	runtime.definition.UpdatedAt = m.now()
	m.savedOrder = append(m.savedOrder, id)
	definition := runtime.definition
	m.mu.Unlock()
	if err := m.persistDefinitions(); err != nil {
		m.mu.Lock()
		runtime.temporary = true
		m.savedOrder = m.savedOrder[:len(m.savedOrder)-1]
		m.mu.Unlock()
		return model.TunnelDefinition{}, err
	}
	return definition, nil
}

// Rename changes a saved tunnel's display name without changing its runtime
// connection state, so it remains available while the tunnel is running.
func (m *Manager) Rename(id, name string) error {
	m.mu.Lock()
	runtime, exists := m.runtimes[id]
	if !exists || runtime.temporary {
		m.mu.Unlock()
		return ErrTunnelNotFound
	}
	updated := runtime.definition
	updated.Name = &name
	updated.UpdatedAt = m.now()
	if err := updated.Validate(); err != nil {
		m.mu.Unlock()
		return err
	}
	previous := runtime.definition
	runtime.definition = updated
	m.mu.Unlock()
	if err := m.persistDefinitions(); err != nil {
		m.mu.Lock()
		if current, ok := m.runtimes[id]; ok {
			current.definition = previous
		}
		m.mu.Unlock()
		return err
	}
	return nil
}

// UpdateSavedDefinition changes the forwarding parameters for a saved tunnel
// only while it is disconnected. Runtime identity and creation time remain
// manager-owned so an edit cannot replace a running tunnel's identity.
func (m *Manager) UpdateSavedDefinition(id string, definition model.TunnelDefinition) error {
	m.mu.Lock()
	runtime, exists := m.runtimes[id]
	if !exists || runtime.temporary {
		m.mu.Unlock()
		return ErrTunnelNotFound
	}
	if runtime.state != model.StateDisconnected {
		m.mu.Unlock()
		return ErrTunnelRunning
	}
	updated := definition
	updated.ID = id
	updated.CreatedAt = runtime.definition.CreatedAt
	updated.LastConnectedAt = runtime.definition.LastConnectedAt
	updated.UpdatedAt = m.now()
	if err := updated.Validate(); err != nil {
		m.mu.Unlock()
		return err
	}
	previous := runtime.definition
	runtime.definition = updated
	m.mu.Unlock()
	if err := m.persistDefinitions(); err != nil {
		m.mu.Lock()
		if current, ok := m.runtimes[id]; ok {
			current.definition = previous
		}
		m.mu.Unlock()
		return err
	}
	return nil
}

// Delete removes a disconnected saved tunnel after the UI has confirmed the
// action. Temporary tunnels are removed by Disconnect instead.
func (m *Manager) Delete(id string) error {
	m.mu.Lock()
	runtime, exists := m.runtimes[id]
	if !exists || runtime.temporary {
		m.mu.Unlock()
		return ErrTunnelNotFound
	}
	if runtime.state != model.StateDisconnected {
		m.mu.Unlock()
		return ErrTunnelRunning
	}
	delete(m.runtimes, id)
	for index, savedID := range m.savedOrder {
		if savedID == id {
			m.savedOrder = append(m.savedOrder[:index], m.savedOrder[index+1:]...)
			break
		}
	}
	m.mu.Unlock()
	return m.persistDefinitions()
}

func (m *Manager) Snapshot(id string) (model.TunnelRuntime, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	runtime, exists := m.runtimes[id]
	if !exists {
		return model.TunnelRuntime{}, false
	}
	return snapshotOf(id, runtime), true
}

// Snapshots returns saved runtimes in persistence order followed by temporary
// runtimes in stable ID order for deterministic native UI rendering.
func (m *Manager) Snapshots() []model.TunnelRuntime {
	m.mu.Lock()
	defer m.mu.Unlock()

	snapshots := make([]model.TunnelRuntime, 0, len(m.runtimes))
	for _, id := range m.savedOrder {
		if runtime, exists := m.runtimes[id]; exists && !runtime.temporary {
			snapshots = append(snapshots, snapshotOf(id, runtime))
		}
	}
	temporaryIDs := make([]string, 0)
	for id, runtime := range m.runtimes {
		if runtime.temporary {
			temporaryIDs = append(temporaryIDs, id)
		}
	}
	sort.Strings(temporaryIDs)
	for _, id := range temporaryIDs {
		snapshots = append(snapshots, snapshotOf(id, m.runtimes[id]))
	}
	return snapshots
}

func snapshotOf(id string, runtime *managedRuntime) model.TunnelRuntime {
	return model.TunnelRuntime{ID: id, Definition: runtime.definition, Temporary: runtime.temporary, State: runtime.state, DesiredConnection: runtime.desired, LastError: runtime.lastError, LogLines: runtime.log.Lines()}
}

func (m *Manager) persistDefinitions() error {
	m.mu.Lock()
	definitions := make([]model.TunnelDefinition, 0, len(m.savedOrder))
	for _, id := range m.savedOrder {
		if runtime, exists := m.runtimes[id]; exists && !runtime.temporary {
			definitions = append(definitions, runtime.definition)
		}
	}
	m.mu.Unlock()
	return m.repository.ReplaceAll(definitions)
}

func endpointFor(definition model.TunnelDefinition) Endpoint {
	return Endpoint{Address: definition.LocalAddress, Port: definition.LocalPort}
}
