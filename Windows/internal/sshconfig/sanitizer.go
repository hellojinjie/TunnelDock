package sshconfig

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

var removedRuntimeDirectives = map[string]struct{}{
	"include":             {},
	"localforward":        {},
	"remoteforward":       {},
	"dynamicforward":      {},
	"clearallforwardings": {},
	"controlmaster":       {},
	"controlpath":         {},
	"controlpersist":      {},
}

func Sanitize(lines []ExpandedLine) string {
	retained := make([]string, 0, len(lines))
	for _, line := range lines {
		if _, remove := removedRuntimeDirectives[directiveKeyword(line.Text)]; remove {
			continue
		}
		retained = append(retained, line.Text)
	}
	if len(retained) == 0 {
		return ""
	}
	return strings.Join(retained, "\n") + "\n"
}

func directiveKeyword(line string) string {
	trimmed := strings.TrimLeft(line, " \t")
	end := strings.IndexFunc(trimmed, func(character rune) bool {
		return character == '=' || unicode.IsSpace(character) || character == '#'
	})
	if end < 0 {
		end = len(trimmed)
	}
	return strings.ToLower(trimmed[:end])
}

type RuntimeConfigStore struct {
	root string
}

func NewRuntimeConfigStore(root string) RuntimeConfigStore {
	return RuntimeConfigStore{root: filepath.Clean(root)}
}

func (s RuntimeConfigStore) Create(runtimeID string, lines []ExpandedLine) (string, error) {
	directory, err := s.runtimeDirectory(runtimeID)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return "", fmt.Errorf("create runtime config directory: %w", err)
	}
	path := filepath.Join(directory, "ssh_config")
	if err := os.WriteFile(path, []byte(Sanitize(lines)), 0o600); err != nil {
		_ = os.RemoveAll(directory)
		return "", fmt.Errorf("write runtime SSH config: %w", err)
	}
	return path, nil
}

func (s RuntimeConfigStore) Remove(runtimeID string) error {
	directory, err := s.runtimeDirectory(runtimeID)
	if err != nil {
		return err
	}
	if err := os.RemoveAll(directory); err != nil {
		return fmt.Errorf("remove runtime config: %w", err)
	}
	return nil
}

func (s RuntimeConfigStore) RemoveStale() error {
	if err := s.validateRoot(); err != nil {
		return err
	}
	if err := os.MkdirAll(s.root, 0o700); err != nil {
		return fmt.Errorf("create runtime root: %w", err)
	}
	entries, err := os.ReadDir(s.root)
	if err != nil {
		return fmt.Errorf("read runtime root: %w", err)
	}
	for _, entry := range entries {
		target := filepath.Join(s.root, entry.Name())
		if err := os.RemoveAll(target); err != nil {
			return fmt.Errorf("remove stale runtime path %q: %w", target, err)
		}
	}
	return nil
}

func (s RuntimeConfigStore) runtimeDirectory(runtimeID string) (string, error) {
	if err := s.validateRoot(); err != nil {
		return "", err
	}
	if runtimeID == "" || runtimeID == "." || runtimeID == ".." ||
		strings.ContainsAny(runtimeID, `/\\`) || strings.ContainsFunc(runtimeID, unicode.IsControl) {
		return "", fmt.Errorf("invalid runtime ID %q", runtimeID)
	}
	return filepath.Join(s.root, runtimeID), nil
}

func (s RuntimeConfigStore) validateRoot() error {
	if s.root == "" || s.root == "." || filepath.Dir(s.root) == s.root {
		return fmt.Errorf("unsafe runtime root %q", s.root)
	}
	return nil
}
