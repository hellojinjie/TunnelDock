package model

import (
	"errors"
	"regexp"
	"testing"
	"time"
)

func TestTunnelDefinitionDisplayName(t *testing.T) {
	tests := []struct {
		name       string
		explicit   *string
		localPort  uint16
		remotePort uint16
		want       string
	}{
		{name: "matching unnamed ports", localPort: 8888, remotePort: 8888, want: "8888"},
		{name: "different unnamed ports", localPort: 18888, remotePort: 8888, want: "18888 → 8888"},
		{name: "explicit name", explicit: stringPointer("Jupyter"), localPort: 18888, remotePort: 8888, want: "Jupyter"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			definition := TunnelDefinition{
				Name:       tt.explicit,
				LocalPort:  tt.localPort,
				RemotePort: tt.remotePort,
			}

			if got := definition.DisplayName(); got != tt.want {
				t.Fatalf("DisplayName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTunnelRuntimeUsesDefinitionDisplayNameRules(t *testing.T) {
	runtime := TunnelRuntime{
		Definition: TunnelDefinition{LocalPort: 18888, RemotePort: 8888},
	}

	if got, want := runtime.DisplayName(), "18888 → 8888"; got != want {
		t.Fatalf("DisplayName() = %q, want %q", got, want)
	}
}

func TestTunnelDefinitionValidation(t *testing.T) {
	valid := validDefinition()
	tests := []struct {
		name  string
		alter func(*TunnelDefinition)
		want  error
	}{
		{name: "missing host alias", alter: func(d *TunnelDefinition) { d.HostAlias = "" }, want: ErrRequired},
		{name: "missing remote host", alter: func(d *TunnelDefinition) { d.RemoteHost = "  " }, want: ErrRequired},
		{name: "missing local address", alter: func(d *TunnelDefinition) { d.LocalAddress = "" }, want: ErrRequired},
		{name: "zero remote port", alter: func(d *TunnelDefinition) { d.RemotePort = 0 }, want: ErrInvalidPort},
		{name: "zero local port", alter: func(d *TunnelDefinition) { d.LocalPort = 0 }, want: ErrInvalidPort},
		{name: "newline in alias", alter: func(d *TunnelDefinition) { d.HostAlias = "gpu\nHost attacker" }, want: ErrInvalidText},
		{name: "nul in remote host", alter: func(d *TunnelDefinition) { d.RemoteHost = "host\x00name" }, want: ErrInvalidText},
		{name: "control character in local address", alter: func(d *TunnelDefinition) { d.LocalAddress = "127.0.0.1\x1f" }, want: ErrInvalidText},
		{name: "control character in name", alter: func(d *TunnelDefinition) { d.Name = stringPointer("bad\x7fname") }, want: ErrInvalidText},
	}

	if err := valid.Validate(); err != nil {
		t.Fatalf("valid definition rejected: %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			definition := valid
			tt.alter(&definition)
			if err := definition.Validate(); !errors.Is(err, tt.want) {
				t.Fatalf("Validate() error = %v, want error matching %v", err, tt.want)
			}
		})
	}
}

func TestNewUUIDv4ReturnsCanonicalVersion4UUID(t *testing.T) {
	canonical := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	seen := make(map[string]struct{})

	for range 32 {
		id, err := NewUUIDv4()
		if err != nil {
			t.Fatalf("NewUUIDv4() error: %v", err)
		}
		if !canonical.MatchString(id) {
			t.Fatalf("NewUUIDv4() = %q, want canonical UUIDv4", id)
		}
		if _, duplicate := seen[id]; duplicate {
			t.Fatalf("NewUUIDv4() returned duplicate %q", id)
		}
		seen[id] = struct{}{}
	}
}

func validDefinition() TunnelDefinition {
	return TunnelDefinition{
		ID:           "00000000-0000-4000-8000-000000000001",
		HostAlias:    "gpu",
		RemoteHost:   "127.0.0.1",
		RemotePort:   8888,
		LocalAddress: "127.0.0.1",
		LocalPort:    8888,
		WebProtocol:  TunnelProtocolHTTP,
		CreatedAt:    time.Unix(0, 0).UTC(),
		UpdatedAt:    time.Unix(0, 0).UTC(),
	}
}

func stringPointer(value string) *string {
	return &value
}
