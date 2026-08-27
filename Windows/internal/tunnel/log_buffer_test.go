package tunnel

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestLogBufferKeepsExactlyNewest500Lines(t *testing.T) {
	buffer := NewLogBuffer()
	for index := range 501 {
		buffer.Add(fmt.Sprintf("line-%03d", index))
	}
	lines := buffer.Lines()
	if len(lines) != 500 {
		t.Fatalf("len(Lines()) = %d, want 500", len(lines))
	}
	if lines[0] != "line-001" || lines[499] != "line-500" {
		t.Fatalf("retained range = %q ... %q", lines[0], lines[499])
	}
}

func TestLogBufferReturnsIndependentSnapshot(t *testing.T) {
	buffer := NewLogBuffer()
	buffer.Add("original")
	lines := buffer.Lines()
	lines[0] = "changed"
	if got := buffer.Lines()[0]; got != "original" {
		t.Fatalf("buffer changed through snapshot: %q", got)
	}
}

func TestLogBufferFormatsLifecycleTimestamp(t *testing.T) {
	buffer := NewLogBuffer()
	buffer.AddAt(time.Date(2026, 8, 27, 11, 30, 2, 0, time.Local), "Connecting...")
	if got, want := buffer.Lines()[0], "[11:30:02] Connecting..."; got != want {
		t.Fatalf("AddAt() = %q, want %q", got, want)
	}
}

func TestLogBufferSupportsConcurrentWriters(t *testing.T) {
	buffer := NewLogBuffer()
	var waitGroup sync.WaitGroup
	for worker := range 20 {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			for line := range 100 {
				buffer.Add(fmt.Sprintf("worker-%02d-line-%03d", worker, line))
			}
		}()
	}
	waitGroup.Wait()
	lines := buffer.Lines()
	if len(lines) != 500 {
		t.Fatalf("len(Lines()) = %d, want 500", len(lines))
	}
	seen := make(map[string]struct{}, len(lines))
	for _, line := range lines {
		seen[line] = struct{}{}
	}
	if len(seen) != 500 {
		t.Fatalf("unique retained lines = %d, want 500", len(seen))
	}
}
