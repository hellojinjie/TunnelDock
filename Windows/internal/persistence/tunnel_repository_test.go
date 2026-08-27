package persistence

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/hellojinjie/TunnelDock/Windows/internal/model"
)

func TestRepositoryLoadsMacOSSavedTunnels(t *testing.T) {
	repository := NewTunnelRepository(filepath.Join("testdata", "macos-saved-tunnels.json"))
	tunnels, err := repository.Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if len(tunnels) != 1 || tunnels[0].ID != "00000000-0000-4000-8000-000000000001" {
		t.Fatalf("Load() = %#v, want macOS fixture tunnel", tunnels)
	}
	if tunnels[0].WebProtocol != model.TunnelProtocolHTTPS || tunnels[0].LastConnectedAt == nil {
		t.Fatalf("Load() lost protocol or lastConnectedAt: %#v", tunnels[0])
	}
}

func TestRepositoryRoundTripPreservesFieldsWithoutRuntimeState(t *testing.T) {
	path := filepath.Join(t.TempDir(), "saved-tunnels.json")
	repository := NewTunnelRepository(path)
	connectedAt := time.Date(2026, 8, 27, 2, 3, 4, 0, time.UTC)
	definition := model.TunnelDefinition{
		ID: "00000000-0000-4000-8000-000000000001", HostAlias: "gpu", Name: stringPtr("Jupyter"),
		RemoteHost: "127.0.0.1", RemotePort: 8888, LocalAddress: "127.0.0.1", LocalPort: 18888,
		WebProtocol: model.TunnelProtocolHTTPS, CreatedAt: connectedAt.Add(-time.Hour), UpdatedAt: connectedAt,
		LastConnectedAt: &connectedAt,
	}
	if err := repository.ReplaceAll([]model.TunnelDefinition{definition}); err != nil {
		t.Fatalf("ReplaceAll() error: %v", err)
	}
	got, err := repository.Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if !reflect.DeepEqual(got, []model.TunnelDefinition{definition}) {
		t.Fatalf("round trip = %#v, want %#v", got, definition)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"desiredConnection", "logLines", "pid", "state"} {
		if containsJSONKey(data, forbidden) {
			t.Fatalf("saved JSON contains runtime field %q: %s", forbidden, data)
		}
	}
}

func TestRepositoryUnsupportedSchemaLocksWritesUntilValidReload(t *testing.T) {
	path := filepath.Join(t.TempDir(), "saved-tunnels.json")
	if err := os.WriteFile(path, []byte(`{"schemaVersion":2,"tunnels":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	repository := NewTunnelRepository(path)
	if _, err := repository.Load(); !errors.Is(err, ErrUnsupportedSchema) {
		t.Fatalf("Load() error = %v, want ErrUnsupportedSchema", err)
	}
	if err := repository.ReplaceAll(nil); !errors.Is(err, ErrWriteLocked) {
		t.Fatalf("ReplaceAll() error = %v, want ErrWriteLocked", err)
	}
	if err := os.WriteFile(path, []byte(`{"schemaVersion":1,"tunnels":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := repository.Load(); err != nil {
		t.Fatalf("Load() valid replacement error: %v", err)
	}
	if err := repository.ReplaceAll(nil); err != nil {
		t.Fatalf("ReplaceAll() remained locked: %v", err)
	}
}

func TestRepositoryMalformedFileIsPreservedAndLocksWrites(t *testing.T) {
	path := filepath.Join(t.TempDir(), "saved-tunnels.json")
	original := []byte("not-json")
	if err := os.WriteFile(path, original, 0o600); err != nil {
		t.Fatal(err)
	}
	repository := NewTunnelRepository(path)
	if _, err := repository.Load(); !errors.Is(err, ErrMalformedFile) {
		t.Fatalf("Load() error = %v, want ErrMalformedFile", err)
	}
	if err := repository.ReplaceAll(nil); !errors.Is(err, ErrWriteLocked) {
		t.Fatalf("ReplaceAll() error = %v, want ErrWriteLocked", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(original) {
		t.Fatalf("malformed file changed to %q", got)
	}
}

func stringPtr(value string) *string { return &value }

func containsJSONKey(data []byte, key string) bool {
	return bytes.Contains(data, []byte(`"`+key+`"`))
}
