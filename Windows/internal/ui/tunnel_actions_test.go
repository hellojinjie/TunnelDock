package ui

import (
	"testing"

	"github.com/hellojinjie/TunnelDock/Windows/internal/model"
)

func TestTunnelBrowserURLUsesLoopbackAndConfiguredProtocol(t *testing.T) {
	for _, test := range []struct {
		protocol model.TunnelProtocol
		want     string
	}{
		{model.TunnelProtocolHTTP, "http://127.0.0.1:8888"},
		{model.TunnelProtocolHTTPS, "https://127.0.0.1:9443"},
	} {
		port := uint16(8888)
		if test.protocol == model.TunnelProtocolHTTPS {
			port = 9443
		}
		if got := TunnelBrowserURL(model.TunnelDefinition{LocalPort: port, WebProtocol: test.protocol}); got != test.want {
			t.Fatalf("TunnelBrowserURL(%q) = %q, want %q", test.protocol, got, test.want)
		}
	}
}

func TestOpenBrowserDelegatesToWindowsShell(t *testing.T) {
	previous := shellOpenURL
	t.Cleanup(func() { shellOpenURL = previous })
	var got string
	shellOpenURL = func(url string) error { got = url; return nil }

	if err := OpenBrowser("https://127.0.0.1:9443"); err != nil {
		t.Fatalf("OpenBrowser() error: %v", err)
	}
	if got != "https://127.0.0.1:9443" {
		t.Fatalf("shell URL = %q", got)
	}
}

func TestCanDeleteTunnelStateMatchesRecentTunnelMenu(t *testing.T) {
	for _, test := range []struct {
		state model.TunnelState
		want  bool
	}{
		{state: model.StateDisconnected, want: true},
		{state: model.StateFailed, want: true},
		{state: model.StateConnecting, want: false},
		{state: model.StateConnected, want: false},
		{state: model.StateReconnecting, want: false},
	} {
		if got := canDeleteTunnelState(test.state); got != test.want {
			t.Fatalf("canDeleteTunnelState(%v) = %v, want %v", test.state, got, test.want)
		}
	}
}
