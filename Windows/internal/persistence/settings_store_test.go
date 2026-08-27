package persistence

import (
	"path/filepath"
	"testing"
)

func TestSettingsStoreDefaultsTrayIconToVisible(t *testing.T) {
	store := NewSettingsStore(filepath.Join(t.TempDir(), "settings.json"))
	settings, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if !settings.ShowTrayIcon {
		t.Fatal("Load() ShowTrayIcon = false, want true for missing settings file")
	}
}

func TestSettingsStoreRoundTripsTrayIcon(t *testing.T) {
	store := NewSettingsStore(filepath.Join(t.TempDir(), "settings.json"))
	if err := store.Save(Settings{ShowTrayIcon: false}); err != nil {
		t.Fatalf("Save() error: %v", err)
	}
	settings, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if settings.ShowTrayIcon {
		t.Fatal("Load() ShowTrayIcon = true, want saved false")
	}
}
