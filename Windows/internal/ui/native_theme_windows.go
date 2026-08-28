package ui

import (
	"unsafe"

	"github.com/tailscale/walk"
	"github.com/tailscale/win"
	"golang.org/x/sys/windows"
)

func ApplyWindowAppearance(window walk.Window, appearance Appearance) {
	if window == nil || window.Handle() == 0 {
		return
	}
	dark := int32(0)
	theme := "Explorer"
	if appearance == AppearanceDark {
		dark = 1
		theme = "DarkMode_Explorer"
	}
	if _, topLevel := window.(walk.Form); topLevel {
		win.DwmSetWindowAttribute(window.Handle(), win.DWMWA_USE_IMMERSIVE_DARK_MODE, unsafe.Pointer(&dark), uint32(unsafe.Sizeof(dark)))
	}
	if themeName, err := windows.UTF16PtrFromString(theme); err == nil {
		win.SetWindowTheme(window.Handle(), themeName, nil)
	}
	if container, ok := window.(walk.Container); ok {
		for index := 0; index < container.Children().Len(); index++ {
			ApplyWindowAppearance(container.Children().At(index), appearance)
		}
	}
	win.RedrawWindow(window.Handle(), nil, 0, win.RDW_INVALIDATE|win.RDW_FRAME|win.RDW_ALLCHILDREN)
}
