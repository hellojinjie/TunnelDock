package app

import (
	"errors"
	"fmt"

	"github.com/tailscale/win"
	"golang.org/x/sys/windows"
)

var ErrAlreadyRunning = errors.New("TunnelDock is already running")

var (
	findWindow          = win.FindWindow
	showWindow          = win.ShowWindow
	setForegroundWindow = win.SetForegroundWindow
)

type SingleInstance struct{ handle windows.Handle }

func AcquireSingleInstance(name string) (*SingleInstance, error) {
	utf16Name, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return nil, fmt.Errorf("encode mutex name: %w", err)
	}
	handle, err := windows.CreateMutex(nil, true, utf16Name)
	if errors.Is(err, windows.ERROR_ALREADY_EXISTS) {
		_ = windows.CloseHandle(handle)
		return nil, ErrAlreadyRunning
	}
	if err != nil {
		return nil, fmt.Errorf("create instance mutex: %w", err)
	}
	return &SingleInstance{handle: handle}, nil
}

func (instance *SingleInstance) Close() error {
	if instance == nil || instance.handle == 0 {
		return nil
	}
	handle := instance.handle
	instance.handle = 0
	_ = windows.ReleaseMutex(handle)
	return windows.CloseHandle(handle)
}

// ActivateExistingMainWindow restores the primary application's native window
// before a second process exits. Windows may decline foreground activation;
// restoring still provides a visible taskbar entry in that case.
func ActivateExistingMainWindow(title string) bool {
	utf16Title, err := windows.UTF16PtrFromString(title)
	if err != nil {
		return false
	}
	handle := findWindow(nil, utf16Title)
	if handle == 0 {
		return false
	}
	showWindow(handle, win.SW_RESTORE)
	setForegroundWindow(handle)
	return true
}
