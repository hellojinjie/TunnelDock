package ui

import (
	"github.com/hellojinjie/TunnelDock/Windows/internal/model"
	"github.com/tailscale/walk"
	"github.com/tailscale/win"
)

type HostRowWidget struct {
	*walk.CustomWidget
	env          *UIEnvironment
	presentation HostRowPresentation
	selected     bool
	hovered      bool
	pressed      bool
	onActivate   func(string)
}

func NewHostRowWidget(parent walk.Container, env *UIEnvironment, row HostRowPresentation, activate func(string)) (*HostRowWidget, error) {
	widget := &HostRowWidget{env: env, presentation: row, onActivate: activate}
	custom, err := walk.NewCustomWidgetPixels(parent, uint(win.WS_TABSTOP), func(canvas *walk.Canvas, bounds walk.Rectangle) error {
		return widget.paint(canvas, bounds)
	})
	if err != nil {
		return nil, err
	}
	widget.CustomWidget = custom
	widget.SetPaintMode(walk.PaintBuffered)
	widget.SetInvalidatesOnResize(true)
	if err := widget.updateHeight(); err != nil {
		widget.Dispose()
		return nil, err
	}
	widget.MouseMove().Attach(func(_, _ int, _ walk.MouseButton) {
		if !widget.hovered {
			widget.hovered = true
			widget.Invalidate()
		}
	})
	widget.MouseDown().Attach(func(_, _ int, button walk.MouseButton) {
		if button != walk.LeftButton {
			return
		}
		widget.pressed = true
		_ = widget.SetFocus()
		widget.Invalidate()
	})
	widget.MouseUp().Attach(func(_, _ int, button walk.MouseButton) {
		if button != walk.LeftButton || !widget.pressed {
			return
		}
		widget.pressed = false
		widget.Invalidate()
		if widget.onActivate != nil {
			widget.onActivate(widget.presentation.ID)
		}
	})
	widget.KeyDown().Attach(func(key walk.Key) {
		if key == walk.KeyReturn || key == walk.KeySpace {
			if widget.onActivate != nil {
				widget.onActivate(widget.presentation.ID)
			}
		}
	})
	widget.FocusedChanged().Attach(func() { _ = widget.Invalidate() })
	widget.SetToolTipText(row.Title)
	return widget, nil
}

func (w *HostRowWidget) SetPresentation(row HostRowPresentation) {
	w.presentation = row
	w.SetToolTipText(row.Title)
	_ = w.updateHeight()
	w.Invalidate()
}

func (w *HostRowWidget) SetSelected(selected bool) {
	if w.selected == selected {
		return
	}
	w.selected = selected
	w.Invalidate()
}

func (w *HostRowWidget) updateHeight() error {
	resources, err := w.env.Resources(w.DPI())
	if err != nil {
		return err
	}
	height := resources.Metrics.HostRowHeight
	return w.SetMinMaxSize(walk.Size{Height: height}, walk.Size{Height: height})
}

func (w *HostRowWidget) paint(canvas *walk.Canvas, updateBounds walk.Rectangle) error {
	resources, err := w.env.Resources(w.DPI())
	if err != nil {
		return err
	}
	bounds := w.ClientBoundsPixels()
	if err := canvas.FillRectanglePixels(resources.SidebarBrush, bounds); err != nil {
		return err
	}
	surface := walk.Brush(nil)
	switch {
	case w.pressed:
		surface = resources.SelectedBrush
	case w.selected:
		surface = resources.SelectedBrush
	case w.hovered:
		surface = resources.HoverBrush
	}
	if surface != nil {
		if err := FillRoundedSurface(canvas, surface, insetRect(bounds, 2), resources.Metrics.RowRadius); err != nil {
			return err
		}
	}
	iconBounds := walk.Rectangle{
		X:     resources.Metrics.SidebarPadding,
		Y:     (bounds.Height - resources.Metrics.IconSize) / 2,
		Width: resources.Metrics.IconSize, Height: resources.Metrics.IconSize,
	}
	icon := IconServer
	iconColor := resources.Palette.SecondaryText
	if w.presentation.Missing || w.presentation.Availability != model.HostAvailable {
		icon = IconWarning
		iconColor = resources.Palette.Warning
	}
	if err := DrawIcon(canvas, resources, icon, iconBounds, iconColor); err != nil {
		return err
	}
	textBounds := walk.Rectangle{
		X:      iconBounds.X + iconBounds.Width + 8,
		Y:      0,
		Width:  max(1, bounds.Width-iconBounds.X-iconBounds.Width-44),
		Height: bounds.Height,
	}
	if err := DrawTextEllipsized(canvas, w.presentation.Title, resources.BodyFont, resources.Palette.PrimaryText, textBounds); err != nil {
		return err
	}
	if w.presentation.Active {
		activeBounds := walk.Rectangle{
			X:     bounds.Width - resources.Metrics.SidebarPadding - resources.Metrics.IconSize,
			Y:     (bounds.Height - resources.Metrics.IconSize) / 2,
			Width: resources.Metrics.IconSize, Height: resources.Metrics.IconSize,
		}
		if err := DrawIcon(canvas, resources, IconActive, activeBounds, resources.Palette.Success); err != nil {
			return err
		}
	}
	if w.Focused() {
		return DrawFocusRing(canvas, resources, insetRect(bounds, 2), resources.Metrics.RowRadius)
	}
	return nil
}
