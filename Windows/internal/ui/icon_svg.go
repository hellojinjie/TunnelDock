package ui

import (
	"bytes"
	"embed"
	"fmt"
	"image"
	"image/color"
	"strings"

	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
)

// Fluent UI System Icons 1.1.337, licensed under MIT.
// Source: https://github.com/microsoft/fluentui-system-icons/tree/1.1.337
//
//go:embed icons/fluent/*.svg
var fluentIconFiles embed.FS

var fluentIconNames = [...]string{
	"stack_20_regular.svg",
	"server_20_regular.svg",
	"warning_20_regular.svg",
	"plug_connected_20_regular.svg",
	"add_20_regular.svg",
	"document_edit_20_regular.svg",
	"arrow_sync_20_regular.svg",
	"settings_20_regular.svg",
	"search_20_regular.svg",
	"globe_20_regular.svg",
	"more_circle_20_regular.svg",
	"chevron_right_20_regular.svg",
	"chevron_down_20_regular.svg",
	"circle_20_regular.svg",
	"arrow_sync_circle_20_regular.svg",
	"checkmark_circle_20_filled.svg",
	"error_circle_20_filled.svg",
}

func fluentSVG(kind IconKind) ([]byte, bool) {
	if int(kind) < 0 || int(kind) >= len(fluentIconNames) {
		return nil, false
	}
	data, err := fluentIconFiles.ReadFile("icons/fluent/" + fluentIconNames[kind])
	return data, err == nil
}

func rasterizeFluentIcon(kind IconKind, width, height int, tint color.RGBA) (*image.RGBA, error) {
	if width <= 0 || height <= 0 {
		return nil, fmt.Errorf("invalid icon size %dx%d", width, height)
	}
	data, ok := fluentSVG(kind)
	if !ok {
		return nil, fmt.Errorf("no Fluent SVG for icon %d", kind)
	}
	hexColor := fmt.Sprintf("#%02X%02X%02X", tint.R, tint.G, tint.B)
	tinted := strings.ReplaceAll(string(data), "#212121", hexColor)
	icon, err := oksvg.ReadIconStream(bytes.NewBufferString(tinted))
	if err != nil {
		return nil, fmt.Errorf("parse Fluent SVG for icon %d: %w", kind, err)
	}
	icon.SetTarget(0, 0, float64(width), float64(height))
	result := image.NewRGBA(image.Rect(0, 0, width, height))
	scanner := rasterx.NewScannerGV(width, height, result, result.Bounds())
	rasterizer := rasterx.NewDasher(width, height, scanner)
	icon.Draw(rasterizer, 1)
	return result, nil
}
