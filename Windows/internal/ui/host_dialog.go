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

func (input SSHHostInput) ConfigBlock() string {
	block := fmt.Sprintf("Host %s\n    HostName %s\n    Port %d\n", input.Alias, input.Hostname, input.Port)
	if input.User != "" {
		block += fmt.Sprintf("    User %s\n", input.User)
	}
	return block
}

// ShowAddHostDialog keeps parsing, persistence errors, and focus inside the
// modal sheet. The dialog accepts only after submit succeeds.
func ShowAddHostDialog(owner walk.Form, env *UIEnvironment, submit func(SSHHostInput) error) {
	shell, err := NewDialogShell(owner, env, DialogSpec{
		Title: "Add SSH Host", Description: "Add a compact Host entry to your OpenSSH config.",
		PrimaryText: "Add Host", Size: walk.Size{Width: 440, Height: 360},
	})
	if err != nil {
		showDialogError(owner, err)
		return
	}
	defer shell.Dispose()
	layout := walk.NewVBoxLayout()
	layout.SetMargins(walk.Margins{HNear: 14, VNear: 12, HFar: 14, VFar: 12})
	layout.SetSpacing(8)
	if err := shell.Content.SetLayout(layout); err != nil {
		showDialogError(owner, err)
		return
	}
	fields := make(map[string]*walk.LineEdit)
	for _, field := range []struct{ label, key, value string }{
		{"Host", "alias", ""}, {"HostName", "hostname", ""}, {"User", "user", ""}, {"Port", "port", "22"},
	} {
		input, fieldErr := addDialogField(shell.Content, field.label, field.value)
		if fieldErr != nil {
			showDialogError(owner, fieldErr)
			return
		}
		fields[field.key] = input
	}
	shell.Primary.Clicked().Attach(func() {
		input, parseErr := parseSSHHostInput(fields)
		if parseErr != nil {
			shell.SetValidation(parseErr.Error(), hostValidationField(fields, parseErr.Error()))
			return
		}
		if submit != nil {
			if submitErr := submit(input); submitErr != nil {
				shell.SetValidation(submitErr.Error(), fields["alias"])
				return
			}
		}
		shell.SetValidation("", nil)
		shell.Accept()
	})
	_ = fields["alias"].SetFocus()
	shell.Run()
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

func hostValidationField(fields map[string]*walk.LineEdit, message string) walk.Widget {
	switch {
	case strings.HasPrefix(message, "Port"):
		return fields["port"]
	case fields["alias"].Text() == "" || strings.HasPrefix(message, "Host ") || strings.Contains(message, "Host cannot"):
		return fields["alias"]
	case fields["hostname"].Text() == "" || strings.HasPrefix(message, "HostName"):
		return fields["hostname"]
	default:
		return fields["user"]
	}
}

func showDialogError(owner walk.Form, err error) {
	walk.MsgBox(owner, "TunnelDock", err.Error(), walk.MsgBoxOK|walk.MsgBoxIconError)
}
