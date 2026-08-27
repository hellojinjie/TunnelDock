package model

import (
	"crypto/rand"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"
)

type TunnelProtocol string

const (
	TunnelProtocolHTTP  TunnelProtocol = "http"
	TunnelProtocolHTTPS TunnelProtocol = "https"
)

var (
	ErrRequired    = errors.New("required field is empty")
	ErrInvalidText = errors.New("field contains a control character")
	ErrInvalidPort = errors.New("port must be between 1 and 65535")
)

type ValidationError struct {
	Field string
	Err   error
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %v", e.Field, e.Err)
}

func (e *ValidationError) Unwrap() error {
	return e.Err
}

type TunnelDefinition struct {
	ID              string         `json:"id"`
	HostAlias       string         `json:"hostAlias"`
	Name            *string        `json:"name,omitempty"`
	RemoteHost      string         `json:"remoteHost"`
	RemotePort      uint16         `json:"remotePort"`
	LocalAddress    string         `json:"localAddress"`
	LocalPort       uint16         `json:"localPort"`
	WebProtocol     TunnelProtocol `json:"webProtocol"`
	CreatedAt       time.Time      `json:"createdAt"`
	UpdatedAt       time.Time      `json:"updatedAt"`
	LastConnectedAt *time.Time     `json:"lastConnectedAt,omitempty"`
}

func (d TunnelDefinition) DisplayName() string {
	if d.Name != nil && *d.Name != "" {
		return *d.Name
	}
	if d.LocalPort == d.RemotePort {
		return fmt.Sprintf("%d", d.RemotePort)
	}
	return fmt.Sprintf("%d → %d", d.LocalPort, d.RemotePort)
}

func (d TunnelDefinition) Validate() error {
	for _, field := range []struct {
		name     string
		value    string
		required bool
	}{
		{name: "hostAlias", value: d.HostAlias, required: true},
		{name: "remoteHost", value: d.RemoteHost, required: true},
		{name: "localAddress", value: d.LocalAddress, required: true},
	} {
		if err := validateText(field.value, field.required); err != nil {
			return &ValidationError{Field: field.name, Err: err}
		}
	}
	if d.Name != nil {
		if err := validateText(*d.Name, false); err != nil {
			return &ValidationError{Field: "name", Err: err}
		}
	}
	if d.RemotePort == 0 {
		return &ValidationError{Field: "remotePort", Err: ErrInvalidPort}
	}
	if d.LocalPort == 0 {
		return &ValidationError{Field: "localPort", Err: ErrInvalidPort}
	}
	return nil
}

func validateText(value string, required bool) error {
	if required && strings.TrimSpace(value) == "" {
		return ErrRequired
	}
	if strings.ContainsFunc(value, unicode.IsControl) {
		return ErrInvalidText
	}
	return nil
}

func NewUUIDv4() (string, error) {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", fmt.Errorf("generate UUIDv4: %w", err)
	}

	value[6] = value[6]&0x0f | 0x40
	value[8] = value[8]&0x3f | 0x80
	return fmt.Sprintf(
		"%08x-%04x-%04x-%04x-%012x",
		value[0:4], value[4:6], value[6:8], value[8:10], value[10:16],
	), nil
}
