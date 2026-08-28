package ui

import (
	"errors"

	"github.com/hellojinjie/TunnelDock/Windows/internal/sshclient"
	"github.com/hellojinjie/TunnelDock/Windows/internal/tunnel"
	"github.com/tailscale/walk"
)

type ConnectionErrorPresentation struct {
	Title                  string
	Summary                string
	Action                 string
	Details                string
	RequiresInteractiveSSH bool
}

func PresentConnectionError(err error) ConnectionErrorPresentation {
	var failure *sshclient.ConnectionFailure
	if errors.As(err, &failure) {
		return ConnectionErrorPresentation{
			Title: "Connection failed", Summary: failure.Error(), Action: failure.SuggestedAction(), Details: failure.Details(), RequiresInteractiveSSH: failure.RequiresInteractiveSSH(),
		}
	}
	if errors.Is(err, tunnel.ErrPortUnavailable) {
		return ConnectionErrorPresentation{
			Title: "Local port unavailable", Summary: "TunnelDock cannot listen on the selected Local Port.",
			Action: "Choose a different Local Port under Advanced, then try again.", Details: err.Error(),
		}
	}
	return ConnectionErrorPresentation{
		Title: "Connection failed", Summary: "TunnelDock could not start this tunnel.",
		Action: "Review the technical details below and verify the SSH Host settings.", Details: err.Error(),
	}
}

func ShowConnectionError(owner walk.Form, env *UIEnvironment, err error, hostAlias string) {
	presentation := PresentConnectionError(err)
	shell, createErr := NewDialogShell(owner, env, DialogSpec{
		Title: presentation.Title, Description: presentation.Summary,
		PrimaryText: "Close", Size: walk.Size{Width: 640, Height: 440},
	})
	if createErr != nil {
		showDialogError(owner, createErr)
		return
	}
	defer shell.Dispose()
	shell.Cancel.SetVisible(false)
	layout := walk.NewVBoxLayout()
	layout.SetMargins(walk.Margins{HNear: 14, VNear: 12, HFar: 14, VFar: 12})
	layout.SetSpacing(8)
	if createErr = shell.Content.SetLayout(layout); createErr != nil {
		showDialogError(owner, createErr)
		return
	}
	actionLabel, createErr := walk.NewLabel(shell.Content)
	if createErr != nil {
		showDialogError(owner, createErr)
		return
	}
	_ = actionLabel.SetText(presentation.Action)
	detailsLabel, createErr := walk.NewLabel(shell.Content)
	if createErr != nil {
		showDialogError(owner, createErr)
		return
	}
	_ = detailsLabel.SetText("Technical details")
	details, createErr := walk.NewTextEdit(shell.Content)
	if createErr != nil {
		showDialogError(owner, createErr)
		return
	}
	_ = details.SetReadOnly(true)
	_ = details.SetText(presentation.Details)
	_ = details.SetMinMaxSize(walk.Size{Height: 150}, walk.Size{})
	if presentation.RequiresInteractiveSSH && sshclient.CanStartInteractiveSSH(hostAlias) {
		terminalButton, terminalErr := walk.NewPushButton(shell.Content)
		if terminalErr != nil {
			showDialogError(owner, terminalErr)
			return
		}
		_ = terminalButton.SetText("Open SSH Terminal")
		terminalButton.SetToolTipText(sshclient.InteractiveSSHCommand(hostAlias))
		terminalButton.Clicked().Attach(func() {
			if startErr := sshclient.StartInteractiveSSH(hostAlias); startErr != nil {
				shell.SetValidation(startErr.Error(), terminalButton)
				return
			}
			shell.Accept()
		})
	}
	shell.Primary.Clicked().Attach(shell.Accept)
	shell.Run()
}
