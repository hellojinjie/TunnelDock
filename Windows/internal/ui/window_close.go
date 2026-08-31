package ui

import (
	"github.com/tailscale/walk"
	"github.com/tailscale/win"
)

type closeAction uint8

const (
	closeWindowToTray closeAction = iota
	confirmApplicationExit
)

func windowCloseAction(trayVisible bool) closeAction {
	if trayVisible {
		return closeWindowToTray
	}
	return confirmApplicationExit
}

// closeConfirmation is the user's answer to presentWindowCloseWarning.
type closeConfirmation uint8

const (
	closeConfirmCancel closeConfirmation = iota
	closeConfirmQuit
	closeConfirmEnableTray
)

func confirmationFromMsgBoxResult(result int) closeConfirmation {
	switch result {
	case win.IDYES:
		return closeConfirmQuit
	case win.IDNO:
		return closeConfirmEnableTray
	default:
		return closeConfirmCancel
	}
}

type windowCloseWarning struct {
	Title string
	Text  string
}

func presentWindowCloseWarning() windowCloseWarning {
	return windowCloseWarning{
		Title: "Quit TunnelDock?",
		Text: "Quitting TunnelDock will close all active tunnels.\r\n\r\n" +
			"Yes: quit TunnelDock and close the tunnels.\r\n" +
			"No: show the tray icon and keep TunnelDock and its tunnels running.\r\n" +
			"Cancel: return to TunnelDock.",
	}
}

// closeDecision is the effect a close request should have. It contains no
// Win32 calls, so the policy in decideWindowClose can be exercised by tests
// without a real window, NotifyIcon, or MsgBox.
type closeDecision struct {
	HideWindow  bool
	ShowTray    bool
	ExitProcess bool
}

// decideWindowClose is the single source of truth for what a click on the
// main window's close button does. confirm is only invoked (and the user is
// only ever asked) when the tray icon is not currently visible; it must
// return the user's choice from presentWindowCloseWarning.
func decideWindowClose(trayVisible bool, confirm func() closeConfirmation) closeDecision {
	if windowCloseAction(trayVisible) == closeWindowToTray {
		return closeDecision{HideWindow: true}
	}
	switch confirm() {
	case closeConfirmQuit:
		return closeDecision{ExitProcess: true}
	case closeConfirmEnableTray:
		return closeDecision{HideWindow: true, ShowTray: true}
	default:
		return closeDecision{}
	}
}

// HandleWindowClose always cancels native form disposal first. This makes a
// close request fail safe: an unexpected dialog or tray error cannot destroy
// the main window and terminate the application. It returns true only after
// the user explicitly confirms application exit; the caller is responsible
// for actually exiting the process (via walk.App().Exit) in that case, since
// walk.MainWindow's built-in WM_CLOSE handling has been disabled
// (SetExitOnClose(false), see NewMainWindowWithEnvironment) specifically so
// that no close request can terminate the app on its own.
func (t *Tray) HandleWindowClose(cancel *bool) bool {
	*cancel = true

	decision := decideWindowClose(t.controller.Visible(), func() closeConfirmation {
		presentation := presentWindowCloseWarning()
		result := walk.MsgBox(
			t.window,
			presentation.Title,
			presentation.Text,
			walk.MsgBoxYesNoCancel|walk.MsgBoxIconWarning|walk.MsgBoxDefButton3,
		)
		return confirmationFromMsgBoxResult(result)
	})
	return t.applyCloseDecision(decision)
}

// applyCloseDecision carries out a closeDecision's side effects and reports
// whether the caller should now exit the process. Split out from
// HandleWindowClose so tests can drive every branch (hide, enable-tray,
// quit, cancel) against real Window/NotifyIcon/TrayController state without
// going through the modal confirmation dialog.
func (t *Tray) applyCloseDecision(decision closeDecision) bool {
	if decision.ShowTray {
		if err := t.controller.SetVisible(true); err != nil {
			walk.MsgBox(t.window, "TunnelDock", err.Error(), walk.MsgBoxIconError)
			return false
		}
		if err := t.SetVisible(true); err != nil {
			_ = t.controller.SetVisible(false)
			walk.MsgBox(t.window, "TunnelDock", err.Error(), walk.MsgBoxIconError)
			return false
		}
	}
	if decision.HideWindow {
		t.window.SetVisible(false)
	}
	return decision.ExitProcess
}
