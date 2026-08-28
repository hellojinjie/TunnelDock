package ui

import "github.com/tailscale/walk"

type dialogValidationState struct {
	Accepted   bool
	Message    string
	FocusField string
}

func (s *dialogValidationState) Reject(message, focusField string) {
	s.Accepted = false
	s.Message = message
	s.FocusField = focusField
}

func (s *dialogValidationState) Accept() {
	s.Accepted = true
	s.Message = ""
	s.FocusField = ""
}

type DialogSpec struct {
	Title       string
	Description string
	PrimaryText string
	Size        walk.Size
	MinSize     walk.Size
	Resizable   bool
	Destructive bool
}

type DialogShell struct {
	*walk.Dialog
	Content     *walk.Composite
	Validation  *walk.Label
	Primary     *walk.PushButton
	Cancel      *walk.PushButton
	card        *Card
	env         *UIEnvironment
	unsubscribe func()
	disposed    bool
}

func NewDialogShell(owner walk.Form, env *UIEnvironment, spec DialogSpec) (*DialogShell, error) {
	var dialog *walk.Dialog
	var err error
	if spec.Resizable {
		dialog, err = walk.NewDialog(owner)
	} else {
		dialog, err = walk.NewDialogWithFixedSize(owner)
	}
	if err != nil {
		return nil, err
	}
	shell := &DialogShell{Dialog: dialog, env: env}
	fail := func(cause error) (*DialogShell, error) {
		shell.Dispose()
		return nil, cause
	}
	if err := dialog.SetTitle(spec.Title); err != nil {
		return fail(err)
	}
	if spec.Size.Width > 0 || spec.Size.Height > 0 {
		if err := dialog.SetSize(spec.Size); err != nil {
			return fail(err)
		}
	}
	if spec.MinSize.Width > 0 || spec.MinSize.Height > 0 {
		if err := dialog.SetMinMaxSize(spec.MinSize, walk.Size{}); err != nil {
			return fail(err)
		}
	}
	layout := walk.NewVBoxLayout()
	layout.SetMargins(walk.Margins{HNear: 18, VNear: 16, HFar: 18, VFar: 16})
	layout.SetSpacing(10)
	if err := dialog.SetLayout(layout); err != nil {
		return fail(err)
	}
	resources, err := env.Resources(dialog.DPI())
	if err != nil {
		return fail(err)
	}
	dialog.SetBackground(resources.WindowBrush)
	if spec.Description != "" {
		description, labelErr := walk.NewLabel(dialog)
		if labelErr != nil {
			return fail(labelErr)
		}
		_ = description.SetText(spec.Description)
		description.SetFont(resources.BodyFont)
	}
	shell.card, err = NewCard(dialog, env)
	if err != nil {
		return fail(err)
	}
	shell.Content = shell.card.Content
	shell.Validation, err = walk.NewLabel(dialog)
	if err != nil {
		return fail(err)
	}
	shell.Validation.SetFont(resources.CaptionFont)
	shell.Validation.SetTextColor(resources.Palette.Failure)
	shell.Validation.SetVisible(false)
	buttons, err := walk.NewComposite(dialog)
	if err != nil {
		return fail(err)
	}
	buttonLayout := walk.NewHBoxLayout()
	buttonLayout.SetMargins(walk.Margins{})
	buttonLayout.SetSpacing(8)
	if err := buttons.SetLayout(buttonLayout); err != nil {
		return fail(err)
	}
	spacer, err := walk.NewHSpacer(buttons)
	if err != nil {
		return fail(err)
	}
	_ = buttonLayout.SetStretchFactor(spacer, 1)
	shell.Cancel, err = walk.NewPushButton(buttons)
	if err != nil {
		return fail(err)
	}
	_ = shell.Cancel.SetText("Cancel")
	shell.Cancel.Clicked().Attach(dialog.Cancel)
	shell.Primary, err = walk.NewPushButton(buttons)
	if err != nil {
		return fail(err)
	}
	primaryText := spec.PrimaryText
	if primaryText == "" {
		primaryText = "OK"
	}
	_ = shell.Primary.SetText(primaryText)
	if err := dialog.SetDefaultButton(shell.Primary); err != nil {
		return fail(err)
	}
	if err := dialog.SetCancelButton(shell.Cancel); err != nil {
		return fail(err)
	}
	if err := env.ApplyNativeFont(dialog, dialog.DPI()); err != nil {
		return fail(err)
	}
	shell.Validation.SetFont(resources.CaptionFont)
	ApplyWindowAppearance(dialog, env.Appearance())
	shell.unsubscribe = env.Subscribe(func(appearance Appearance) {
		if refreshed, resourceErr := env.Resources(dialog.DPI()); resourceErr == nil {
			dialog.SetBackground(refreshed.WindowBrush)
			shell.Validation.SetTextColor(refreshed.Palette.Failure)
			ApplyWindowAppearance(dialog, appearance)
		}
	})
	return shell, nil
}

func (s *DialogShell) SetValidation(message string, field walk.Widget) {
	_ = s.Validation.SetText(message)
	s.Validation.SetVisible(message != "")
	if message != "" && field != nil {
		_ = field.SetFocus()
	}
}

func (s *DialogShell) Dispose() {
	if s == nil || s.disposed {
		return
	}
	s.disposed = true
	if s.unsubscribe != nil {
		s.unsubscribe()
		s.unsubscribe = nil
	}
	if s.card != nil {
		s.card.Dispose()
		s.card = nil
	}
	s.Dialog.Dispose()
}
