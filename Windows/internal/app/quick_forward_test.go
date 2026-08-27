package app

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/hellojinjie/TunnelDock/Windows/internal/model"
)

type recordingTemporaryConnector struct {
	definition model.TunnelDefinition
	err        error
}

func (c *recordingTemporaryConnector) ConnectTemporary(_ context.Context, definition model.TunnelDefinition) (string, error) {
	c.definition = definition
	if c.err != nil {
		return "", c.err
	}
	return "temporary-1", nil
}

func TestQuickForwardDefaultsAndLocalPortFollow(t *testing.T) {
	quick := NewQuickForward()
	if quick.RemoteHost != "127.0.0.1" || quick.LocalAddress != "127.0.0.1" || quick.WebProtocol != model.TunnelProtocolHTTP || quick.AdvancedExpanded {
		t.Fatalf("defaults = %#v", quick)
	}
	quick.SetRemotePort("8888")
	if quick.RemotePort != "8888" || quick.LocalPort != "8888" {
		t.Fatalf("follow = %#v", quick)
	}
	quick.SetRemotePort("9999")
	if quick.LocalPort != "9999" {
		t.Fatalf("continued follow local = %q", quick.LocalPort)
	}
}

func TestQuickForwardTunnelDefinitionUsesFollowedLocalPort(t *testing.T) {
	quick := NewQuickForward()
	quick.SetRemotePort("8888")

	definition, err := quick.TunnelDefinition("gpu-build")
	if err != nil {
		t.Fatalf("TunnelDefinition() error: %v", err)
	}
	if definition.HostAlias != "gpu-build" || definition.RemoteHost != "127.0.0.1" || definition.RemotePort != 8888 || definition.LocalAddress != "127.0.0.1" || definition.LocalPort != 8888 || definition.WebProtocol != model.TunnelProtocolHTTP {
		t.Fatalf("TunnelDefinition() = %#v", definition)
	}
}

func TestQuickForwardTunnelDefinitionRejectsNonNumericRemotePort(t *testing.T) {
	quick := NewQuickForward()
	quick.SetRemotePort("https")

	_, err := quick.TunnelDefinition("gpu-build")
	if err == nil || !strings.Contains(err.Error(), "remote port") {
		t.Fatalf("TunnelDefinition() error = %v, want remote port validation error", err)
	}
}

func TestQuickForwardConnectPassesTemporaryDefinitionToConnector(t *testing.T) {
	quick := NewQuickForward()
	quick.SetRemotePort("8888")
	connector := &recordingTemporaryConnector{}

	id, err := quick.Connect(context.Background(), connector, "gpu-build")
	if err != nil {
		t.Fatalf("Connect() error: %v", err)
	}
	if id != "temporary-1" || connector.definition.HostAlias != "gpu-build" || connector.definition.LocalPort != 8888 {
		t.Fatalf("Connect() = %q, definition = %#v", id, connector.definition)
	}
}

func TestQuickForwardConnectDoesNotCallConnectorForInvalidForm(t *testing.T) {
	quick := NewQuickForward()
	quick.SetRemotePort("invalid")
	connector := &recordingTemporaryConnector{err: errors.New("connector must not run")}

	_, err := quick.Connect(context.Background(), connector, "gpu-build")
	if err == nil || connector.definition.HostAlias != "" {
		t.Fatalf("Connect() error = %v, connector definition = %#v", err, connector.definition)
	}
}

func TestQuickForwardManualLocalPortEditDetachesFollow(t *testing.T) {
	quick := NewQuickForward()
	quick.SetRemotePort("8888")
	quick.SetLocalPort("18888")
	quick.SetRemotePort("9999")
	if quick.LocalPort != "18888" || quick.LocalPortFollowsRemote() {
		t.Fatalf("detached quick forward = %#v", quick)
	}
}

func TestQuickForwardResetRestoresDefaultsAndFollow(t *testing.T) {
	quick := NewQuickForward()
	quick.SetRemotePort("8888")
	quick.SetLocalPort("18888")
	quick.RemoteHost = "service.internal"
	quick.LocalAddress = "0.0.0.0"
	quick.WebProtocol = model.TunnelProtocolHTTPS
	quick.HandlePortConflict()
	quick.Reset()
	if quick.RemotePort != "" || quick.LocalPort != "" || quick.RemoteHost != "127.0.0.1" || quick.LocalAddress != "127.0.0.1" || quick.WebProtocol != model.TunnelProtocolHTTP || quick.AdvancedExpanded || !quick.LocalPortFollowsRemote() {
		t.Fatalf("Reset() = %#v", quick)
	}
}

func TestQuickForwardPortConflictPreservesValuesAndRequestsLocalPortFocus(t *testing.T) {
	quick := NewQuickForward()
	quick.SetRemotePort("8888")
	quick.SetLocalPort("18888")
	quick.RemoteHost = "service.internal"
	quick.HandlePortConflict()
	if !quick.AdvancedExpanded || quick.Focus != FocusLocalPort || quick.RemotePort != "8888" || quick.LocalPort != "18888" || quick.RemoteHost != "service.internal" {
		t.Fatalf("HandlePortConflict() = %#v", quick)
	}
}
