package app

import (
	"errors"
	"fmt"

	"golang.org/x/sys/windows"
)

var ErrAlreadyRunning = errors.New("TunnelDock is already running")

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
