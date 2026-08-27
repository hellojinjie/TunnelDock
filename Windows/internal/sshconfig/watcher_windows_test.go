package sshconfig

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDebounceEventsCoalescesBurstAndEmitsLaterChange(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	input := make(chan struct{}, 8)
	output := debounceEvents(ctx, input, 30*time.Millisecond)
	input <- struct{}{}
	input <- struct{}{}
	input <- struct{}{}
	waitForWatchEvent(t, output)
	select {
	case <-output:
		t.Fatal("burst produced more than one event")
	case <-time.After(60 * time.Millisecond):
	}
	input <- struct{}{}
	waitForWatchEvent(t, output)
}

func TestWatcherReportsTemporaryDirectoryChange(t *testing.T) {
	directory := t.TempDir()
	config := filepath.Join(directory, "config")
	if err := os.WriteFile(config, []byte("Host before\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	watcher := NewWatcher(50 * time.Millisecond)
	events, err := watcher.Events(ctx, ExpandedConfig{
		WatchedFiles: []string{config}, WatchedDirectories: []string{directory},
	})
	if err != nil {
		t.Fatalf("Events() error: %v", err)
	}
	if err := os.WriteFile(config, []byte("Host after\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	waitForWatchEvent(t, events)
}

func TestWatchDirectoriesIncludesParentsOfFilesAndNearestExistingAncestor(t *testing.T) {
	root := t.TempDir()
	existing := filepath.Join(root, "existing")
	if err := os.Mkdir(existing, 0o700); err != nil {
		t.Fatal(err)
	}
	got := watchDirectories(ExpandedConfig{
		WatchedFiles:       []string{filepath.Join(existing, "config")},
		WatchedDirectories: []string{filepath.Join(root, "missing", "nested")},
	})
	set := make(map[string]bool, len(got))
	for _, directory := range got {
		set[directory] = true
	}
	if len(got) != 2 || !set[existing] || !set[root] {
		t.Fatalf("watchDirectories() = %#v, want %q and %q", got, existing, root)
	}
}

func waitForWatchEvent(t *testing.T, events <-chan struct{}) {
	t.Helper()
	select {
	case _, open := <-events:
		if !open {
			t.Fatal("watch event channel closed")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for watch event")
	}
}
