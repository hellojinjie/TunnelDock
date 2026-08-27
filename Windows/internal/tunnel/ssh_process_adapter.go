package tunnel

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/hellojinjie/TunnelDock/Windows/internal/model"
	"github.com/hellojinjie/TunnelDock/Windows/internal/sshclient"
	"github.com/hellojinjie/TunnelDock/Windows/internal/sshconfig"
)

type sshProcessController interface {
	Start(context.Context, sshclient.ProcessStartRequest) (*sshclient.ProcessHandle, error)
	WaitUntilReady(context.Context, *sshclient.ProcessHandle, string, uint16, time.Duration) error
}

// SSHProcessAdapter makes the Windows OpenSSH controller available to
// TunnelManager without exposing process details to application or UI code.
type SSHProcessAdapter struct {
	mu         sync.RWMutex
	controller sshProcessController
	executable string
	config     []sshconfig.ExpandedLine
}

func NewSSHProcessAdapter(controller sshProcessController, executable string, config []sshconfig.ExpandedLine) *SSHProcessAdapter {
	return &SSHProcessAdapter{controller: controller, executable: executable, config: append([]sshconfig.ExpandedLine(nil), config...)}
}

func (a *SSHProcessAdapter) SetExpandedConfig(config []sshconfig.ExpandedLine) {
	a.mu.Lock()
	a.config = append([]sshconfig.ExpandedLine(nil), config...)
	a.mu.Unlock()
}

func (a *SSHProcessAdapter) Start(ctx context.Context, runtimeID string, definition model.TunnelDefinition) (ManagedProcess, error) {
	forwardSpec, err := FormatForwardSpec(definition.LocalAddress, definition.LocalPort, definition.RemoteHost, definition.RemotePort)
	if err != nil {
		return nil, err
	}
	a.mu.RLock()
	config := append([]sshconfig.ExpandedLine(nil), a.config...)
	executable := a.executable
	a.mu.RUnlock()
	if executable == "" {
		return nil, fmt.Errorf("OpenSSH executable is unavailable")
	}
	return a.controller.Start(ctx, sshclient.ProcessStartRequest{
		RuntimeID: runtimeID, HostAlias: definition.HostAlias, ForwardSpec: forwardSpec,
		ExpandedConfig: config, SSHExecutable: executable,
	})
}

func (a *SSHProcessAdapter) WaitUntilReady(ctx context.Context, process ManagedProcess, address string, port uint16, timeout time.Duration) error {
	handle, ok := process.(*sshclient.ProcessHandle)
	if !ok {
		return fmt.Errorf("unexpected managed SSH process type %T", process)
	}
	return a.controller.WaitUntilReady(ctx, handle, address, port, timeout)
}
