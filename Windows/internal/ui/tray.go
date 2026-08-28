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

type SettingsPresentation struct {
	ShowTrayIcon   bool
	AppearanceText string
}

func PresentSettings(showTray bool, appearance Appearance) SettingsPresentation {
	appearanceName := "Light"
	if appearance == AppearanceDark {
		appearanceName = "Dark"
	}
	return SettingsPresentation{ShowTrayIcon: showTray, AppearanceText: "Appearance follows Windows (" + appearanceName + ")"}
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
	presentation := PresentSettings(t.controller.Visible(), t.window.Environment().Appearance())
	shell, err := NewDialogShell(t.window, t.window.Environment(), DialogSpec{
		Title: "TunnelDock Settings", Description: "Application behavior and appearance.",
		PrimaryText: "Close", Size: walk.Size{Width: 440, Height: 280},
	})
	if err != nil {
		walk.MsgBox(t.window, "Settings", err.Error(), walk.MsgBoxIconError)
		return
	}
	defer shell.Dispose()
	shell.Cancel.SetVisible(false)
	layout := walk.NewVBoxLayout()
	layout.SetMargins(walk.Margins{HNear: 14, VNear: 12, HFar: 14, VFar: 12})
	layout.SetSpacing(10)
	if err := shell.Content.SetLayout(layout); err != nil {
		showDialogError(t.window, err)
		return
	}
	showTray, err := walk.NewCheckBox(shell.Content)
	if err != nil {
		showDialogError(t.window, err)
		return
	}
	_ = showTray.SetText("Show TunnelDock tray icon")
	if presentation.ShowTrayIcon {
		showTray.SetCheckState(walk.CheckChecked)
	}
	appearance, err := walk.NewLabel(shell.Content)
	if err != nil {
		showDialogError(t.window, err)
		return
	}
	_ = appearance.SetText(presentation.AppearanceText)
	syncing := false
	showTray.CheckStateChanged().Attach(func() {
		if syncing {
			return
		}
		previous := t.controller.Visible()
		visible := showTray.CheckState() == walk.CheckChecked
		if visible == previous {
			return
		}
		if setErr := t.controller.SetVisible(visible); setErr != nil {
			syncing = true
			showTray.SetCheckState(checkState(previous))
			syncing = false
			shell.SetValidation(setErr.Error(), showTray)
			return
		}
		if setErr := t.SetVisible(visible); setErr != nil {
			_ = t.controller.SetVisible(previous)
			syncing = true
			showTray.SetCheckState(checkState(previous))
			syncing = false
			shell.SetValidation(setErr.Error(), showTray)
			return
		}
		shell.SetValidation("", nil)
	})
	shell.Primary.Clicked().Attach(shell.Accept)
	shell.Run()
}

func checkState(checked bool) walk.CheckState {
	if checked {
		return walk.CheckChecked
	}
	return walk.CheckUnchecked
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
