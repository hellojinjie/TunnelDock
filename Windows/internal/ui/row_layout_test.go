package ui

import "testing"

func TestTunnelRowHeightExpandsForError(t *testing.T) {
	metrics := MetricsForDPI(96)
	normal := TunnelRowPresentation{ID: "normal"}
	failed := TunnelRowPresentation{ID: "failed", ErrorText: "Permission denied"}
	if got := TunnelRowHeight(normal, metrics); got != metrics.TunnelRowHeight {
		t.Fatalf("normal height = %d", got)
	}
	if got := TunnelRowHeight(failed, metrics); got != metrics.TunnelErrorHeight {
		t.Fatalf("failed height = %d", got)
	}
}

func TestTunnelRowHitTestSeparatesInlineActions(t *testing.T) {
	row := TunnelRowPresentation{ShowBrowser: true, ShowMore: true, PrimaryAction: TunnelRowDisconnect}
	layout := LayoutTunnelRow(720, 68, MetricsForDPI(96), row)
	if got := layout.HitTest(layout.Browser.X+1, layout.Browser.Y+1); got != TunnelRowOpenBrowser {
		t.Fatalf("browser hit = %v", got)
	}
	if got := layout.HitTest(layout.Primary.X+1, layout.Primary.Y+1); got != TunnelRowDisconnect {
		t.Fatalf("primary hit = %v", got)
	}
	if got := layout.HitTest(layout.More.X+1, layout.More.Y+1); got != TunnelRowMore {
		t.Fatalf("more hit = %v", got)
	}
}

func TestTunnelRowLayoutGivesTextPositiveWidthAtMinimumWindow(t *testing.T) {
	row := TunnelRowPresentation{ShowMore: true, PrimaryAction: TunnelRowConnect}
	layout := LayoutTunnelRow(560, 68, MetricsForDPI(96), row)
	if layout.Text.Width <= 0 || layout.Text.X+layout.Text.Width > layout.StateLabel.X {
		t.Fatalf("layout = %#v", layout)
	}
}
