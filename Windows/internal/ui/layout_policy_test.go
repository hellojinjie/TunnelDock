package ui

import (
	"testing"

	"github.com/tailscale/walk"
)

func TestDefaultWindowLayoutPolicyBoundsFixedChrome(t *testing.T) {
	policy := defaultWindowLayoutPolicy()
	if policy.WindowWidth != 1120 {
		t.Fatalf("window width = %d, want 1120", policy.WindowWidth)
	}
	if policy.SidebarWidth != 240 {
		t.Fatalf("sidebar width = %d, want 240", policy.SidebarWidth)
	}
	if policy.ToolbarHeight != 30 {
		t.Fatalf("toolbar height = %d, want 30", policy.ToolbarHeight)
	}
	if policy.HeaderHeight != 44 {
		t.Fatalf("header height = %d, want 44", policy.HeaderHeight)
	}
	if policy.IconButtonSize != 30 {
		t.Fatalf("icon button size = %d, want 30", policy.IconButtonSize)
	}
	if policy.RowAlignment != walk.AlignHNearVNear {
		t.Fatalf("row alignment = %v, want top-left", policy.RowAlignment)
	}
}

func TestDetailVisibilityForAllTunnelsAndEmptyList(t *testing.T) {
	visibility := detailVisibilityFor(true, 0, false)
	if visibility.QuickForward || visibility.TunnelScroll || visibility.Advanced {
		t.Fatalf("unexpected visible detail controls: %#v", visibility)
	}
	if !visibility.EmptyTunnels {
		t.Fatalf("empty message hidden: %#v", visibility)
	}
}

func TestDetailVisibilityForSelectedHostWithRowsAndAdvanced(t *testing.T) {
	visibility := detailVisibilityFor(false, 2, true)
	if !visibility.QuickForward || !visibility.TunnelScroll || !visibility.Advanced {
		t.Fatalf("expected visible detail controls: %#v", visibility)
	}
	if visibility.EmptyTunnels {
		t.Fatalf("empty message visible: %#v", visibility)
	}
}
