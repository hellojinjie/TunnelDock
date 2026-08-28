package ui

import (
	"github.com/tailscale/walk"
	"github.com/tailscale/win"
)

type windowLayoutPolicy struct {
	WindowWidth    int
	SidebarWidth   int
	ToolbarHeight  int
	HeaderHeight   int
	IconButtonSize int
	RowAlignment   walk.Alignment2D
}

func defaultWindowLayoutPolicy() windowLayoutPolicy {
	return windowLayoutPolicy{
		WindowWidth: 1120, SidebarWidth: 240, ToolbarHeight: 30,
		HeaderHeight: 44, IconButtonSize: 30, RowAlignment: walk.AlignHNearVNear,
	}
}

type detailVisibility struct {
	QuickForward bool
	EmptyTunnels bool
	TunnelScroll bool
	Advanced     bool
}

func detailVisibilityFor(allTunnels bool, tunnelCount int, advanced bool) detailVisibility {
	return detailVisibility{
		QuickForward: !allTunnels,
		EmptyTunnels: tunnelCount == 0,
		TunnelScroll: tunnelCount > 0,
		Advanced:     advanced,
	}
}

// setChildVisible updates WS_VISIBLE directly. walk.WindowBase.SetVisible uses
// effective visibility, so hiding a child while its top-level parent is still
// hidden otherwise becomes a no-op.
func setChildVisible(window walk.Window, visible bool) {
	command := int32(win.SW_HIDE)
	if visible {
		command = win.SW_SHOWNA
	}
	win.ShowWindow(window.Handle(), command)
}
