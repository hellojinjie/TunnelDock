package tunnel

import (
	"context"
	"testing"
	"time"

	"github.com/hellojinjie/TunnelDock/Windows/internal/model"
	"github.com/hellojinjie/TunnelDock/Windows/internal/sshclient"
	"github.com/hellojinjie/TunnelDock/Windows/internal/sshconfig"
)

type recordingSSHProcessController struct {
	request sshclient.ProcessStartRequest
}

func (c *recordingSSHProcessController) Start(_ context.Context, request sshclient.ProcessStartRequest) (*sshclient.ProcessHandle, error) {
	c.request = request
	return &sshclient.ProcessHandle{}, nil
}

func (c *recordingSSHProcessController) WaitUntilReady(context.Context, *sshclient.ProcessHandle, string, uint16, time.Duration) error {
	return nil
}

func TestSSHProcessAdapterBuildsIsolatedForwardRequest(t *testing.T) {
	controller := &recordingSSHProcessController{}
	adapter := NewSSHProcessAdapter(controller, "C:\\Windows\\System32\\OpenSSH\\ssh.exe", []sshconfig.ExpandedLine{{Text: "Host build"}})

	_, err := adapter.Start(context.Background(), "runtime-1", model.TunnelDefinition{
		HostAlias: "build", LocalAddress: "127.0.0.1", LocalPort: 18888,
		RemoteHost: "127.0.0.1", RemotePort: 8888,
	})
	if err != nil {
		t.Fatalf("Start() error: %v", err)
	}
	if controller.request.RuntimeID != "runtime-1" || controller.request.HostAlias != "build" || controller.request.ForwardSpec != "127.0.0.1:18888:127.0.0.1:8888" || controller.request.SSHExecutable == "" || len(controller.request.ExpandedConfig) != 1 {
		t.Fatalf("request = %#v", controller.request)
	}
}
