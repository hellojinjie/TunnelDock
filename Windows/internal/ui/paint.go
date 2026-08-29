package ui

import (
	"image/color"

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

func DrawIcon(canvas *walk.Canvas, resources *UIResources, kind IconKind, bounds walk.Rectangle, tint walk.Color) error {
	key := iconBitmapKey{kind: kind, width: bounds.Width, height: bounds.Height, color: tint}
	bitmap := resources.iconBitmaps[key]
	if bitmap == nil {
		pixels, err := rasterizeFluentIcon(kind, bounds.Width, bounds.Height, color.RGBA{R: tint.R(), G: tint.G(), B: tint.B(), A: 0xff})
		if err != nil {
			return err
		}
		bitmap, err = walk.NewBitmapFromImageForDPI(pixels, canvas.DPI())
		if err != nil {
			return err
		}
		resources.iconBitmaps[key] = bitmap
	}
	return canvas.DrawImagePixels(bitmap, walk.Point{X: bounds.X, Y: bounds.Y})
}

type iconBitmapKey struct {
	kind          IconKind
	width, height int
	color         walk.Color
}

func insetRect(bounds walk.Rectangle, inset int) walk.Rectangle {
	return walk.Rectangle{X: bounds.X + inset, Y: bounds.Y + inset, Width: max(1, bounds.Width-inset*2), Height: max(1, bounds.Height-inset*2)}
}
