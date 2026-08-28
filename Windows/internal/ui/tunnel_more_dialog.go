package ui

import (
	"fmt"

	"github.com/hellojinjie/TunnelDock/Windows/internal/model"
	"github.com/tailscale/walk"
)

type tunnelMoreAction int

const (
	tunnelMoreNone tunnelMoreAction = iota
	tunnelMoreLog
	tunnelMoreSave
	tunnelMoreRename
	tunnelMoreEdit
	tunnelMoreDelete
)

// promptTunnelMore keeps infrequent tunnel management actions out of the main
// workflow while preserving all operations in one native dialog.
func promptTunnelMore(owner walk.Form, runtime model.TunnelRuntime) (tunnelMoreAction, error) {
	dialog, err := walk.NewDialogWithFixedSize(owner)
	if err != nil {
		return tunnelMoreNone, err
	}
	defer dialog.Dispose()
	if err := dialog.SetTitle("Manage Tunnel"); err != nil {
		return tunnelMoreNone, err
	}
	if err := dialog.SetSize(walk.Size{Width: 320, Height: 260}); err != nil {
		return tunnelMoreNone, err
	}
	if err := dialog.SetLayout(walk.NewVBoxLayout()); err != nil {
		return tunnelMoreNone, err
	}
	detail, err := walk.NewLabel(dialog)
	if err != nil {
		return tunnelMoreNone, err
	}
	_ = detail.SetText(fmt.Sprintf("%s\n%s:%d → %s:%d", runtime.DisplayName(), runtime.Definition.LocalAddress, runtime.Definition.LocalPort, runtime.Definition.RemoteHost, runtime.Definition.RemotePort))
	selected := tunnelMoreNone
	add := func(text string, action tunnelMoreAction) error {
		button, err := walk.NewPushButton(dialog)
		if err != nil {
			return err
		}
		_ = button.SetText(text)
		button.Clicked().Attach(func() { selected = action; dialog.Accept() })
		return nil
	}
	if err := add("View Log", tunnelMoreLog); err != nil {
		return tunnelMoreNone, err
	}
	if runtime.Temporary {
		if err := add("Save Tunnel", tunnelMoreSave); err != nil {
			return tunnelMoreNone, err
		}
	} else {
		if err := add("Rename…", tunnelMoreRename); err != nil {
			return tunnelMoreNone, err
		}
		if runtime.State == model.StateDisconnected {
			if err := add("Edit…", tunnelMoreEdit); err != nil {
				return tunnelMoreNone, err
			}
			if err := add("Delete…", tunnelMoreDelete); err != nil {
				return tunnelMoreNone, err
			}
		}
	}
	closeButton, err := walk.NewPushButton(dialog)
	if err != nil {
		return tunnelMoreNone, err
	}
	_ = closeButton.SetText("Close")
	closeButton.Clicked().Attach(dialog.Cancel)
	if err := ApplyStandardTextScale(dialog); err != nil {
		return tunnelMoreNone, err
	}
	dialog.Run()
	return selected, nil
}
