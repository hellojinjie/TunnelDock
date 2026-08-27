package app

import (
	"testing"

	"github.com/hellojinjie/TunnelDock/Windows/internal/model"
)

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
