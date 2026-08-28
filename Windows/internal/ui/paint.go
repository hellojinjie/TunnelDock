package ui

import "github.com/tailscale/walk"

type IconKind uint8

const (
	IconAllTunnels IconKind = iota
	IconServer
	IconWarning
	IconActive
	IconPlus
	IconEdit
	IconRefresh
	IconSettings
	IconSearch
	IconBrowser
	IconMore
	IconChevronRight
	IconChevronDown
	IconDisconnected
	IconConnecting
	IconConnected
	IconFailed
)

func FillRoundedSurface(canvas *walk.Canvas, brush walk.Brush, bounds walk.Rectangle, radius int) error {
	return canvas.FillRoundedRectanglePixels(brush, bounds, walk.Size{Width: radius * 2, Height: radius * 2})
}

func DrawFocusRing(canvas *walk.Canvas, resources *UIResources, bounds walk.Rectangle, radius int) error {
	return canvas.DrawRoundedRectanglePixels(resources.FocusPen, bounds, walk.Size{Width: radius * 2, Height: radius * 2})
}

func DrawTextEllipsized(canvas *walk.Canvas, text string, font *walk.Font, color walk.Color, bounds walk.Rectangle) error {
	return canvas.DrawTextPixels(text, font, color, bounds, walk.TextSingleLine|walk.TextVCenter|walk.TextEndEllipsis|walk.TextNoPrefix)
}

func DrawIcon(canvas *walk.Canvas, resources *UIResources, kind IconKind, bounds walk.Rectangle, color walk.Color) error {
	pen, err := walk.NewCosmeticPen(walk.PenSolid, color)
	if err != nil {
		return err
	}
	defer pen.Dispose()
	brush, err := walk.NewSolidColorBrush(color)
	if err != nil {
		return err
	}
	defer brush.Dispose()

	cx := bounds.X + bounds.Width/2
	cy := bounds.Y + bounds.Height/2
	left, right := bounds.X+2, bounds.X+bounds.Width-2
	top, bottom := bounds.Y+2, bounds.Y+bounds.Height-2
	switch kind {
	case IconActive, IconConnected:
		return canvas.FillEllipsePixels(brush, insetRect(bounds, 4))
	case IconDisconnected:
		return canvas.DrawEllipsePixels(pen, insetRect(bounds, 4))
	case IconConnecting, IconRefresh:
		if err := canvas.DrawEllipsePixels(pen, insetRect(bounds, 3)); err != nil {
			return err
		}
		return canvas.DrawLinePixels(pen, walk.Point{X: right - 3, Y: top}, walk.Point{X: right, Y: top + 4})
	case IconFailed, IconWarning:
		if err := canvas.DrawPolylinePixels(pen, []walk.Point{{X: cx, Y: top}, {X: right, Y: bottom}, {X: left, Y: bottom}, {X: cx, Y: top}}); err != nil {
			return err
		}
		return canvas.DrawLinePixels(pen, walk.Point{X: cx, Y: cy - 3}, walk.Point{X: cx, Y: cy + 3})
	case IconPlus:
		if err := canvas.DrawLinePixels(pen, walk.Point{X: left, Y: cy}, walk.Point{X: right, Y: cy}); err != nil {
			return err
		}
		return canvas.DrawLinePixels(pen, walk.Point{X: cx, Y: top}, walk.Point{X: cx, Y: bottom})
	case IconMore:
		for _, x := range []int{cx - 5, cx, cx + 5} {
			if err := canvas.FillEllipsePixels(brush, walk.Rectangle{X: x - 1, Y: cy - 1, Width: 3, Height: 3}); err != nil {
				return err
			}
		}
		return nil
	case IconChevronRight:
		return canvas.DrawPolylinePixels(pen, []walk.Point{{X: cx - 2, Y: top + 2}, {X: cx + 3, Y: cy}, {X: cx - 2, Y: bottom - 2}})
	case IconChevronDown:
		return canvas.DrawPolylinePixels(pen, []walk.Point{{X: left + 2, Y: cy - 2}, {X: cx, Y: cy + 3}, {X: right - 2, Y: cy - 2}})
	case IconSearch:
		if err := canvas.DrawEllipsePixels(pen, walk.Rectangle{X: left, Y: top, Width: bounds.Width - 7, Height: bounds.Height - 7}); err != nil {
			return err
		}
		return canvas.DrawLinePixels(pen, walk.Point{X: cx + 3, Y: cy + 3}, walk.Point{X: right, Y: bottom})
	case IconBrowser:
		if err := canvas.DrawEllipsePixels(pen, insetRect(bounds, 2)); err != nil {
			return err
		}
		if err := canvas.DrawLinePixels(pen, walk.Point{X: left, Y: cy}, walk.Point{X: right, Y: cy}); err != nil {
			return err
		}
		return canvas.DrawLinePixels(pen, walk.Point{X: cx, Y: top}, walk.Point{X: cx, Y: bottom})
	case IconEdit:
		if err := canvas.DrawLinePixels(pen, walk.Point{X: left + 2, Y: bottom - 2}, walk.Point{X: right - 1, Y: top + 1}); err != nil {
			return err
		}
		return canvas.DrawLinePixels(pen, walk.Point{X: left, Y: bottom}, walk.Point{X: left + 5, Y: bottom - 1})
	case IconSettings:
		if err := canvas.DrawEllipsePixels(pen, insetRect(bounds, 3)); err != nil {
			return err
		}
		return canvas.FillEllipsePixels(brush, insetRect(bounds, 7))
	case IconServer:
		if err := canvas.DrawRoundedRectanglePixels(pen, insetRect(bounds, 2), walk.Size{Width: 3, Height: 3}); err != nil {
			return err
		}
		return canvas.DrawLinePixels(pen, walk.Point{X: left, Y: cy}, walk.Point{X: right, Y: cy})
	case IconAllTunnels:
		for row := 0; row < 2; row++ {
			for column := 0; column < 2; column++ {
				rect := walk.Rectangle{X: left + column*(bounds.Width/2), Y: top + row*(bounds.Height/2), Width: 5, Height: 5}
				if err := canvas.DrawRoundedRectanglePixels(pen, rect, walk.Size{Width: 2, Height: 2}); err != nil {
					return err
				}
			}
		}
		return nil
	default:
		return nil
	}
}

func insetRect(bounds walk.Rectangle, inset int) walk.Rectangle {
	return walk.Rectangle{X: bounds.X + inset, Y: bounds.Y + inset, Width: max(1, bounds.Width-inset*2), Height: max(1, bounds.Height-inset*2)}
}
