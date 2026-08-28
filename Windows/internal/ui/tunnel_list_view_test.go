package ui

import (
	"testing"

	"github.com/hellojinjie/TunnelDock/Windows/internal/model"
)

func TestTunnelRowCallbackPolicyForTemporaryTunnel(t *testing.T) {
	row := TunnelRowPresentation{
		ID:            "temporary",
		Temporary:     true,
		State:         model.StateConnected,
		PrimaryAction: TunnelRowDisconnect,
		ShowBrowser:   true,
		ShowMore:      true,
	}
	menu := MoreMenuItems(row)
	if !menu.Enabled(TunnelMenuLog) || !menu.Enabled(TunnelMenuSave) {
		t.Fatalf("temporary menu = %#v", menu)
	}
	if menu.Contains(TunnelMenuRename) || menu.Contains(TunnelMenuEdit) || menu.Contains(TunnelMenuDelete) {
		t.Fatalf("temporary menu exposes saved-tunnel actions: %#v", menu)
	}
}

func TestTunnelRowCallbackPolicyDisablesMutationWhileRunning(t *testing.T) {
	row := TunnelRowPresentation{ID: "saved", State: model.StateConnected, ShowMore: true}
	menu := MoreMenuItems(row)
	if !menu.Enabled(TunnelMenuLog) || !menu.Enabled(TunnelMenuRename) {
		t.Fatalf("running saved menu = %#v", menu)
	}
	if !menu.Contains(TunnelMenuEdit) || menu.Enabled(TunnelMenuEdit) {
		t.Fatalf("edit policy = %#v", menu)
	}
	if !menu.Contains(TunnelMenuDelete) || menu.Enabled(TunnelMenuDelete) {
		t.Fatalf("delete policy = %#v", menu)
	}
}

func TestTunnelRowCallbackPolicyAllowsMutationWhenStopped(t *testing.T) {
	row := TunnelRowPresentation{ID: "saved", State: model.StateDisconnected, ShowMore: true}
	menu := MoreMenuItems(row)
	for _, item := range []TunnelMenuItem{TunnelMenuLog, TunnelMenuRename, TunnelMenuEdit, TunnelMenuDelete} {
		if !menu.Enabled(item) {
			t.Fatalf("item %v disabled in %#v", item, menu)
		}
	}
}
