package app

import (
	"testing"

	"github.com/hellojinjie/TunnelDock/Windows/internal/persistence"
)

type memoryTraySettings struct{ settings persistence.Settings }

func (s *memoryTraySettings) Load() (persistence.Settings, error) { return s.settings, nil }
func (s *memoryTraySettings) Save(settings persistence.Settings) error {
	s.settings = settings
	return nil
}

func TestTrayControllerPersistsImmediateVisibilityChanges(t *testing.T) {
	store := &memoryTraySettings{settings: persistence.Settings{ShowTrayIcon: true}}
	controller, err := NewTrayController(store)
	if err != nil {
		t.Fatal(err)
	}
	if !controller.Visible() {
		t.Fatal("Visible() = false")
	}
	if err := controller.SetVisible(false); err != nil {
		t.Fatal(err)
	}
	if controller.Visible() || store.settings.ShowTrayIcon {
		t.Fatalf("settings = %#v", store.settings)
	}
}
