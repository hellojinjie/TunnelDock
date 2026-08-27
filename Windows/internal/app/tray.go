package app

import "github.com/hellojinjie/TunnelDock/Windows/internal/persistence"

type TraySettingsStore interface {
	Load() (persistence.Settings, error)
	Save(persistence.Settings) error
}

type TrayController struct {
	store   TraySettingsStore
	visible bool
}

func NewTrayController(store TraySettingsStore) (*TrayController, error) {
	settings, err := store.Load()
	if err != nil {
		return nil, err
	}
	return &TrayController{store: store, visible: settings.ShowTrayIcon}, nil
}

func (c *TrayController) Visible() bool { return c.visible }

func (c *TrayController) SetVisible(visible bool) error {
	if err := c.store.Save(persistence.Settings{ShowTrayIcon: visible}); err != nil {
		return err
	}
	c.visible = visible
	return nil
}
