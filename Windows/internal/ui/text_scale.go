package ui

import "github.com/tailscale/walk"

const standardTextPointSize = 9

// ApplyStandardTextScale raises Walk's 8pt default by one point. Applying it
// to a container cascades to its complete native control tree.
func ApplyStandardTextScale(window walk.Window) error {
	font, err := walk.NewFont("MS Shell Dlg 2", standardTextPointSize, walk.FontNormal)
	if err != nil {
		return err
	}
	window.SetFont(font)
	return nil
}
