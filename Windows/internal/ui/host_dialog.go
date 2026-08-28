package ui

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"github.com/tailscale/walk"
)

type SSHHostInput struct {
	Alias    string
	Hostname string
	User     string
	Port     uint16
}

// Validate keeps the config-writing boundary independent from native controls,
// so both the dialog and the runtime can reject malformed Host blocks.
func (input SSHHostInput) Validate() error {
	if input.Alias == "" || input.Hostname == "" {
		return fmt.Errorf("Host and HostName are required")
	}
	for name, value := range map[string]string{"Host": input.Alias, "HostName": input.Hostname, "User": input.User} {
		if strings.ContainsFunc(value, unicode.IsControl) {
			return fmt.Errorf("%s contains a control character", name)
		}
	}
	if strings.IndexFunc(input.Alias, unicode.IsSpace) >= 0 {
		return fmt.Errorf("Host cannot contain whitespace")
	}
	if input.Port == 0 {
		return fmt.Errorf("Port must be between 1 and 65535")
	}
	return nil
}

// ConfigBlock returns the minimal OpenSSH block created by Quick Add Host.
func (input SSHHostInput) ConfigBlock() string {
	block := fmt.Sprintf("Host %s\n    HostName %s\n    Port %d\n", input.Alias, input.Hostname, input.Port)
	if input.User != "" {
		block += fmt.Sprintf("    User %s\n", input.User)
	}
	return block
}

// ShowAddHostDialog implements the compact native equivalent of the macOS
// Quick Add Host sheet. The callback owns persistence and config reloading.
func ShowAddHostDialog(owner walk.Form, submit func(SSHHostInput) error) {
	dialog, err := walk.NewDialogWithFixedSize(owner)
	if err != nil {
		showDialogError(owner, err)
		return
	}
	defer dialog.Dispose()
	_ = dialog.SetTitle("Add SSH Host")
	_ = dialog.SetSize(walk.Size{Width: 400, Height: 260})
	if err := dialog.SetLayout(walk.NewVBoxLayout()); err != nil {
		showDialogError(owner, err)
		return
	}
	fields := make(map[string]*walk.LineEdit)
	for _, field := range []struct {
		label string
		key   string
		value string
	}{
		{"Host", "alias", ""},
		{"HostName", "hostname", ""},
		{"User", "user", ""},
		{"Port", "port", "22"},
	} {
		input, err := addDialogField(dialog, field.label, field.value)
		if err != nil {
			showDialogError(owner, err)
			return
		}
		fields[field.key] = input
	}
	buttons, err := walk.NewComposite(dialog)
	if err != nil {
		showDialogError(owner, err)
		return
	}
	_ = buttons.SetLayout(walk.NewHBoxLayout())
	add, err := walk.NewPushButton(buttons)
	if err != nil {
		showDialogError(owner, err)
		return
	}
	_ = add.SetText("Add Host")
	add.Clicked().Attach(dialog.Accept)
	cancel, err := walk.NewPushButton(buttons)
	if err != nil {
		showDialogError(owner, err)
		return
	}
	_ = cancel.SetText("Cancel")
	cancel.Clicked().Attach(dialog.Cancel)
	if err := ApplyStandardTextScale(dialog); err != nil {
		showDialogError(owner, err)
		return
	}
	if dialog.Run() != walk.DlgCmdOK {
		return
	}
	input, err := parseSSHHostInput(fields)
	if err != nil {
		showDialogError(owner, err)
		return
	}
	if err := submit(input); err != nil {
		showDialogError(owner, err)
	}
}

func parseSSHHostInput(fields map[string]*walk.LineEdit) (SSHHostInput, error) {
	alias := strings.TrimSpace(fields["alias"].Text())
	hostname := strings.TrimSpace(fields["hostname"].Text())
	user := strings.TrimSpace(fields["user"].Text())
	port, err := strconv.ParseUint(strings.TrimSpace(fields["port"].Text()), 10, 16)
	if err != nil || port == 0 {
		return SSHHostInput{}, fmt.Errorf("Port must be between 1 and 65535")
	}
	input := SSHHostInput{Alias: alias, Hostname: hostname, User: user, Port: uint16(port)}
	if err := input.Validate(); err != nil {
		return SSHHostInput{}, err
	}
	return input, nil
}

func showDialogError(owner walk.Form, err error) {
	walk.MsgBox(owner, "TunnelDock", err.Error(), walk.MsgBoxOK|walk.MsgBoxIconError)
}
