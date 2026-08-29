package ui

import "testing"

func TestIconButtonPresentationUsesItsContainerBackground(t *testing.T) {
	tests := []struct {
		name string
		on   iconButtonSurface
		want iconButtonBase
	}{
		{name: "sidebar toolbar", on: iconButtonOnSidebar, want: iconButtonBaseSidebar},
		{name: "window header", on: iconButtonOnWindow, want: iconButtonBaseWindow},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := iconButtonPresentation(test.on, false, false, false, false)
			if got.Base != test.want {
				t.Fatalf("base = %d, want %d", got.Base, test.want)
			}
			if got.Overlay != iconButtonOverlayNone {
				t.Fatalf("overlay = %d, want none", got.Overlay)
			}
		})
	}
}

func TestIconButtonPresentationInsetsInteractiveSurface(t *testing.T) {
	hovered := iconButtonPresentation(iconButtonOnSidebar, true, false, false, false)
	if hovered.Overlay != iconButtonOverlayHover || hovered.OverlayInset != 2 {
		t.Fatalf("hovered presentation = %#v", hovered)
	}
	pressed := iconButtonPresentation(iconButtonOnSidebar, true, true, true, false)
	if pressed.Overlay != iconButtonOverlayPressed || pressed.OverlayInset != 2 {
		t.Fatalf("pressed presentation = %#v", pressed)
	}
}

func TestIconButtonPresentationShowsFocusOnlyForKeyboardNavigation(t *testing.T) {
	mouseFocus := iconButtonPresentation(iconButtonOnSidebar, true, true, true, false)
	if mouseFocus.ShowFocus {
		t.Fatal("mouse press must not draw the keyboard focus ring")
	}
	keyboardFocus := iconButtonPresentation(iconButtonOnSidebar, false, false, true, true)
	if !keyboardFocus.ShowFocus {
		t.Fatal("keyboard navigation must retain a visible focus ring")
	}
}
