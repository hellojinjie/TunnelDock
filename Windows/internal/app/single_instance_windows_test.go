package app

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/tailscale/win"
)

func TestAcquireSingleInstanceRejectsSecondOwner(t *testing.T) {
	name := fmt.Sprintf("Local\\TunnelDock.Test.%d", time.Now().UnixNano())
	first, err := AcquireSingleInstance(name)
	if err != nil {
		t.Fatal(err)
	}
	defer first.Close()
	second, err := AcquireSingleInstance(name)
	if !errors.Is(err, ErrAlreadyRunning) {
		t.Fatalf("second acquire error = %v", err)
	}
	if second != nil {
		t.Fatal("second owner = non-nil")
	}
}

func TestActivateExistingMainWindowRestoresAndForegrounds(t *testing.T) {
	previousFind, previousShow, previousForeground := findWindow, showWindow, setForegroundWindow
	t.Cleanup(func() { findWindow, showWindow, setForegroundWindow = previousFind, previousShow, previousForeground })
	handle := win.HWND(42)
	findWindow = func(_ *uint16, _ *uint16) win.HWND { return handle }
	var restored, foregrounded bool
	showWindow = func(got win.HWND, command int32) bool {
		restored = got == handle && command == win.SW_RESTORE
		return true
	}
	setForegroundWindow = func(got win.HWND) bool {
		foregrounded = got == handle
		return true
	}

	if !ActivateExistingMainWindow("TunnelDock") {
		t.Fatal("ActivateExistingMainWindow() = false")
	}
	if !restored || !foregrounded {
		t.Fatalf("restored=%v foregrounded=%v", restored, foregrounded)
	}
}
