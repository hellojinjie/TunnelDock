package ui

import (
	"context"
	"fmt"

	"github.com/hellojinjie/TunnelDock/Windows/internal/app"
	"github.com/hellojinjie/TunnelDock/Windows/internal/model"
	"github.com/hellojinjie/TunnelDock/Windows/internal/tunnel"
	"github.com/tailscale/walk"
	"github.com/tailscale/win"
)

// Tray owns the native notification icon and its Windows-only menu. It never
// lists temporary tunnels because their lifetime is intentionally ephemeral.
type Tray struct {
	window     *Window
	controller *app.TrayController
	manager    *tunnel.Manager
	icon       *walk.NotifyIcon
	onRefresh  func()
	onQuit     func()
}

func NewTray(window *Window, controller *app.TrayController, manager *tunnel.Manager, onRefresh, onQuit func()) (*Tray, error) {
	if window == nil || controller == nil {
		return nil, fmt.Errorf("tray requires a window and settings controller")
	}
	icon, err := walk.NewIconFromResource("APP")
	if err != nil {
		return nil, fmt.Errorf("load application icon: %w", err)
	}
	notifyIcon, err := walk.NewNotifyIcon()
	if err != nil {
		return nil, err
	}
	tray := &Tray{window: window, controller: controller, manager: manager, icon: notifyIcon, onRefresh: onRefresh, onQuit: onQuit}
	if err := notifyIcon.SetIcon(icon); err != nil {
		_ = notifyIcon.Dispose()
		return nil, err
	}
	if err := notifyIcon.SetToolTip("TunnelDock"); err != nil {
		_ = notifyIcon.Dispose()
		return nil, err
	}
	notifyIcon.MouseDown().Attach(func(_ int, _ int, button walk.MouseButton) {
		if button == walk.LeftButton {
			tray.OpenWindow()
		}
	})
	notifyIcon.ShowingContextMenu().Attach(func() bool {
		tray.rebuildMenu()
		return true
	})
	tray.rebuildMenu()
	if err := tray.SetVisible(controller.Visible()); err != nil {
		_ = notifyIcon.Dispose()
		return nil, err
	}
	return tray, nil
}

func (t *Tray) Dispose() error { return t.icon.Dispose() }

func (t *Tray) Visible() bool { return t.icon.Visible() }

func (t *Tray) SetVisible(visible bool) error { return t.icon.SetVisible(visible) }

func (t *Tray) OpenWindow() {
	t.window.SetVisible(true)
	t.window.Show()
}

func (t *Tray) ShowSettings() {
	dialog, err := walk.NewDialogWithFixedSize(t.window)
	if err != nil {
		walk.MsgBox(t.window, "Settings", err.Error(), walk.MsgBoxIconError)
		return
	}
	defer dialog.Dispose()
	_ = dialog.SetTitle("TunnelDock Settings")
	_ = dialog.SetSize(walk.Size{Width: 360, Height: 130})
	_ = dialog.SetLayout(walk.NewVBoxLayout())
	showTray, err := walk.NewCheckBox(dialog)
	if err != nil {
		walk.MsgBox(t.window, "Settings", err.Error(), walk.MsgBoxIconError)
		return
	}
	_ = showTray.SetText("Show TunnelDock tray icon")
	if t.controller.Visible() {
		showTray.SetCheckState(walk.CheckChecked)
	}
	buttons, err := walk.NewComposite(dialog)
	if err != nil {
		walk.MsgBox(t.window, "Settings", err.Error(), walk.MsgBoxIconError)
		return
	}
	_ = buttons.SetLayout(walk.NewHBoxLayout())
	save, err := walk.NewPushButton(buttons)
	if err != nil {
		walk.MsgBox(t.window, "Settings", err.Error(), walk.MsgBoxIconError)
		return
	}
	_ = save.SetText("Save")
	save.Clicked().Attach(dialog.Accept)
	cancel, err := walk.NewPushButton(buttons)
	if err != nil {
		walk.MsgBox(t.window, "Settings", err.Error(), walk.MsgBoxIconError)
		return
	}
	_ = cancel.SetText("Cancel")
	cancel.Clicked().Attach(dialog.Cancel)
	if dialog.Run() != walk.DlgCmdOK {
		return
	}
	visible := showTray.CheckState() == walk.CheckChecked
	if err := t.controller.SetVisible(visible); err != nil {
		walk.MsgBox(t.window, "Settings", err.Error(), walk.MsgBoxIconError)
		return
	}
	if err := t.SetVisible(visible); err != nil {
		walk.MsgBox(t.window, "Settings", err.Error(), walk.MsgBoxIconError)
	}
}

func (t *Tray) rebuildMenu() {
	actions := t.icon.ContextMenu().Actions()
	_ = actions.Clear()
	t.addAction("TunnelDock", nil, false)
	t.addAction("Open TunnelDock", t.OpenWindow, true)
	t.addAction("Refresh SSH Config", t.onRefresh, true)
	for _, runtime := range t.manager.Snapshots() {
		if runtime.Temporary {
			continue
		}
		runtime := runtime
		verb := "Connect"
		if runtime.State != model.StateDisconnected {
			verb = "Disconnect"
		}
		t.addAction(fmt.Sprintf("%s — %s", runtime.DisplayName(), verb), func() {
			go t.toggleTunnel(runtime.ID, verb == "Connect")
		}, true)
	}
	t.addAction("Settings", t.ShowSettings, true)
	t.addAction("Quit TunnelDock", t.onQuit, true)
}

func (t *Tray) addAction(text string, callback func(), enabled bool) {
	action := walk.NewAction()
	_ = action.SetText(text)
	_ = action.SetEnabled(enabled)
	if callback != nil {
		action.Triggered().Attach(callback)
	}
	_ = t.icon.ContextMenu().Actions().Add(action)
}

func (t *Tray) toggleTunnel(id string, connect bool) {
	var err error
	if connect {
		err = t.manager.ConnectSaved(context.Background(), id)
	} else {
		err = t.manager.Disconnect(id)
	}
	walk.App().Synchronize(func() {
		if err != nil {
			walk.MsgBox(t.window, "TunnelDock", err.Error(), walk.MsgBoxIconError)
		}
		_ = t.window.refreshTunnels()
		t.rebuildMenu()
	})
}

func (t *Tray) MinimizeOnClose(cancel *bool) {
	*cancel = true
	if t.Visible() {
		t.window.SetVisible(false)
		return
	}
	win.ShowWindow(t.window.Handle(), win.SW_MINIMIZE)
}
