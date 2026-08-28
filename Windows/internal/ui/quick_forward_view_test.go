package ui

import (
	"testing"

	"github.com/hellojinjie/TunnelDock/Windows/internal/app"
)

func TestPresentQuickForwardReflectsBusyAndConflict(t *testing.T) {
	quick := app.NewQuickForward()
	quick.SetRemotePort("8888")
	state := PresentQuickForward(quick, true)
	if state.ConnectText != "Connecting..." || state.ConnectEnabled {
		t.Fatalf("busy state = %#v", state)
	}
	quick.HandlePortConflict()
	state = PresentQuickForward(quick, false)
	if !state.AdvancedExpanded || state.FocusField != app.FocusLocalPort {
		t.Fatalf("conflict state = %#v", state)
	}
}

func TestPresentQuickForwardRequiresRemotePort(t *testing.T) {
	quick := app.NewQuickForward()
	state := PresentQuickForward(quick, false)
	if state.ConnectEnabled {
		t.Fatalf("empty quick forward is enabled: %#v", state)
	}
	quick.SetRemotePort("22")
	if state = PresentQuickForward(quick, false); !state.ConnectEnabled {
		t.Fatalf("ready quick forward is disabled: %#v", state)
	}
}
