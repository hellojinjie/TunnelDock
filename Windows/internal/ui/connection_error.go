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

func ShowConnectionError(owner walk.Form, err error, hostAlias string) {
	presentation := PresentConnectionError(err)
	dialog, createErr := walk.NewDialogWithFixedSize(owner)
	if createErr != nil {
		showDialogError(owner, err)
		return
	}
	defer dialog.Dispose()
	_ = dialog.SetTitle(presentation.Title)
	_ = dialog.SetSize(walk.Size{Width: 620, Height: 360})
	if createErr = dialog.SetLayout(walk.NewVBoxLayout()); createErr != nil {
		showDialogError(owner, err)
		return
	}
	for _, value := range []string{presentation.Summary, presentation.Action, "Technical details:"} {
		label, labelErr := walk.NewLabel(dialog)
		if labelErr != nil {
			showDialogError(owner, err)
			return
		}
		_ = label.SetText(value)
	}
	details, detailsErr := walk.NewTextEdit(dialog)
	if detailsErr != nil {
		showDialogError(owner, err)
		return
	}
	_ = details.SetReadOnly(true)
	_ = details.SetText(presentation.Details)
	_ = details.SetMinMaxSize(walk.Size{Width: 560, Height: 160}, walk.Size{Width: 560, Height: 160})
	buttons, buttonsErr := walk.NewComposite(dialog)
	if buttonsErr != nil {
		showDialogError(owner, err)
		return
	}
	_ = buttons.SetLayout(walk.NewHBoxLayout())
	if presentation.RequiresInteractiveSSH && sshclient.CanStartInteractiveSSH(hostAlias) {
		terminalButton, terminalErr := walk.NewPushButton(buttons)
		if terminalErr != nil {
			showDialogError(owner, err)
			return
		}
		_ = terminalButton.SetText("Open Terminal: " + sshclient.InteractiveSSHCommand(hostAlias))
		terminalButton.Clicked().Attach(func() {
			if startErr := sshclient.StartInteractiveSSH(hostAlias); startErr != nil {
				showDialogError(dialog, startErr)
				return
			}
			dialog.Accept()
		})
	}
	closeButton, buttonErr := walk.NewPushButton(buttons)
	if buttonErr != nil {
		showDialogError(owner, err)
		return
	}
	_ = closeButton.SetText("Close")
	closeButton.Clicked().Attach(dialog.Accept)
	if createErr = ApplyStandardTextScale(dialog); createErr != nil {
		showDialogError(owner, err)
		return
	}
	_ = dialog.Run()
}
