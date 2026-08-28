package ui

import "github.com/tailscale/walk"

// Walk accepts integer point sizes only. 11pt is the smallest size that meets
// the requested 10.5pt minimum on every native control.
const standardTextPointSize = 11

// ApplyStandardTextScale explicitly applies the standard font to every native
// control, including both panes of split windows. This avoids relying on a
// particular control's parent-font inheritance behavior.
func ApplyStandardTextScale(window walk.Window) error {
	font, err := walk.NewFont("MS Shell Dlg 2", standardTextPointSize, walk.FontNormal)
	if err != nil {
		return err
	}
	applyTextScale(window, font)
	return nil
}

func applyTextScale(window walk.Window, font *walk.Font) {
	window.SetFont(font)
	container, isContainer := window.(walk.Container)
	if !isContainer {
		return
	}
	children := container.Children()
	for index := 0; index < children.Len(); index++ {
		applyTextScale(children.At(index), font)
	}
}
