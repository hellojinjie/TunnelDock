package persistence

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

type Settings struct {
	ShowTrayIcon bool `json:"showTrayIcon"`
}

type SettingsStore struct {
	path string
}

func NewSettingsStore(path string) *SettingsStore {
	return &SettingsStore{path: path}
}

func (s *SettingsStore) Load() (Settings, error) {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return Settings{ShowTrayIcon: true}, nil
	}
	if err != nil {
		return Settings{}, fmt.Errorf("read settings: %w", err)
	}
	var settings Settings
	if err := json.Unmarshal(data, &settings); err != nil {
		return Settings{}, fmt.Errorf("decode settings: %w", err)
	}
	return settings, nil
}

func (s *SettingsStore) Save(settings Settings) error {
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("encode settings: %w", err)
	}
	return atomicWriteFile(s.path, append(data, '\n'))
}
