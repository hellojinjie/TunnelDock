package ui

import (
	"math"
	"testing"

	"github.com/tailscale/walk"
)

func TestPaletteForProvidesDistinctReadableThemes(t *testing.T) {
	light := PaletteFor(AppearanceLight)
	dark := PaletteFor(AppearanceDark)
	if light.Window == dark.Window || light.PrimaryText == dark.PrimaryText {
		t.Fatal("light and dark palettes must differ")
	}
	if got := contrastRatio(light.PrimaryText, light.Window); got < 4.5 {
		t.Fatalf("light primary contrast = %.2f, want at least 4.5", got)
	}
	if got := contrastRatio(dark.PrimaryText, dark.Window); got < 4.5 {
		t.Fatalf("dark primary contrast = %.2f, want at least 4.5", got)
	}
}

func TestMetricsForDPIScalesLogicalValues(t *testing.T) {
	at96 := MetricsForDPI(96)
	at144 := MetricsForDPI(144)
	if at96.CardRadius != 8 || at96.PageMargin != 24 {
		t.Fatalf("96 DPI metrics = %#v", at96)
	}
	if at144.CardRadius != 12 || at144.PageMargin != 36 {
		t.Fatalf("144 DPI metrics = %#v", at144)
	}
}

func contrastRatio(foreground, background walk.Color) float64 {
	lighter := relativeLuminance(foreground)
	darker := relativeLuminance(background)
	if lighter < darker {
		lighter, darker = darker, lighter
	}
	return (lighter + 0.05) / (darker + 0.05)
}

func relativeLuminance(color walk.Color) float64 {
	channel := func(value byte) float64 {
		component := float64(value) / 255
		if component <= 0.04045 {
			return component / 12.92
		}
		return math.Pow((component+0.055)/1.055, 2.4)
	}
	return 0.2126*channel(color.R()) + 0.7152*channel(color.G()) + 0.0722*channel(color.B())
}
