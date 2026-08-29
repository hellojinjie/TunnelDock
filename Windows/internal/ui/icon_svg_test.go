package ui

import (
	"bytes"
	"image/color"
	"testing"
)

func TestEveryIconHasEmbeddedFluentSVG(t *testing.T) {
	for kind := IconAllTunnels; kind <= IconFailed; kind++ {
		data, ok := fluentSVG(kind)
		if !ok {
			t.Fatalf("icon %d has no Fluent SVG mapping", kind)
		}
		if !bytes.Contains(data, []byte("<svg")) || !bytes.Contains(data, []byte(`viewBox="0 0 20 20"`)) {
			t.Fatalf("icon %d is not a 20px SVG", kind)
		}
	}
}

func TestRasterizedFluentIconUsesRequestedColor(t *testing.T) {
	want := color.RGBA{R: 0x12, G: 0x78, B: 0xd4, A: 0xff}
	image, err := rasterizeFluentIcon(IconSettings, 40, 40, want)
	if err != nil {
		t.Fatal(err)
	}
	coloredPixels := 0
	for y := image.Bounds().Min.Y; y < image.Bounds().Max.Y; y++ {
		for x := image.Bounds().Min.X; x < image.Bounds().Max.X; x++ {
			pixel := color.RGBAModel.Convert(image.At(x, y)).(color.RGBA)
			if pixel.A == 0 {
				continue
			}
			coloredPixels++
			if pixel.A == 0xff && (pixel.R != want.R || pixel.G != want.G || pixel.B != want.B) {
				t.Fatalf("pixel at %d,%d = %#v, want RGB %#v", x, y, pixel, want)
			}
		}
	}
	if coloredPixels == 0 {
		t.Fatal("rasterized icon is fully transparent")
	}
}

func TestRasterizedFluentIconRejectsInvalidSize(t *testing.T) {
	if _, err := rasterizeFluentIcon(IconSettings, 0, 20, color.RGBA{}); err == nil {
		t.Fatal("expected invalid size error")
	}
}
