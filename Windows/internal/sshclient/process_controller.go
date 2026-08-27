package sshclient

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hellojinjie/TunnelDock/Windows/internal/sshconfig"
)

var (
	ErrProcessExited     = errors.New("SSH process exited during startup")
	ErrReadinessTimedOut = errors.New("SSH local listener did not become ready before timeout")
)

type ProcessEventKind int

const (
	ProcessEventStdout ProcessEventKind = iota
	ProcessEventStderr
	ProcessEventReadError
	ProcessEventExited
)

type ProcessEvent struct {
	Kind     ProcessEventKind
	Data     string
	ExitCode int
	Err      error
}

type ProcessStartRequest struct {
	RuntimeID      string
	HostAlias      string
	ForwardSpec    string
	ExpandedConfig []sshconfig.ExpandedLine
	SSHExecutable  string
}

type RuntimeConfigStore interface {
	Create(runtimeID string, lines []sshconfig.ExpandedLine) (string, error)
	Remove(runtimeID string) error
}

type JobAssigner interface {
	Assign(process *os.Process) error
}

type LaunchedProcess interface {
	Stdout() io.ReadCloser
	Stderr() io.ReadCloser
	Process() *os.Process
	Wait() (exitCode int, err error)
	Kill() error
}

type ProcessLauncher interface {
	Start(ctx context.Context, command Command) (LaunchedProcess, error)
}

type ProcessController struct {
	runtimeStore RuntimeConfigStore
	launcher     ProcessLauncher
	job          JobAssigner
}

func NewProcessController(runtimeStore RuntimeConfigStore, launcher ProcessLauncher, job JobAssigner) *ProcessController {
	return &ProcessController{runtimeStore: runtimeStore, launcher: launcher, job: job}
}

type ProcessHandle struct {
	RuntimeID string
	process   LaunchedProcess
	events    chan ProcessEvent
	done      chan struct{}
	running   atomic.Bool
	terminate sync.Once
}

func (c *ProcessController) Start(parent context.Context, request ProcessStartRequest) (*ProcessHandle, error) {
	runtimeConfig, err := c.runtimeStore.Create(request.RuntimeID, request.ExpandedConfig)
	if err != nil {
		return nil, err
	}
	cleanup := func() { _ = c.runtimeStore.Remove(request.RuntimeID) }

	ctx, cancel := context.WithCancel(parent)
	command := BuildTunnelCommand(request.SSHExecutable, runtimeConfig, request.ForwardSpec, request.HostAlias)
	process, err := c.launcher.Start(ctx, command)
	if err != nil {
		cancel()
		cleanup()
		return nil, fmt.Errorf("start SSH process: %w", err)
	}
	if err := c.job.Assign(process.Process()); err != nil {
		cancel()
		_ = process.Kill()
		_, _ = process.Wait()
		cleanup()
		return nil, fmt.Errorf("assign SSH process to Job Object: %w", err)
	}

	handle := &ProcessHandle{
		RuntimeID: request.RuntimeID,
		process:   process,
		events:    make(chan ProcessEvent, 1024),
		done:      make(chan struct{}),
	}
	handle.running.Store(true)
	go c.collect(handle, cancel)
	go func() {
		select {
		case <-parent.Done():
			_ = handle.Terminate()
		case <-handle.done:
		}
	}()
	return handle, nil
}

func (c *ProcessController) collect(handle *ProcessHandle, cancel context.CancelFunc) {
	var readers sync.WaitGroup
	readers.Add(2)
	go c.scan(handle.process.Stdout(), ProcessEventStdout, handle.events, &readers)
	go c.scan(handle.process.Stderr(), ProcessEventStderr, handle.events, &readers)
	readers.Wait()
	exitCode, waitErr := handle.process.Wait()
	handle.running.Store(false)
	cancel()
	cleanupErr := c.runtimeStore.Remove(handle.RuntimeID)
	if cleanupErr != nil && waitErr == nil {
		waitErr = cleanupErr
	}
	handle.events <- ProcessEvent{Kind: ProcessEventExited, ExitCode: exitCode, Err: waitErr}
	close(handle.events)
	close(handle.done)
}

func (c *ProcessController) scan(reader io.ReadCloser, kind ProcessEventKind, events chan<- ProcessEvent, waitGroup *sync.WaitGroup) {
	defer waitGroup.Done()
	defer reader.Close()
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		events <- ProcessEvent{Kind: kind, Data: scanner.Text()}
	}
	if err := scanner.Err(); err != nil {
		events <- ProcessEvent{Kind: ProcessEventReadError, Err: err}
	}
}

func (c *ProcessController) WaitUntilReady(ctx context.Context, handle *ProcessHandle, address string, port uint16, timeout time.Duration) error {
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	observedOccupied := false
	for {
		if !handle.IsRunning() {
			return ErrProcessExited
		}
		listener, err := net.Listen("tcp", net.JoinHostPort(address, strconv.Itoa(int(port))))
		if err != nil {
			if observedOccupied {
				if !handle.IsRunning() {
					return ErrProcessExited
				}
				return nil
			}
			observedOccupied = true
		} else {
			_ = listener.Close()
			observedOccupied = false
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-handle.done:
			return ErrProcessExited
		case <-deadline.C:
			return ErrReadinessTimedOut
		case <-ticker.C:
		}
	}
}

func (h *ProcessHandle) Events() <-chan ProcessEvent { return h.events }
func (h *ProcessHandle) IsRunning() bool             { return h.running.Load() }
func (h *ProcessHandle) Done() <-chan struct{}       { return h.done }

func (h *ProcessHandle) Terminate() error {
	var err error
	h.terminate.Do(func() { err = h.process.Kill() })
	return err
}
