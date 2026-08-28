package ui

type Rect struct {
	X      int
	Y      int
	Width  int
	Height int
}

func (r Rect) Contains(x, y int) bool {
	return r.Width > 0 && r.Height > 0 && x >= r.X && y >= r.Y && x < r.X+r.Width && y < r.Y+r.Height
}

type TunnelRowLayout struct {
	StateIcon     Rect
	Text          Rect
	StateLabel    Rect
	Browser       Rect
	Primary       Rect
	More          Rect
	PrimaryAction TunnelRowAction
}

func TunnelRowHeight(row TunnelRowPresentation, metrics Metrics) int {
	if row.ErrorText != "" {
		return metrics.TunnelErrorHeight
	}
	return metrics.TunnelRowHeight
}

func LayoutTunnelRow(width, height int, metrics Metrics, row TunnelRowPresentation) TunnelRowLayout {
	padding := scaleFromMetric(12, metrics)
	gap := scaleFromMetric(8, metrics)
	actionWidth := scaleFromMetric(86, metrics)
	stateWidth := scaleFromMetric(82, metrics)
	iconSize := metrics.IconSize
	actionHeight := metrics.ActionHeight
	centerY := (height - actionHeight) / 2
	right := width - padding

	layout := TunnelRowLayout{PrimaryAction: row.PrimaryAction}
	if row.ShowMore {
		layout.More = Rect{X: right - actionHeight, Y: centerY, Width: actionHeight, Height: actionHeight}
		right = layout.More.X - gap
	}
	if row.PrimaryAction != TunnelRowNoAction {
		layout.Primary = Rect{X: right - actionWidth, Y: centerY, Width: actionWidth, Height: actionHeight}
		right = layout.Primary.X - gap
	}
	if row.ShowBrowser {
		layout.Browser = Rect{X: right - actionHeight, Y: centerY, Width: actionHeight, Height: actionHeight}
		right = layout.Browser.X - gap
	}
	layout.StateLabel = Rect{X: right - stateWidth, Y: centerY, Width: stateWidth, Height: actionHeight}
	right = layout.StateLabel.X - gap
	layout.StateIcon = Rect{X: padding, Y: (height - iconSize) / 2, Width: iconSize, Height: iconSize}
	textX := layout.StateIcon.X + layout.StateIcon.Width + scaleFromMetric(12, metrics)
	layout.Text = Rect{X: textX, Y: padding, Width: max(1, right-textX), Height: max(1, height-2*padding)}
	return layout
}

func (layout TunnelRowLayout) HitTest(x, y int) TunnelRowAction {
	if layout.Browser.Contains(x, y) {
		return TunnelRowOpenBrowser
	}
	if layout.Primary.Contains(x, y) {
		return layout.PrimaryAction
	}
	if layout.More.Contains(x, y) {
		return TunnelRowMore
	}
	return TunnelRowNoAction
}

func scaleFromMetric(value int, metrics Metrics) int {
	if metrics.CardRadius == 0 {
		return value
	}
	return max(1, value*metrics.CardRadius/8)
}
