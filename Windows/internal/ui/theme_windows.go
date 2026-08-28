package ui

import (
	"context"
	"syscall"
	"time"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

const personalizeRegistryPath = `Software\Microsoft\Windows\CurrentVersion\Themes\Personalize`

var regNotifyChangeKeyValue = syscall.NewLazyDLL("advapi32.dll").NewProc("RegNotifyChangeKeyValue")

type appearanceSource interface {
	Current() Appearance
	Watch(context.Context) <-chan Appearance
}

type windowsAppearanceSource struct{}

func (windowsAppearanceSource) Current() Appearance {
	key, err := registry.OpenKey(registry.CURRENT_USER, personalizeRegistryPath, registry.QUERY_VALUE)
	if err != nil {
		return AppearanceLight
	}
	defer key.Close()
	value, _, err := key.GetIntegerValue("AppsUseLightTheme")
	if err == nil && value == 0 {
		return AppearanceDark
	}
	return AppearanceLight
}

func (source windowsAppearanceSource) Watch(ctx context.Context) <-chan Appearance {
	changes := make(chan Appearance, 1)
	go func() {
		defer close(changes)
		key, err := registry.OpenKey(registry.CURRENT_USER, personalizeRegistryPath, registry.QUERY_VALUE|registry.NOTIFY)
		if err != nil {
			return
		}
		defer key.Close()
		event, err := windows.CreateEvent(nil, 0, 0, nil)
		if err != nil {
			return
		}
		defer windows.CloseHandle(event)
		last := source.Current()
		for {
			result, _, _ := regNotifyChangeKeyValue.Call(
				uintptr(key), 0, windows.REG_NOTIFY_CHANGE_LAST_SET, uintptr(event), 1,
			)
			if result != 0 {
				return
			}
			for {
				select {
				case <-ctx.Done():
					return
				default:
				}
				wait, _ := windows.WaitForSingleObject(event, 250)
				if wait == windows.WAIT_OBJECT_0 {
					break
				}
				if wait != uint32(windows.WAIT_TIMEOUT) {
					return
				}
				time.Sleep(10 * time.Millisecond)
			}
			current := source.Current()
			if current != last {
				last = current
				select {
				case changes <- current:
				default:
				}
			}
		}
	}()
	return changes
}
