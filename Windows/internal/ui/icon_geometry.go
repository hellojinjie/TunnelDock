package ui

import "math"

const (
	iconViewBox                 = 24.0
	defaultIconStrokeWidth96DPI = 1
)

type iconPoint struct {
	X float64
	Y float64
}

type iconRect struct {
	X      float64
	Y      float64
	Width  float64
	Height float64
	Radius float64
}

type iconPath struct {
	Points []iconPoint
	Closed bool
}

type iconGeometry struct {
	Paths                 []iconPath
	Ellipses              []iconRect
	RoundedRects          []iconRect
	FilledEllipses        []iconRect
	InversePaths          []iconPath
	InverseFilledEllipses []iconRect
}

func buildIconGeometry(kind IconKind) iconGeometry {
	var geometry iconGeometry
	path := func(points ...iconPoint) { geometry.Paths = append(geometry.Paths, iconPath{Points: points}) }
	ellipse := func(x, y, width, height float64) {
		geometry.Ellipses = append(geometry.Ellipses, iconRect{X: x, Y: y, Width: width, Height: height})
	}
	filledEllipse := func(x, y, width, height float64) {
		geometry.FilledEllipses = append(geometry.FilledEllipses, iconRect{X: x, Y: y, Width: width, Height: height})
	}
	roundedRect := func(x, y, width, height, radius float64) {
		geometry.RoundedRects = append(geometry.RoundedRects, iconRect{X: x, Y: y, Width: width, Height: height, Radius: radius})
	}

	switch kind {
	case IconAllTunnels:
		roundedRect(6, 4, 14, 14, 2)
		roundedRect(4, 7, 14, 14, 2)
	case IconServer:
		for _, y := range []float64{3, 10, 17} {
			roundedRect(3, y, 18, 5, 1.5)
			filledEllipse(6, y+1.5, 2, 2)
		}
	case IconWarning:
		geometry.Paths = append(geometry.Paths, iconPath{Points: []iconPoint{{12, 3}, {22, 21}, {2, 21}}, Closed: true})
		path(iconPoint{12, 8}, iconPoint{12, 15})
		filledEllipse(11, 17, 2, 2)
	case IconActive:
		path(iconPoint{5, 18}, iconPoint{12, 5}, iconPoint{19, 18}, iconPoint{5, 18})
		for _, point := range []iconPoint{{5, 18}, {12, 5}, {19, 18}} {
			filledEllipse(point.X-2, point.Y-2, 4, 4)
		}
	case IconPlus:
		path(iconPoint{4, 12}, iconPoint{20, 12})
		path(iconPoint{12, 4}, iconPoint{12, 20})
	case IconEdit:
		roundedRect(3, 5, 14, 16, 2)
		path(iconPoint{8, 17}, iconPoint{18, 7})
		path(iconPoint{10, 19}, iconPoint{20, 9})
		path(iconPoint{18, 7}, iconPoint{20, 9})
		path(iconPoint{8, 17}, iconPoint{7, 20}, iconPoint{10, 19})
	case IconRefresh:
		geometry.Paths = append(geometry.Paths, refreshPaths()...)
	case IconSettings:
		gear := make([]iconPoint, 0, 16)
		for index := 0; index < 16; index++ {
			angle := -math.Pi/2 + float64(index)*math.Pi/8
			radius := 9.5
			if index%2 != 0 {
				radius = 7.5
			}
			gear = append(gear, iconPoint{X: 12 + math.Cos(angle)*radius, Y: 12 + math.Sin(angle)*radius})
		}
		geometry.Paths = append(geometry.Paths, iconPath{Points: gear, Closed: true})
		ellipse(9, 9, 6, 6)
	case IconSearch:
		ellipse(3, 3, 13, 13)
		path(iconPoint{15, 15}, iconPoint{21, 21})
	case IconBrowser:
		ellipse(2, 2, 20, 20)
		ellipse(8, 2, 8, 20)
		path(iconPoint{3, 12}, iconPoint{21, 12})
	case IconMore:
		ellipse(2, 2, 20, 20)
		for _, x := range []float64{7, 11, 15} {
			filledEllipse(x, 11, 2, 2)
		}
	case IconChevronRight:
		path(iconPoint{9, 5}, iconPoint{16, 12}, iconPoint{9, 19})
	case IconChevronDown:
		path(iconPoint{5, 9}, iconPoint{12, 16}, iconPoint{19, 9})
	case IconDisconnected:
		ellipse(4, 4, 16, 16)
	case IconConnecting:
		ellipse(2, 2, 20, 20)
		geometry.Paths = append(geometry.Paths, refreshPaths()...)
	case IconConnected:
		filledEllipse(2, 2, 20, 20)
		geometry.InversePaths = append(geometry.InversePaths, iconPath{Points: []iconPoint{{7, 12}, {10.5, 15.5}, {17.5, 8}}})
	case IconFailed:
		filledEllipse(2, 2, 20, 20)
		geometry.InversePaths = append(geometry.InversePaths, iconPath{Points: []iconPoint{{12, 6.5}, {12, 14}}})
		geometry.InverseFilledEllipses = append(geometry.InverseFilledEllipses, iconRect{X: 11, Y: 17, Width: 2, Height: 2})
	}

	return geometry
}

func refreshPaths() []iconPath {
	return []iconPath{
		{Points: arcPoints(12, 12, 8, -135, 35, 12)},
		{Points: arcPoints(12, 12, 8, 45, 215, 12)},
		{Points: []iconPoint{{18.6, 5.8}, {20.6, 4.7}}},
		{Points: []iconPoint{{18.6, 5.8}, {20.1, 7.8}}},
	}
}

func arcPoints(cx, cy, radius, startDegrees, endDegrees float64, segments int) []iconPoint {
	points := make([]iconPoint, segments+1)
	for index := 0; index <= segments; index++ {
		angle := (startDegrees + (endDegrees-startDegrees)*float64(index)/float64(segments)) * math.Pi / 180
		points[index] = iconPoint{X: cx + math.Cos(angle)*radius, Y: cy + math.Sin(angle)*radius}
	}
	return points
}
