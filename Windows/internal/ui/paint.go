package ui

import (
	"math"

	"github.com/tailscale/walk"
)

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
	brush, err := walk.NewSolidColorBrush(color)
	if err != nil {
		return err
	}
	defer brush.Dispose()
	pen, err := walk.NewGeometricPen(walk.PenSolid|walk.PenCapRound|walk.PenJoinRound, defaultIconStrokeWidth96DPI, brush)
	if err != nil {
		return err
	}
	defer pen.Dispose()
	inversePen, err := walk.NewGeometricPen(walk.PenSolid|walk.PenCapRound|walk.PenJoinRound, defaultIconStrokeWidth96DPI, resources.SurfaceBrush)
	if err != nil {
		return err
	}
	defer inversePen.Dispose()

	geometry := buildIconGeometry(kind)
	for _, rect := range geometry.RoundedRects {
		if err := canvas.DrawRoundedRectanglePixels(pen, mapIconRect(rect, bounds), mapIconSize(rect.Radius*2, rect.Radius*2, bounds)); err != nil {
			return err
		}
	}
	for _, rect := range geometry.Ellipses {
		if err := canvas.DrawEllipsePixels(pen, mapIconRect(rect, bounds)); err != nil {
			return err
		}
	}
	for _, rect := range geometry.FilledEllipses {
		if err := canvas.FillEllipsePixels(brush, mapIconRect(rect, bounds)); err != nil {
			return err
		}
	}
	for _, path := range geometry.Paths {
		if err := drawIconPath(canvas, pen, path, bounds); err != nil {
			return err
		}
	}
	for _, path := range geometry.InversePaths {
		if err := drawIconPath(canvas, inversePen, path, bounds); err != nil {
			return err
		}
	}
	for _, rect := range geometry.InverseFilledEllipses {
		if err := canvas.FillEllipsePixels(resources.SurfaceBrush, mapIconRect(rect, bounds)); err != nil {
			return err
		}
	}
	return nil
}

func drawIconPath(canvas *walk.Canvas, pen walk.Pen, path iconPath, bounds walk.Rectangle) error {
	if len(path.Points) < 2 {
		return nil
	}
	points := make([]walk.Point, 0, len(path.Points)+1)
	for _, point := range path.Points {
		points = append(points, mapIconPoint(point, bounds))
	}
	if path.Closed {
		points = append(points, points[0])
	}
	return canvas.DrawPolylinePixels(pen, points)
}

func mapIconPoint(point iconPoint, bounds walk.Rectangle) walk.Point {
	return walk.Point{
		X: bounds.X + int(math.Round(point.X/iconViewBox*float64(bounds.Width))),
		Y: bounds.Y + int(math.Round(point.Y/iconViewBox*float64(bounds.Height))),
	}
}

func mapIconRect(rect iconRect, bounds walk.Rectangle) walk.Rectangle {
	topLeft := mapIconPoint(iconPoint{X: rect.X, Y: rect.Y}, bounds)
	return walk.Rectangle{
		X:      topLeft.X,
		Y:      topLeft.Y,
		Width:  max(1, int(math.Round(rect.Width/iconViewBox*float64(bounds.Width)))),
		Height: max(1, int(math.Round(rect.Height/iconViewBox*float64(bounds.Height)))),
	}
}

func mapIconSize(width, height float64, bounds walk.Rectangle) walk.Size {
	return walk.Size{
		Width:  max(1, int(math.Round(width/iconViewBox*float64(bounds.Width)))),
		Height: max(1, int(math.Round(height/iconViewBox*float64(bounds.Height)))),
	}
}

func insetRect(bounds walk.Rectangle, inset int) walk.Rectangle {
	return walk.Rectangle{X: bounds.X + inset, Y: bounds.Y + inset, Width: max(1, bounds.Width-inset*2), Height: max(1, bounds.Height-inset*2)}
}
