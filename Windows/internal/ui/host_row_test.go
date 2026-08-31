package ui

import "testing"

func TestHostRowHoverTransitionClearsHoverAndPressWhenPointerLeaves(t *testing.T) {
	next := hostRowHoverTransition{hovered: true, pressed: true}.pointerMoved(false)

	if next.hovered {
		t.Fatal("hovered remains true after pointer leaves the host row")
	}
	if next.pressed {
		t.Fatal("pressed remains true after pointer leaves the host row")
	}
	if !next.releaseCapture {
		t.Fatal("pointer leave does not release mouse capture")
	}
}
