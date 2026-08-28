package ui

import (
	"github.com/tailscale/walk"
	"github.com/tailscale/win"
)

type IconButton struct {
	*walk.CustomWidget
	env     *UIEnvironment
	icon    IconKind
	pressed bool
	hovered bool
	onClick func()
}

func NewIconButton(parent walk.Container, env *UIEnvironment, icon IconKind, tooltip string, onClick func()) (*IconButton, error) {
	button := &IconButton{env: env, icon: icon, onClick: onClick}
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
	if err := button.SetMinMaxSize(walk.Size{Width: size, Height: size}, walk.Size{Width: size, Height: size}); err != nil {
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
			_ = button.SetFocus()
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
	button.FocusedChanged().Attach(func() { button.Invalidate() })
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
	brush := resources.SurfaceBrush
	if b.pressed {
		brush = resources.SelectedBrush
	} else if b.hovered {
		brush = resources.HoverBrush
	}
	if err := FillRoundedSurface(canvas, brush, bounds, resources.Metrics.RowRadius); err != nil {
		return err
	}
	color := resources.Palette.SecondaryText
	if !b.Enabled() {
		color = resources.Palette.DisabledText
	}
	iconBounds := insetRect(bounds, max(5, resources.Metrics.IconSize/3))
	if err := DrawIcon(canvas, resources, b.icon, iconBounds, color); err != nil {
		return err
	}
	if b.Focused() {
		return DrawFocusRing(canvas, resources, insetRect(bounds, 1), resources.Metrics.RowRadius)
	}
	return nil
}
