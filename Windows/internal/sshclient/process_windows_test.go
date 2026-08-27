package sshclient

import (
	"context"
	"errors"
	"io"
	"net"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hellojinjie/TunnelDock/Windows/internal/sshconfig"
)

func TestProcessControllerForwardsOutputAndCleansAfterExit(t *testing.T) {
	process := newFakeLaunchedProcess("first\nsecond\n", "warning\n")
	process.finish(23, errors.New("exit status 23"))
	runtimeStore := &fakeRuntimeConfigStore{path: `C:\runtime\run-1\ssh_config`}
	launcher := &fakeLauncher{process: process}
	job := &fakeJobAssigner{}
	controller := NewProcessController(runtimeStore, launcher, job)

	handle, err := controller.Start(context.Background(), ProcessStartRequest{
		RuntimeID: "run-1", HostAlias: "gpu", ForwardSpec: "127.0.0.1:8888:127.0.0.1:8888",
		ExpandedConfig: []sshconfig.ExpandedLine{{Text: "Host gpu"}}, SSHExecutable: "ssh.exe",
	})
	if err != nil {
		t.Fatalf("Start() error: %v", err)
	}
	events := collectEvents(t, handle.Events())
	if got := eventData(events, ProcessEventStdout); !reflect.DeepEqual(got, []string{"first", "second"}) {
		t.Fatalf("stdout events = %#v", got)
	}
	if got := eventData(events, ProcessEventStderr); !reflect.DeepEqual(got, []string{"warning"}) {
		t.Fatalf("stderr events = %#v", got)
	}
	last := events[len(events)-1]
	if last.Kind != ProcessEventExited || last.ExitCode != 23 {
		t.Fatalf("last event = %#v", last)
	}
	if !reflect.DeepEqual(runtimeStore.removed, []string{"run-1"}) || job.assignCount != 1 {
		t.Fatalf("cleanup = %#v, job assignments = %d", runtimeStore.removed, job.assignCount)
	}
	if handle.IsRunning() {
		t.Fatal("handle remained running after exit")
	}
}

func TestProcessControllerEarlyExitStillCleansRuntime(t *testing.T) {
	process := newFakeLaunchedProcess("", "fatal\n")
	process.finish(255, errors.New("exit status 255"))
	store := &fakeRuntimeConfigStore{path: `C:\runtime\early\ssh_config`}
	handle, err := NewProcessController(store, &fakeLauncher{process: process}, &fakeJobAssigner{}).Start(
		context.Background(), ProcessStartRequest{RuntimeID: "early", HostAlias: "bad", ForwardSpec: "127.0.0.1:1:127.0.0.1:1", SSHExecutable: "ssh.exe"},
	)
	if err != nil {
		t.Fatal(err)
	}
	_ = collectEvents(t, handle.Events())
	if !reflect.DeepEqual(store.removed, []string{"early"}) {
		t.Fatalf("removed = %#v", store.removed)
	}
}

func TestProcessControllerCancellationTerminatesAndCleans(t *testing.T) {
	process := newFakeLaunchedProcess("", "")
	store := &fakeRuntimeConfigStore{path: `C:\runtime\cancel\ssh_config`}
	ctx, cancel := context.WithCancel(context.Background())
	handle, err := NewProcessController(store, &fakeLauncher{process: process}, &fakeJobAssigner{}).Start(
		ctx, ProcessStartRequest{RuntimeID: "cancel", HostAlias: "gpu", ForwardSpec: "127.0.0.1:1:127.0.0.1:1", SSHExecutable: "ssh.exe"},
	)
	if err != nil {
		t.Fatal(err)
	}
	cancel()
	_ = collectEvents(t, handle.Events())
	if process.killCount() != 1 || !reflect.DeepEqual(store.removed, []string{"cancel"}) {
		t.Fatalf("kills = %d, removed = %#v", process.killCount(), store.removed)
	}
}

func TestProcessControllerReadinessObservesLocalListenerWithoutConnecting(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	_, portText, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	port, err := strconv.ParseUint(portText, 10, 16)
	if err != nil {
		t.Fatal(err)
	}

	process := newFakeLaunchedProcess("", "")
	controller := NewProcessController(&fakeRuntimeConfigStore{path: `C:\runtime\ready\ssh_config`}, &fakeLauncher{process: process}, &fakeJobAssigner{})
	handle, err := controller.Start(context.Background(), ProcessStartRequest{RuntimeID: "ready", HostAlias: "gpu", ForwardSpec: "spec", SSHExecutable: "ssh.exe"})
	if err != nil {
		t.Fatal(err)
	}
	if err := controller.WaitUntilReady(context.Background(), handle, "127.0.0.1", uint16(port), time.Second); err != nil {
		t.Fatalf("WaitUntilReady() error: %v", err)
	}
	if err := handle.Terminate(); err != nil {
		t.Fatal(err)
	}
	_ = collectEvents(t, handle.Events())
}

func TestProcessControllerReadinessDoesNotMistakeOccupiedPortForEarlyExitedSSH(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	_, portText, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	port, err := strconv.ParseUint(portText, 10, 16)
	if err != nil {
		t.Fatal(err)
	}

	process := newFakeLaunchedProcess("", "bind failed\n")
	process.finish(255, errors.New("exit status 255"))
	controller := NewProcessController(&fakeRuntimeConfigStore{path: `C:\runtime\failed\ssh_config`}, &fakeLauncher{process: process}, &fakeJobAssigner{})
	handle, err := controller.Start(context.Background(), ProcessStartRequest{RuntimeID: "failed", HostAlias: "gpu", ForwardSpec: "spec", SSHExecutable: "ssh.exe"})
	if err != nil {
		t.Fatal(err)
	}
	if err := controller.WaitUntilReady(context.Background(), handle, "127.0.0.1", uint16(port), time.Second); !errors.Is(err, ErrProcessExited) {
		t.Fatalf("WaitUntilReady() error = %v, want ErrProcessExited", err)
	}
	_ = collectEvents(t, handle.Events())
}

func collectEvents(t *testing.T, events <-chan ProcessEvent) []ProcessEvent {
	t.Helper()
	var values []ProcessEvent
	timer := time.NewTimer(3 * time.Second)
	defer timer.Stop()
	for {
		select {
		case event, open := <-events:
			if !open {
				return values
			}
			values = append(values, event)
		case <-timer.C:
			t.Fatal("timed out waiting for process events")
		}
	}
}

func eventData(events []ProcessEvent, kind ProcessEventKind) []string {
	var values []string
	for _, event := range events {
		if event.Kind == kind {
			values = append(values, event.Data)
		}
	}
	return values
}

type fakeRuntimeConfigStore struct {
	path    string
	removed []string
	mu      sync.Mutex
}

func (s *fakeRuntimeConfigStore) Create(string, []sshconfig.ExpandedLine) (string, error) {
	return s.path, nil
}
func (s *fakeRuntimeConfigStore) Remove(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.removed = append(s.removed, id)
	return nil
}

type fakeJobAssigner struct{ assignCount int }

func (j *fakeJobAssigner) Assign(*os.Process) error { j.assignCount++; return nil }

type fakeLauncher struct{ process *fakeLaunchedProcess }

func (l *fakeLauncher) Start(context.Context, Command) (LaunchedProcess, error) {
	return l.process, nil
}

type fakeExit struct {
	code int
	err  error
}
type fakeLaunchedProcess struct {
	stdout io.ReadCloser
	stderr io.ReadCloser
	exit   chan fakeExit
	killed chan struct{}
	once   sync.Once
	mu     sync.Mutex
	kills  int
}

func newFakeLaunchedProcess(stdout, stderr string) *fakeLaunchedProcess {
	return &fakeLaunchedProcess{
		stdout: io.NopCloser(strings.NewReader(stdout)), stderr: io.NopCloser(strings.NewReader(stderr)),
		exit: make(chan fakeExit, 1), killed: make(chan struct{}),
	}
}
func (p *fakeLaunchedProcess) Stdout() io.ReadCloser { return p.stdout }
func (p *fakeLaunchedProcess) Stderr() io.ReadCloser { return p.stderr }
func (p *fakeLaunchedProcess) Process() *os.Process  { return nil }
func (p *fakeLaunchedProcess) Wait() (int, error) {
	select {
	case result := <-p.exit:
		return result.code, result.err
	case <-p.killed:
		return -1, context.Canceled
	}
}
func (p *fakeLaunchedProcess) Kill() error {
	p.mu.Lock()
	p.kills++
	p.mu.Unlock()
	p.once.Do(func() { close(p.killed) })
	return nil
}
func (p *fakeLaunchedProcess) finish(code int, err error) { p.exit <- fakeExit{code: code, err: err} }
func (p *fakeLaunchedProcess) killCount() int             { p.mu.Lock(); defer p.mu.Unlock(); return p.kills }
