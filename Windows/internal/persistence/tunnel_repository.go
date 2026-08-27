package persistence

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/hellojinjie/TunnelDock/Windows/internal/model"
)

const tunnelSchemaVersion = 1

var (
	ErrMalformedFile     = errors.New("saved tunnels file is malformed")
	ErrUnsupportedSchema = errors.New("saved tunnels schema is unsupported")
	ErrWriteLocked       = errors.New("saved tunnels repository is write-locked")
)

type TunnelEnvelope struct {
	SchemaVersion int                      `json:"schemaVersion"`
	Tunnels       []model.TunnelDefinition `json:"tunnels"`
}

type TunnelRepository struct {
	mu          sync.Mutex
	path        string
	writeLocked bool
}

func NewTunnelRepository(path string) *TunnelRepository {
	return &TunnelRepository{path: path}
}

func (r *TunnelRepository) Load() ([]model.TunnelDefinition, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	data, err := os.ReadFile(r.path)
	if errors.Is(err, os.ErrNotExist) {
		r.writeLocked = false
		return []model.TunnelDefinition{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read saved tunnels: %w", err)
	}

	var envelope TunnelEnvelope
	if err := json.Unmarshal(data, &envelope); err != nil {
		r.writeLocked = true
		return nil, fmt.Errorf("%w: %v", ErrMalformedFile, err)
	}
	if envelope.SchemaVersion != tunnelSchemaVersion {
		r.writeLocked = true
		return nil, fmt.Errorf("%w: %d", ErrUnsupportedSchema, envelope.SchemaVersion)
	}

	r.writeLocked = false
	return envelope.Tunnels, nil
}

func (r *TunnelRepository) ReplaceAll(tunnels []model.TunnelDefinition) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.writeLocked {
		return ErrWriteLocked
	}
	if tunnels == nil {
		tunnels = []model.TunnelDefinition{}
	}
	data, err := json.MarshalIndent(TunnelEnvelope{
		SchemaVersion: tunnelSchemaVersion,
		Tunnels:       tunnels,
	}, "", "  ")
	if err != nil {
		return fmt.Errorf("encode saved tunnels: %w", err)
	}
	data = append(data, '\n')
	return atomicWriteFile(r.path, data)
}

func atomicWriteFile(path string, data []byte) (returnErr error) {
	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return fmt.Errorf("create storage directory: %w", err)
	}
	temporary, err := os.CreateTemp(directory, ".tunneldock-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary storage file: %w", err)
	}
	temporaryPath := temporary.Name()
	defer func() {
		if returnErr != nil {
			_ = temporary.Close()
			_ = os.Remove(temporaryPath)
		}
	}()

	if err := temporary.Chmod(0o600); err != nil {
		return fmt.Errorf("set temporary storage permissions: %w", err)
	}
	if _, err := temporary.Write(data); err != nil {
		return fmt.Errorf("write temporary storage file: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		return fmt.Errorf("sync temporary storage file: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close temporary storage file: %w", err)
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf("replace storage file: %w", err)
	}
	return nil
}
