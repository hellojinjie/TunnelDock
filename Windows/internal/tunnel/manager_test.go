package tunnel

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/hellojinjie/TunnelDock/Windows/internal/model"
	"github.com/hellojinjie/TunnelDock/Windows/internal/sshclient"
)

func TestManagerInitialFailureDoesNotReconnect(t *testing.T) {
	fixture := newManagerFixture(t, []model.TunnelDefinition{managerDefinition("saved-1", "gpu", 8888)}, []error{errors.New("not ready")})
	err := fixture.manager.ConnectSaved(context.Background(), "saved-1")
	if err == nil {
		t.Fatal("ConnectSaved() error = nil")
	}
	snapshot, _ := fixture.manager.Snapshot("saved-1")
	if snapshot.State != model.StateFailed || snapshot.DesiredConnection {
		t.Fatalf("snapshot = %#v", snapshot)
	}
	if fixture.controller.startCount() != 1 || fixture.clock.sleepCount() != 0 {
		t.Fatalf("starts = %d, sleeps = %d", fixture.controller.startCount(), fixture.clock.sleepCount())
	}
}

func TestManagerEstablishedExitReconnectsWithBackoffAndResets(t *testing.T) {
	fixture := newManagerFixture(t, []model.TunnelDefinition{managerDefinition("saved-1", "gpu", 8888)}, []error{
		nil,
		errors.New("retry 1"), errors.New("retry 2"), errors.New("retry 3"),
		errors.New("retry 4"), errors.New("retry 5"),
		nil, nil,
	})
	if err := fixture.manager.ConnectSaved(context.Background(), "saved-1"); err != nil {
		t.Fatal(err)
	}
	fixture.controller.process(0).exit(255)

	for _, want := range []time.Duration{time.Second, 2 * time.Second, 5 * time.Second, 10 * time.Second, 30 * time.Second, 30 * time.Second} {
		if got := fixture.clock.nextSleep(t); got != want {
			t.Fatalf("retry delay = %v, want %v", got, want)
		}
		fixture.clock.advance()
	}
	eventually(t, func() bool {
		snapshot, _ := fixture.manager.Snapshot("saved-1")
		return snapshot.State == model.StateConnected && fixture.controller.startCount() == 7
	})

	fixture.controller.process(6).exit(255)
	if got := fixture.clock.nextSleep(t); got != time.Second {
		t.Fatalf("delay after successful reconnect = %v, want 1s", got)
	}
	fixture.clock.advance()
}

func TestManagerDisconnectCancelsReconnectAndReleasesEndpoint(t *testing.T) {
	fixture := newManagerFixture(t, []model.TunnelDefinition{managerDefinition("saved-1", "gpu", 8888)}, []error{nil, nil})
	if err := fixture.manager.ConnectSaved(context.Background(), "saved-1"); err != nil {
		t.Fatal(err)
	}
	fixture.controller.process(0).exit(255)
	if got := fixture.clock.nextSleep(t); got != time.Second {
		t.Fatalf("delay = %v", got)
	}
	if err := fixture.manager.Disconnect("saved-1"); err != nil {
		t.Fatal(err)
	}
	fixture.clock.advance()
	time.Sleep(20 * time.Millisecond)
	if fixture.controller.startCount() != 1 {
		t.Fatalf("starts = %d, want 1", fixture.controller.startCount())
	}
	snapshot, _ := fixture.manager.Snapshot("saved-1")
	if snapshot.State != model.StateDisconnected || snapshot.DesiredConnection {
		t.Fatalf("snapshot = %#v", snapshot)
	}
	if fixture.ports.reservationCount() != 0 {
		t.Fatalf("reservations = %d", fixture.ports.reservationCount())
	}
}

func TestManagerMissingHostKeepsLiveTunnelButPreventsReconnect(t *testing.T) {
	fixture := newManagerFixture(t, []model.TunnelDefinition{managerDefinition("saved-1", "gpu", 8888)}, []error{nil, nil})
	if err := fixture.manager.ConnectSaved(context.Background(), "saved-1"); err != nil {
		t.Fatal(err)
	}
	fixture.manager.UpdateHosts(nil)
	snapshot, _ := fixture.manager.Snapshot("saved-1")
	if snapshot.State != model.StateConnected {
		t.Fatalf("live state = %v", snapshot.State)
	}
	fixture.controller.process(0).exit(255)
	eventually(t, func() bool { value, _ := fixture.manager.Snapshot("saved-1"); return value.State == model.StateFailed })
	if fixture.clock.sleepCount() != 0 {
		t.Fatalf("missing host scheduled retry")
	}
	if err := fixture.manager.ConnectSaved(context.Background(), "saved-1"); !errors.Is(err, ErrHostUnavailable) {
		t.Fatalf("missing host error = %v", err)
	}
	fixture.manager.UpdateHosts([]model.SSHHost{{Alias: "gpu", Availability: model.HostAvailable}})
	if err := fixture.manager.ConnectSaved(context.Background(), "saved-1"); err != nil {
		t.Fatal(err)
	}
}

func TestManagerTemporarySaveAndDisconnectBehavior(t *testing.T) {
	fixture := newManagerFixture(t, nil, []error{nil, nil})
	definition := managerDefinition("", "gpu", 9000)
	id, err := fixture.manager.ConnectTemporary(context.Background(), definition)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, ok := fixture.manager.Snapshot(id)
	if !ok || !snapshot.Temporary || snapshot.State != model.StateConnected {
		t.Fatalf("temporary = %#v, %v", snapshot, ok)
	}
	saved, err := fixture.manager.SaveTemporary(id)
	if err != nil {
		t.Fatal(err)
	}
	if saved.ID != id {
		t.Fatalf("saved ID = %q, want %q", saved.ID, id)
	}
	snapshot, _ = fixture.manager.Snapshot(id)
	if snapshot.Temporary {
		t.Fatal("saved runtime remained temporary")
	}
	if len(fixture.repository.snapshot()) != 1 {
		t.Fatalf("saved definitions = %#v", fixture.repository.snapshot())
	}
	if err := fixture.manager.ConnectSaved(context.Background(), id); err != nil {
		t.Fatal(err)
	}
	if fixture.controller.startCount() != 1 {
		t.Fatalf("double connect launched %d processes", fixture.controller.startCount())
	}
	if err := fixture.manager.Disconnect(id); err != nil {
		t.Fatal(err)
	}
	if _, ok := fixture.manager.Snapshot(id); !ok {
		t.Fatal("saved runtime disappeared after disconnect")
	}

	temporaryID, err := fixture.manager.ConnectTemporary(context.Background(), managerDefinition("", "gpu", 9001))
	if err != nil {
		t.Fatal(err)
	}
	if err := fixture.manager.Disconnect(temporaryID); err != nil {
		t.Fatal(err)
	}
	if _, ok := fixture.manager.Snapshot(temporaryID); ok {
		t.Fatal("temporary runtime remained after disconnect")
	}
}

func TestManagerSnapshotsListsSavedBeforeTemporary(t *testing.T) {
	fixture := newManagerFixture(t, []model.TunnelDefinition{managerDefinition("saved-1", "gpu", 9000)}, []error{nil})
	temporaryID, err := fixture.manager.ConnectTemporary(context.Background(), managerDefinition("", "gpu", 9001))
	if err != nil {
		t.Fatal(err)
	}

	snapshots := fixture.manager.Snapshots()
	if len(snapshots) != 2 || snapshots[0].ID != "saved-1" || snapshots[0].Temporary || snapshots[1].ID != temporaryID || !snapshots[1].Temporary {
		t.Fatalf("Snapshots() = %#v", snapshots)
	}
}

func TestManagerRenameAllowsRunningSavedTunnelAndDeleteRejectsIt(t *testing.T) {
	fixture := newManagerFixture(t, []model.TunnelDefinition{managerDefinition("saved-1", "gpu", 9000)}, []error{nil})
	if err := fixture.manager.ConnectSaved(context.Background(), "saved-1"); err != nil {
		t.Fatal(err)
	}
	if err := fixture.manager.Rename("saved-1", "Jupyter"); err != nil {
		t.Fatalf("Rename() error: %v", err)
	}
	snapshot, _ := fixture.manager.Snapshot("saved-1")
	if snapshot.Definition.Name == nil || *snapshot.Definition.Name != "Jupyter" {
		t.Fatalf("renamed snapshot = %#v", snapshot)
	}
	if err := fixture.manager.Delete("saved-1"); !errors.Is(err, ErrTunnelRunning) {
		t.Fatalf("Delete() error = %v, want ErrTunnelRunning", err)
	}
	if err := fixture.manager.Disconnect("saved-1"); err != nil {
		t.Fatal(err)
	}
	if err := fixture.manager.Delete("saved-1"); err != nil {
		t.Fatalf("Delete() error: %v", err)
	}
	if _, exists := fixture.manager.Snapshot("saved-1"); exists {
		t.Fatal("deleted saved runtime remained")
	}
}

func TestManagerShutdownDisconnectsRunningTunnelsAndRejectsNewConnections(t *testing.T) {
	fixture := newManagerFixture(t, []model.TunnelDefinition{managerDefinition("saved-1", "gpu", 9000)}, []error{nil})
	if err := fixture.manager.ConnectSaved(context.Background(), "saved-1"); err != nil {
		t.Fatal(err)
	}
	if err := fixture.manager.Shutdown(); err != nil {
		t.Fatal(err)
	}
	snapshot, _ := fixture.manager.Snapshot("saved-1")
	if snapshot.State != model.StateDisconnected || snapshot.DesiredConnection {
		t.Fatalf("snapshot = %#v", snapshot)
	}
	if err := fixture.manager.ConnectSaved(context.Background(), "saved-1"); !errors.Is(err, ErrApplicationShuttingDown) {
		t.Fatalf("ConnectSaved() error = %v", err)
	}
}

func TestManagerMultipleTunnelsAreIndependentAndStderrDoesNotFailTransport(t *testing.T) {
	definitions := []model.TunnelDefinition{managerDefinition("one", "gpu", 9101), managerDefinition("two", "gpu", 9102)}
	fixture := newManagerFixture(t, definitions, []error{nil, nil})
	if err := fixture.manager.ConnectSaved(context.Background(), "one"); err != nil {
		t.Fatal(err)
	}
	if err := fixture.manager.ConnectSaved(context.Background(), "two"); err != nil {
		t.Fatal(err)
	}
	fixture.controller.process(0).stderr("connect failed: Connection refused")
	eventually(t, func() bool {
		value, _ := fixture.manager.Snapshot("one")
		return len(value.LogLines) > 0
	})
	one, _ := fixture.manager.Snapshot("one")
	two, _ := fixture.manager.Snapshot("two")
	if one.State != model.StateConnected || two.State != model.StateConnected || fixture.controller.startCount() != 2 {
		t.Fatalf("one=%#v two=%#v starts=%d", one, two, fixture.controller.startCount())
	}
	for _, definition := range fixture.repository.snapshot() {
		if definition.LastConnectedAt == nil {
			t.Fatalf("lastConnectedAt was not persisted for %q", definition.ID)
		}
	}
}

type managerFixture struct {
	manager    *Manager
	repository *fakeDefinitionRepository
	controller *fakeManagerController
	clock      *manualManagerClock
	ports      *fakeManagerPorts
}

func newManagerFixture(t *testing.T, definitions []model.TunnelDefinition, readiness []error) managerFixture {
	t.Helper()
	repository := &fakeDefinitionRepository{definitions: append([]model.TunnelDefinition(nil), definitions...)}
	controller := &fakeManagerController{readiness: append([]error(nil), readiness...)}
	clock := newManualManagerClock()
	ports := &fakeManagerPorts{reserved: make(map[Endpoint]struct{})}
	manager := NewManager(ManagerOptions{Repository: repository, Ports: ports, Processes: controller, Clock: clock, Now: func() time.Time { return time.Date(2026, 8, 27, 12, 0, 0, 0, time.UTC) }})
	manager.UpdateHosts([]model.SSHHost{{Alias: "gpu", Availability: model.HostAvailable}})
	if err := manager.LoadSavedDefinitions(); err != nil {
		t.Fatal(err)
	}
	return managerFixture{manager: manager, repository: repository, controller: controller, clock: clock, ports: ports}
}

func managerDefinition(id, alias string, port uint16) model.TunnelDefinition {
	now := time.Date(2026, 8, 27, 11, 0, 0, 0, time.UTC)
	return model.TunnelDefinition{ID: id, HostAlias: alias, RemoteHost: "127.0.0.1", RemotePort: port, LocalAddress: "127.0.0.1", LocalPort: port, WebProtocol: model.TunnelProtocolHTTP, CreatedAt: now, UpdatedAt: now}
}

func eventually(t *testing.T, predicate func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if predicate() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("condition not reached")
}

type fakeDefinitionRepository struct {
	mu          sync.Mutex
	definitions []model.TunnelDefinition
}

func (r *fakeDefinitionRepository) Load() ([]model.TunnelDefinition, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]model.TunnelDefinition(nil), r.definitions...), nil
}
func (r *fakeDefinitionRepository) ReplaceAll(values []model.TunnelDefinition) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.definitions = append([]model.TunnelDefinition(nil), values...)
	return nil
}
func (r *fakeDefinitionRepository) snapshot() []model.TunnelDefinition {
	values, _ := r.Load()
	return values
}

type fakeManagerPorts struct {
	mu       sync.Mutex
	reserved map[Endpoint]struct{}
}

func (p *fakeManagerPorts) Check(endpoint Endpoint) PortStatus {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, ok := p.reserved[endpoint]; ok {
		return PortUsedByTunnelDock
	}
	return PortAvailable
}
func (p *fakeManagerPorts) Reserve(endpoint Endpoint) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, ok := p.reserved[endpoint]; ok {
		return false
	}
	p.reserved[endpoint] = struct{}{}
	return true
}
func (p *fakeManagerPorts) Release(endpoint Endpoint) {
	p.mu.Lock()
	delete(p.reserved, endpoint)
	p.mu.Unlock()
}
func (p *fakeManagerPorts) reservationCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.reserved)
}

type fakeManagerController struct {
	mu        sync.Mutex
	readiness []error
	processes []*fakeManagerProcess
}

func (c *fakeManagerController) Start(context.Context, string, model.TunnelDefinition) (ManagedProcess, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	process := newFakeManagerProcess()
	c.processes = append(c.processes, process)
	return process, nil
}
func (c *fakeManagerController) WaitUntilReady(context.Context, ManagedProcess, string, uint16, time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.readiness) == 0 {
		return nil
	}
	value := c.readiness[0]
	c.readiness = c.readiness[1:]
	return value
}
func (c *fakeManagerController) startCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.processes)
}
func (c *fakeManagerController) process(index int) *fakeManagerProcess {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.processes[index]
}

type fakeManagerProcess struct {
	mu      sync.Mutex
	running bool
	events  chan sshclient.ProcessEvent
	once    sync.Once
}

func newFakeManagerProcess() *fakeManagerProcess {
	return &fakeManagerProcess{running: true, events: make(chan sshclient.ProcessEvent, 16)}
}
func (p *fakeManagerProcess) Events() <-chan sshclient.ProcessEvent { return p.events }
func (p *fakeManagerProcess) IsRunning() bool                       { p.mu.Lock(); defer p.mu.Unlock(); return p.running }
func (p *fakeManagerProcess) Terminate() error                      { p.exit(0); return nil }
func (p *fakeManagerProcess) exit(code int) {
	p.once.Do(func() {
		p.mu.Lock()
		p.running = false
		p.mu.Unlock()
		p.events <- sshclient.ProcessEvent{Kind: sshclient.ProcessEventExited, ExitCode: code}
		close(p.events)
	})
}
func (p *fakeManagerProcess) stderr(value string) {
	p.events <- sshclient.ProcessEvent{Kind: sshclient.ProcessEventStderr, Data: value}
}

type sleepRequest struct {
	duration time.Duration
	release  chan struct{}
}
type manualManagerClock struct {
	requests chan sleepRequest
	mu       sync.Mutex
	count    int
	last     sleepRequest
}

func newManualManagerClock() *manualManagerClock {
	return &manualManagerClock{requests: make(chan sleepRequest, 16)}
}
func (c *manualManagerClock) Sleep(ctx context.Context, duration time.Duration) error {
	request := sleepRequest{duration: duration, release: make(chan struct{})}
	c.mu.Lock()
	c.count++
	c.mu.Unlock()
	c.requests <- request
	select {
	case <-request.release:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
func (c *manualManagerClock) nextSleep(t *testing.T) time.Duration {
	t.Helper()
	select {
	case request := <-c.requests:
		c.mu.Lock()
		c.last = request
		c.mu.Unlock()
		return request.duration
	case <-time.After(2 * time.Second):
		t.Fatal("no retry sleep")
		return 0
	}
}
func (c *manualManagerClock) advance() {
	c.mu.Lock()
	request := c.last
	c.last = sleepRequest{}
	c.mu.Unlock()
	if request.release != nil {
		close(request.release)
	}
}
func (c *manualManagerClock) sleepCount() int { c.mu.Lock(); defer c.mu.Unlock(); return c.count }
