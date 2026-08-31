package ui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hellojinjie/TunnelDock/Windows/internal/app"
	"github.com/hellojinjie/TunnelDock/Windows/internal/persistence"
	"github.com/tailscale/walk"
	"github.com/tailscale/win"
)

func TestWindowCloseActionDependsOnTrayVisibility(t *testing.T) {
	if got := windowCloseAction(true); got != closeWindowToTray {
		t.Fatalf("windowCloseAction(true) = %v, want closeWindowToTray", got)
	}
	if got := windowCloseAction(false); got != confirmApplicationExit {
		t.Fatalf("windowCloseAction(false) = %v, want confirmApplicationExit", got)
	}
}

func TestCloseWarningExplainsTunnelShutdownAndTrayAlternative(t *testing.T) {
	presentation := presentWindowCloseWarning()
	warning := strings.ToLower(presentation.Text)
	if !strings.Contains(warning, "close all active tunnels") ||
		!strings.Contains(warning, "show the tray icon") ||
		!strings.Contains(warning, "cancel") {
		t.Fatalf("Text = %q, want tunnel shutdown, tray alternative, and cancel behavior", presentation.Text)
	}
}

// --- decision policy: pure, no Win32 involved -------------------------------

func TestDecideWindowCloseWithTrayVisibleHidesWithoutAskingOrExiting(t *testing.T) {
	asked := 0
	decision := decideWindowClose(true, func() closeConfirmation {
		asked++
		return closeConfirmQuit // must never be reached: tray-visible short-circuits before asking
	})
	if asked != 0 {
		t.Fatalf("confirm called %d times, want 0 (tray visible must never prompt)", asked)
	}
	if !decision.HideWindow || decision.ShowTray || decision.ExitProcess {
		t.Fatalf("decision = %+v, want {HideWindow:true}", decision)
	}
}

func TestDecideWindowCloseWithTrayHiddenAlwaysAsksExactlyOnce(t *testing.T) {
	for _, confirmation := range []closeConfirmation{closeConfirmCancel, closeConfirmQuit, closeConfirmEnableTray} {
		asked := 0
		decideWindowClose(false, func() closeConfirmation {
			asked++
			return confirmation
		})
		if asked != 1 {
			t.Fatalf("confirmation %v: confirm called %d times, want exactly 1", confirmation, asked)
		}
	}
}

func TestDecideWindowCloseCancelLeavesEverythingRunning(t *testing.T) {
	decision := decideWindowClose(false, func() closeConfirmation { return closeConfirmCancel })
	if decision.HideWindow || decision.ShowTray || decision.ExitProcess {
		t.Fatalf("decision = %+v, want all false", decision)
	}
}

func TestDecideWindowCloseQuitOnlyExitsProcess(t *testing.T) {
	decision := decideWindowClose(false, func() closeConfirmation { return closeConfirmQuit })
	if decision.HideWindow || decision.ShowTray || !decision.ExitProcess {
		t.Fatalf("decision = %+v, want {ExitProcess:true}", decision)
	}
}

func TestDecideWindowCloseEnableTrayHidesWindowAndShowsTrayWithoutExiting(t *testing.T) {
	decision := decideWindowClose(false, func() closeConfirmation { return closeConfirmEnableTray })
	if !decision.HideWindow || !decision.ShowTray || decision.ExitProcess {
		t.Fatalf("decision = %+v, want {HideWindow:true, ShowTray:true}", decision)
	}
}

// --- real Win32 regression coverage -----------------------------------------
//
// The tests below drive an actual *walk.MainWindow / *walk.NotifyIcon through
// the same code paths production uses, on the locked UI thread that TestMain
// (in ui_smoke_windows_test.go) sets up. They exist because the real bug was
// never in the decision policy above: walk.MainWindow posts WM_QUIT and
// terminates Application.Run on *every* WM_CLOSE unless
// SetExitOnClose(false) has been called, regardless of what a Closing
// handler does with its cancel flag. A test that only checks strings or the
// closeAction/closeConfirmation enums cannot catch that class of bug.

type memoryTraySettings struct{ settings persistence.Settings }

func (s *memoryTraySettings) Load() (persistence.Settings, error) { return s.settings, nil }
func (s *memoryTraySettings) Save(settings persistence.Settings) error {
	s.settings = settings
	return nil
}

func TestClosingWithCancelDoesNotTerminateMessageLoop(t *testing.T) {
	if uiTestInitErr != nil {
		t.Skipf("Walk UI is unavailable in this test session: %v", uiTestInitErr)
	}
	var (
		setupErr           error
		visibleAfterClose  bool
		disposedAfterClose bool
		quitWasPosted      bool
	)
	runOnTestUIThread(func() {
		env := newUIEnvironment(fixedAppearanceSource{appearance: AppearanceLight})
		defer env.Dispose()
		window, err := NewMainWindowWithEnvironment(app.NewModel(), nil, env)
		if err != nil {
			setupErr = err
			return
		}
		defer window.Dispose()

		// Mirrors production wiring exactly (see cmd/tunneldock/main.go and
		// the tray-visible branch of Tray.HandleWindowClose): the Closing
		// handler always cancels native disposal and only hides the window.
		// Nothing here ever calls walk.App().Exit for this scenario.
		window.Closing().Attach(func(cancel *bool, _ walk.CloseReason) {
			*cancel = true
			window.SetVisible(false)
		})
		window.Show()

		// Drain any WM_QUIT already queued from earlier in this test binary
		// so the check below is specific to the close triggered here.
		var msg win.MSG
		for win.PeekMessage(&msg, 0, win.WM_QUIT, win.WM_QUIT, win.PM_REMOVE) {
		}

		window.Close() // synchronously sends WM_CLOSE, same as clicking the title bar X

		visibleAfterClose = window.Visible()
		disposedAfterClose = window.IsDisposed()
		quitWasPosted = win.PeekMessage(&msg, 0, win.WM_QUIT, win.WM_QUIT, win.PM_REMOVE)
	})
	if setupErr != nil {
		t.Fatal(setupErr)
	}
	if quitWasPosted {
		t.Fatal("WM_CLOSE with a cancelled Closing handler posted WM_QUIT; " +
			"this would break Application.Run's message loop and exit the whole " +
			"process (killing all running tunnels) even though the user only asked to hide the window")
	}
	if visibleAfterClose {
		t.Fatal("window is still visible after close; expected it to be hidden")
	}
	if disposedAfterClose {
		t.Fatal("window was disposed by a cancelled WM_CLOSE; expected native disposal to be skipped entirely")
	}
}

func TestRepeatedCancelledClosesNeverDisposeNativeWindow(t *testing.T) {
	if uiTestInitErr != nil {
		t.Skipf("Walk UI is unavailable in this test session: %v", uiTestInitErr)
	}
	var setupErr error
	runOnTestUIThread(func() {
		env := newUIEnvironment(fixedAppearanceSource{appearance: AppearanceLight})
		defer env.Dispose()
		window, err := NewMainWindowWithEnvironment(app.NewModel(), nil, env)
		if err != nil {
			setupErr = err
			return
		}
		window.Closing().Attach(func(cancel *bool, _ walk.CloseReason) {
			*cancel = true
			window.SetVisible(false)
		})
		window.Show()
		for i := 0; i < 3; i++ {
			window.Close()
			window.SetVisible(true)
			window.Show()
		}
		// The single legitimate disposal path, mirroring main()'s
		// `defer mainWindow.Dispose()`. Earlier implementations could reach
		// this point having already disposed the native window from inside
		// WM_CLOSE handling, which made this second Dispose panic with
		// "send on closed channel". If that regresses, this call panics and
		// the test binary reports it as a failure.
		window.Dispose()
		if !window.IsDisposed() {
			setupErr = fmt.Errorf("window.IsDisposed() = false after the single legitimate Dispose() call")
		}
	})
	if setupErr != nil {
		t.Fatal(setupErr)
	}
}

func TestApplyCloseDecisionHideOnlyHidesWindowAndLeavesTrayAndProcessAlone(t *testing.T) {
	if uiTestInitErr != nil {
		t.Skipf("Walk UI is unavailable in this test session: %v", uiTestInitErr)
	}
	var (
		setupErr                   error
		windowVisible, trayVisible bool
		exitProcess                bool
	)
	runOnTestUIThread(func() {
		env := newUIEnvironment(fixedAppearanceSource{appearance: AppearanceLight})
		defer env.Dispose()
		window, err := NewMainWindowWithEnvironment(app.NewModel(), nil, env)
		if err != nil {
			setupErr = err
			return
		}
		defer window.Dispose()
		window.Show()

		icon, err := walk.NewNotifyIcon()
		if err != nil {
			setupErr = err
			return
		}
		defer icon.Dispose()
		if err := icon.SetVisible(true); err != nil {
			setupErr = err
			return
		}

		store := &memoryTraySettings{settings: persistence.Settings{ShowTrayIcon: true}}
		controller, err := app.NewTrayController(store)
		if err != nil {
			setupErr = err
			return
		}

		tray := &Tray{window: window, controller: controller, icon: icon}
		exitProcess = tray.applyCloseDecision(closeDecision{HideWindow: true})

		windowVisible = window.Visible()
		trayVisible = tray.Visible()
	})
	if setupErr != nil {
		t.Fatal(setupErr)
	}
	if windowVisible {
		t.Fatal("window still visible after a HideWindow decision")
	}
	if !trayVisible {
		t.Fatal("tray visibility changed unexpectedly for a HideWindow-only decision")
	}
	if exitProcess {
		t.Fatal("HideWindow decision must never signal process exit")
	}
}

func TestApplyCloseDecisionEnableTrayPersistsSettingsShowsTrayHidesWindowKeepsProcessAlive(t *testing.T) {
	if uiTestInitErr != nil {
		t.Skipf("Walk UI is unavailable in this test session: %v", uiTestInitErr)
	}
	var (
		setupErr                           error
		windowVisible, trayVisible         bool
		exitProcess, persistedShowTrayIcon bool
		controllerVisible                  bool
	)
	runOnTestUIThread(func() {
		env := newUIEnvironment(fixedAppearanceSource{appearance: AppearanceLight})
		defer env.Dispose()
		window, err := NewMainWindowWithEnvironment(app.NewModel(), nil, env)
		if err != nil {
			setupErr = err
			return
		}
		defer window.Dispose()
		window.Show()

		icon, err := walk.NewNotifyIcon()
		if err != nil {
			setupErr = err
			return
		}
		defer icon.Dispose()
		// Tray starts hidden: this is the confirmApplicationExit path.
		store := &memoryTraySettings{settings: persistence.Settings{ShowTrayIcon: false}}
		controller, err := app.NewTrayController(store)
		if err != nil {
			setupErr = err
			return
		}

		tray := &Tray{window: window, controller: controller, icon: icon}
		exitProcess = tray.applyCloseDecision(closeDecision{HideWindow: true, ShowTray: true})

		windowVisible = window.Visible()
		trayVisible = tray.Visible()
		controllerVisible = controller.Visible()
		persistedShowTrayIcon = store.settings.ShowTrayIcon
	})
	if setupErr != nil {
		t.Fatal(setupErr)
	}
	if windowVisible {
		t.Fatal("window still visible after choosing to enable the tray")
	}
	if !trayVisible {
		t.Fatal("tray NotifyIcon was not made visible")
	}
	if !controllerVisible || !persistedShowTrayIcon {
		t.Fatalf("settings not persisted: controllerVisible=%v persisted=%v", controllerVisible, persistedShowTrayIcon)
	}
	if exitProcess {
		t.Fatal("enabling the tray must never signal process exit")
	}
}

func TestApplyCloseDecisionQuitReturnsTrueWithoutTouchingWindowOrTray(t *testing.T) {
	if uiTestInitErr != nil {
		t.Skipf("Walk UI is unavailable in this test session: %v", uiTestInitErr)
	}
	var (
		setupErr                   error
		windowVisible, trayVisible bool
		exitProcess                bool
	)
	runOnTestUIThread(func() {
		env := newUIEnvironment(fixedAppearanceSource{appearance: AppearanceLight})
		defer env.Dispose()
		window, err := NewMainWindowWithEnvironment(app.NewModel(), nil, env)
		if err != nil {
			setupErr = err
			return
		}
		defer window.Dispose()
		window.Show()

		icon, err := walk.NewNotifyIcon()
		if err != nil {
			setupErr = err
			return
		}
		defer icon.Dispose()

		store := &memoryTraySettings{settings: persistence.Settings{ShowTrayIcon: false}}
		controller, err := app.NewTrayController(store)
		if err != nil {
			setupErr = err
			return
		}

		tray := &Tray{window: window, controller: controller, icon: icon}
		exitProcess = tray.applyCloseDecision(closeDecision{ExitProcess: true})

		windowVisible = window.Visible()
		trayVisible = tray.Visible()
	})
	if setupErr != nil {
		t.Fatal(setupErr)
	}
	if !exitProcess {
		t.Fatal("Quit decision must signal process exit; the caller (main.go) is what actually calls walk.App().Exit and stops tunnels")
	}
	if !windowVisible {
		t.Fatal("Quit decision must not hide the window itself; the process is expected to exit and tear everything down instead")
	}
	if trayVisible {
		t.Fatal("Quit decision must not touch the tray")
	}
}

func TestApplyCloseDecisionCancelChangesNothing(t *testing.T) {
	if uiTestInitErr != nil {
		t.Skipf("Walk UI is unavailable in this test session: %v", uiTestInitErr)
	}
	var (
		setupErr                   error
		windowVisible, trayVisible bool
		exitProcess                bool
	)
	runOnTestUIThread(func() {
		env := newUIEnvironment(fixedAppearanceSource{appearance: AppearanceLight})
		defer env.Dispose()
		window, err := NewMainWindowWithEnvironment(app.NewModel(), nil, env)
		if err != nil {
			setupErr = err
			return
		}
		defer window.Dispose()
		window.Show()

		icon, err := walk.NewNotifyIcon()
		if err != nil {
			setupErr = err
			return
		}
		defer icon.Dispose()

		store := &memoryTraySettings{settings: persistence.Settings{ShowTrayIcon: false}}
		controller, err := app.NewTrayController(store)
		if err != nil {
			setupErr = err
			return
		}

		tray := &Tray{window: window, controller: controller, icon: icon}
		exitProcess = tray.applyCloseDecision(closeDecision{})

		windowVisible = window.Visible()
		trayVisible = tray.Visible()
	})
	if setupErr != nil {
		t.Fatal(setupErr)
	}
	if !windowVisible {
		t.Fatal("Cancel decision hid the window; expected it to remain visible")
	}
	if trayVisible {
		t.Fatal("Cancel decision changed tray visibility")
	}
	if exitProcess {
		t.Fatal("Cancel decision must never signal process exit")
	}
}
