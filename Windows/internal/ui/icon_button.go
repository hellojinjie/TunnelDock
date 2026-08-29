package ui

import (
	"github.com/tailscale/walk"
	"github.com/tailscale/win"
)

type IconButton struct {
	*walk.CustomWidget
	env                   *UIEnvironment
	icon                  IconKind
	surface               iconButtonSurface
	pressed               bool
	hovered               bool
	keyboardFocus         bool
	settingFocusWithMouse bool
	onClick               func()
}

type iconButtonSurface uint8

const (
	iconButtonOnWindow iconButtonSurface = iota
	iconButtonOnSidebar
)

type iconButtonBase uint8

const (
	iconButtonBaseWindow iconButtonBase = iota
	iconButtonBaseSidebar
)

type iconButtonOverlay uint8

const (
	iconButtonOverlayNone iconButtonOverlay = iota
	iconButtonOverlayHover
	iconButtonOverlayPressed
)

type iconButtonVisual struct {
	Base         iconButtonBase
	Overlay      iconButtonOverlay
	OverlayInset int
	ShowFocus    bool
}

func iconButtonPresentation(surface iconButtonSurface, hovered, pressed, focused, keyboardFocus bool) iconButtonVisual {
	visual := iconButtonVisual{Base: iconButtonBaseWindow, OverlayInset: 2, ShowFocus: focused && keyboardFocus}
	if surface == iconButtonOnSidebar {
		visual.Base = iconButtonBaseSidebar
	}
	if pressed {
		visual.Overlay = iconButtonOverlayPressed
	} else if hovered {
		visual.Overlay = iconButtonOverlayHover
	}
	return visual
}

func NewIconButton(parent walk.Container, env *UIEnvironment, surface iconButtonSurface, icon IconKind, tooltip string, onClick func()) (*IconButton, error) {
	button := &IconButton{env: env, icon: icon, surface: surface, onClick: onClick}
	custom, err := walk.NewCustomWidgetPixels(parent, uint(win.WS_TABSTOP), func(canvas *walk.Canvas, bounds walk.Rectangle) error {
		return button.paint(canvas, bounds)
	})
	if err != nil {
		return nil, err
	}
	button.CustomWidget = custom
	button.SetPaintMode(walk.PaintBuffered)
	button.SetInvalidatesOnResize(true)
	resources, err := env.Resources(button.DPI())
	if err != nil {
		button.Dispose()
		return nil, err
	}
	size := resources.Metrics.ActionHeight
	if err := button.SetMinMaxSizePixels(walk.Size{Width: size, Height: size}, walk.Size{Width: size, Height: size}); err != nil {
		button.Dispose()
		return nil, err
	}
	button.SetToolTipText(tooltip)
	button.SetEnabled(onClick != nil)
	button.MouseMove().Attach(func(_, _ int, _ walk.MouseButton) {
		if !button.hovered {
			button.hovered = true
			button.Invalidate()
		}
	})
	button.MouseDown().Attach(func(_, _ int, mouseButton walk.MouseButton) {
		if mouseButton == walk.LeftButton && button.Enabled() {
			button.pressed = true
			button.keyboardFocus = false
			button.settingFocusWithMouse = true
			_ = button.SetFocus()
			button.settingFocusWithMouse = false
			button.Invalidate()
		}
	})
	button.MouseUp().Attach(func(_, _ int, mouseButton walk.MouseButton) {
		if mouseButton == walk.LeftButton && button.pressed {
			button.pressed = false
			button.Invalidate()
			if button.onClick != nil {
				button.onClick()
			}
		}
	})
	button.KeyDown().Attach(func(key walk.Key) {
		if button.Enabled() && (key == walk.KeyReturn || key == walk.KeySpace) && button.onClick != nil {
			button.onClick()
		}
	})
	button.FocusedChanged().Attach(func() {
		if !button.Focused() {
			button.keyboardFocus = false
		} else if !button.settingFocusWithMouse {
			button.keyboardFocus = true
		}
		button.Invalidate()
	})
	return button, nil
}

func (b *IconButton) SetOnClick(onClick func()) {
	b.onClick = onClick
	b.SetEnabled(onClick != nil)
	b.Invalidate()
}

func (b *IconButton) paint(canvas *walk.Canvas, _ walk.Rectangle) error {
	resources, err := b.env.Resources(b.DPI())
	if err != nil {
		return err
	}
	bounds := b.ClientBoundsPixels()
	visual := iconButtonPresentation(b.surface, b.hovered, b.pressed, b.Focused(), b.keyboardFocus)
	baseBrush := resources.WindowBrush
	if visual.Base == iconButtonBaseSidebar {
		baseBrush = resources.SidebarBrush
	}
	if err := canvas.FillRectanglePixels(baseBrush, bounds); err != nil {
		return err
	}
	var overlay walk.Brush
	switch visual.Overlay {
	case iconButtonOverlayHover:
		overlay = resources.HoverBrush
	case iconButtonOverlayPressed:
		overlay = resources.SelectedBrush
	}
	if overlay != nil {
		if err := FillRoundedSurface(canvas, overlay, insetRect(bounds, visual.OverlayInset), resources.Metrics.RowRadius); err != nil {
			return err
		}
	}
	color := resources.Palette.SecondaryText
	if !b.Enabled() {
		color = resources.Palette.DisabledText
	}
	iconBounds := insetRect(bounds, max(5, resources.Metrics.IconSize/3))
	if err := DrawIcon(canvas, resources, b.icon, iconBounds, color); err != nil {
		return err
	}
	if visual.ShowFocus {
		return DrawFocusRing(canvas, resources, insetRect(bounds, 2), resources.Metrics.RowRadius)
	}
	return nil
}
