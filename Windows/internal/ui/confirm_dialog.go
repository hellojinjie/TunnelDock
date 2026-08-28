package ui

import "github.com/tailscale/walk"

func PresentDeleteConfirmation(name string) DialogSpec {
	return DialogSpec{
		Title: "Delete " + name + "?", Description: "This saved tunnel will be removed. This action cannot be undone.",
		PrimaryText: "Delete", Size: walk.Size{Width: 420, Height: 230}, Destructive: true,
	}
}

func ConfirmDeleteTunnel(owner walk.Form, env *UIEnvironment, name string) bool {
	shell, err := NewDialogShell(owner, env, PresentDeleteConfirmation(name))
	if err != nil {
		showDialogError(owner, err)
		return false
	}
	defer shell.Dispose()
	layout := walk.NewVBoxLayout()
	layout.SetMargins(walk.Margins{HNear: 14, VNear: 12, HFar: 14, VFar: 12})
	if err := shell.Content.SetLayout(layout); err != nil {
		showDialogError(owner, err)
		return false
	}
	warning, err := walk.NewLabel(shell.Content)
	if err != nil {
		showDialogError(owner, err)
		return false
	}
	_ = warning.SetText(name)
	accepted := false
	shell.Primary.Clicked().Attach(func() {
		accepted = true
		shell.Accept()
	})
	return shell.Run() == walk.DlgCmdOK && accepted
}
