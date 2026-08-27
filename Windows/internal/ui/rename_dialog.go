package ui

import "github.com/tailscale/walk"

func promptTunnelRename(owner walk.Form, initial string) (string, bool, error) {
	dialog, err := walk.NewDialogWithFixedSize(owner)
	if err != nil {
		return "", false, err
	}
	defer dialog.Dispose()
	if err := dialog.SetTitle("Rename Tunnel"); err != nil {
		return "", false, err
	}
	if err := dialog.SetSize(walk.Size{Width: 360, Height: 140}); err != nil {
		return "", false, err
	}
	layout := walk.NewVBoxLayout()
	if err := dialog.SetLayout(layout); err != nil {
		return "", false, err
	}
	label, err := walk.NewLabel(dialog)
	if err != nil {
		return "", false, err
	}
	_ = label.SetText("Tunnel name")
	name, err := walk.NewLineEdit(dialog)
	if err != nil {
		return "", false, err
	}
	if err := name.SetText(initial); err != nil {
		return "", false, err
	}
	buttons, err := walk.NewComposite(dialog)
	if err != nil {
		return "", false, err
	}
	buttons.SetLayout(walk.NewHBoxLayout())
	ok, err := walk.NewPushButton(buttons)
	if err != nil {
		return "", false, err
	}
	_ = ok.SetText("Rename")
	ok.Clicked().Attach(dialog.Accept)
	cancel, err := walk.NewPushButton(buttons)
	if err != nil {
		return "", false, err
	}
	_ = cancel.SetText("Cancel")
	cancel.Clicked().Attach(dialog.Cancel)
	if dialog.Run() != walk.DlgCmdOK {
		return "", false, nil
	}
	return name.Text(), true, nil
}
