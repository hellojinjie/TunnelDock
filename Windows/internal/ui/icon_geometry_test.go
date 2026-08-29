package ui

import "testing"

func TestPrimaryIconGeometryMatchesMacOSSemantics(t *testing.T) {
	tests := []struct {
		name              string
		kind              IconKind
		minPaths          int
		minClosedPaths    int
		minEllipses       int
		minRoundedRects   int
		minFilledEllipses int
	}{
		{name: "all tunnels rectangle stack", kind: IconAllTunnels, minRoundedRects: 2},
		{name: "server rack", kind: IconServer, minRoundedRects: 3, minFilledEllipses: 3},
		{name: "edit square and pencil", kind: IconEdit, minPaths: 2, minRoundedRects: 1},
		{name: "refresh double arrow", kind: IconRefresh, minPaths: 4},
		{name: "settings gear", kind: IconSettings, minClosedPaths: 1, minEllipses: 1},
		{name: "more in circle", kind: IconMore, minEllipses: 1, minFilledEllipses: 3},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			geometry := buildIconGeometry(test.kind)
			if len(geometry.Paths) < test.minPaths {
				t.Fatalf("paths = %d, want at least %d", len(geometry.Paths), test.minPaths)
			}
			closed := 0
			for _, path := range geometry.Paths {
				if path.Closed {
					closed++
				}
			}
			if closed < test.minClosedPaths {
				t.Fatalf("closed paths = %d, want at least %d", closed, test.minClosedPaths)
			}
			if len(geometry.Ellipses) < test.minEllipses {
				t.Fatalf("ellipses = %d, want at least %d", len(geometry.Ellipses), test.minEllipses)
			}
			if len(geometry.RoundedRects) < test.minRoundedRects {
				t.Fatalf("rounded rects = %d, want at least %d", len(geometry.RoundedRects), test.minRoundedRects)
			}
			if len(geometry.FilledEllipses) < test.minFilledEllipses {
				t.Fatalf("filled ellipses = %d, want at least %d", len(geometry.FilledEllipses), test.minFilledEllipses)
			}
		})
	}
}

func TestIconStrokeUsesOneDPIAwareLogicalPixel(t *testing.T) {
	if defaultIconStrokeWidth96DPI != 1 {
		t.Fatalf("stroke width = %d, want 1 logical pixel", defaultIconStrokeWidth96DPI)
	}
}

func TestEveryIconGeometryStaysInsideViewBox(t *testing.T) {
	for kind := IconAllTunnels; kind <= IconFailed; kind++ {
		geometry := buildIconGeometry(kind)
		for _, path := range geometry.Paths {
			for _, point := range path.Points {
				if point.X < 0 || point.X > iconViewBox || point.Y < 0 || point.Y > iconViewBox {
					t.Fatalf("kind %d point outside view box: %#v", kind, point)
				}
			}
		}
	}
}
