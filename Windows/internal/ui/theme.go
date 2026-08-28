package ui

import "github.com/tailscale/walk"

type Appearance uint8

const (
	AppearanceLight Appearance = iota
	AppearanceDark
)

type Palette struct {
	Window          walk.Color
	Sidebar         walk.Color
	Surface         walk.Color
	SurfaceHover    walk.Color
	SurfaceSelected walk.Color
	Border          walk.Color
	PrimaryText     walk.Color
	SecondaryText   walk.Color
	DisabledText    walk.Color
	Accent          walk.Color
	Success         walk.Color
	Connecting      walk.Color
	Warning         walk.Color
	Failure         walk.Color
	Focus           walk.Color
}

func PaletteFor(appearance Appearance) Palette {
	if appearance == AppearanceDark {
		return Palette{
			Window:          walk.RGB(30, 30, 32),
			Sidebar:         walk.RGB(39, 39, 42),
			Surface:         walk.RGB(44, 44, 46),
			SurfaceHover:    walk.RGB(58, 58, 60),
			SurfaceSelected: walk.RGB(23, 58, 94),
			Border:          walk.RGB(72, 72, 74),
			PrimaryText:     walk.RGB(245, 245, 247),
			SecondaryText:   walk.RGB(176, 176, 181),
			DisabledText:    walk.RGB(119, 119, 124),
			Accent:          walk.RGB(64, 156, 255),
			Success:         walk.RGB(48, 209, 88),
			Connecting:      walk.RGB(100, 180, 255),
			Warning:         walk.RGB(255, 159, 10),
			Failure:         walk.RGB(255, 69, 58),
			Focus:           walk.RGB(90, 170, 255),
		}
	}
	return Palette{
		Window:          walk.RGB(245, 245, 247),
		Sidebar:         walk.RGB(236, 236, 240),
		Surface:         walk.RGB(255, 255, 255),
		SurfaceHover:    walk.RGB(242, 242, 245),
		SurfaceSelected: walk.RGB(220, 235, 255),
		Border:          walk.RGB(214, 214, 218),
		PrimaryText:     walk.RGB(29, 29, 31),
		SecondaryText:   walk.RGB(110, 110, 115),
		DisabledText:    walk.RGB(161, 161, 166),
		Accent:          walk.RGB(0, 122, 255),
		Success:         walk.RGB(40, 180, 75),
		Connecting:      walk.RGB(0, 122, 255),
		Warning:         walk.RGB(215, 120, 0),
		Failure:         walk.RGB(215, 45, 40),
		Focus:           walk.RGB(0, 122, 255),
	}
}

type Metrics struct {
	PageMargin        int
	SidebarPadding    int
	CardRadius        int
	RowRadius         int
	HostRowHeight     int
	TunnelRowHeight   int
	TunnelErrorHeight int
	IconSize          int
	ActionHeight      int
	FocusWidth        int
}

func MetricsForDPI(dpi int) Metrics {
	return Metrics{
		PageMargin:        scaleMetric(24, dpi),
		SidebarPadding:    scaleMetric(10, dpi),
		CardRadius:        scaleMetric(8, dpi),
		RowRadius:         scaleMetric(6, dpi),
		HostRowHeight:     scaleMetric(32, dpi),
		TunnelRowHeight:   scaleMetric(68, dpi),
		TunnelErrorHeight: scaleMetric(88, dpi),
		IconSize:          scaleMetric(18, dpi),
		ActionHeight:      scaleMetric(30, dpi),
		FocusWidth:        max(1, scaleMetric(2, dpi)),
	}
}

func scaleMetric(value, dpi int) int {
	if dpi <= 0 {
		dpi = 96
	}
	return (value*dpi + 48) / 96
}
