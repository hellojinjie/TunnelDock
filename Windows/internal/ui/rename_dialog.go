package ui

import (
	"errors"

	"github.com/hellojinjie/TunnelDock/Windows/internal/model"
	"github.com/tailscale/walk"
)

func promptTunnelRename(owner walk.Form, env *UIEnvironment, initial string) (string, bool, error) {
	shell, err := NewDialogShell(owner, env, DialogSpec{
		Title: "Rename Tunnel", Description: "Choose the name shown in Recent Tunnels.",
		PrimaryText: "Rename", Size: walk.Size{Width: 400, Height: 240},
	})
	if err != nil {
		return "", false, err
	}
	defer shell.Dispose()
	layout := walk.NewVBoxLayout()
	layout.SetMargins(walk.Margins{HNear: 14, VNear: 12, HFar: 14, VFar: 12})
	layout.SetSpacing(8)
	if err := shell.Content.SetLayout(layout); err != nil {
		return "", false, err
	}
	label, err := walk.NewLabel(shell.Content)
	if err != nil {
		return "", false, err
	}
	_ = label.SetText("Tunnel name")
	name, err := walk.NewLineEdit(shell.Content)
	if err != nil {
		return "", false, err
	}
	if err := name.SetText(initial); err != nil {
		return "", false, err
	}
	accepted := false
	shell.Primary.Clicked().Attach(func() {
		value := name.Text()
		definition := model.TunnelDefinition{
			HostAlias: "validation", Name: &value, RemoteHost: "validation", RemotePort: 1,
			LocalAddress: "127.0.0.1", LocalPort: 1,
		}
		if validationErr := definition.Validate(); validationErr != nil {
			var fieldErr *model.ValidationError
			if errors.As(validationErr, &fieldErr) && fieldErr.Field == "name" {
				shell.SetValidation(validationErr.Error(), name)
				return
			}
		}
		accepted = true
		shell.SetValidation("", nil)
		shell.Accept()
	})
	_ = name.SetFocus()
	name.SetTextSelection(0, len(initial))
	if shell.Run() != walk.DlgCmdOK || !accepted {
		return "", false, nil
	}
	return name.Text(), true, nil
}
