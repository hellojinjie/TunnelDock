package ui

import "testing"

func TestLogTextJoinsRuntimeLinesForReadOnlyViewer(t *testing.T) {
	if got := LogText([]string{"[10:00:00] Connecting...", "stderr: denied"}); got != "[10:00:00] Connecting...\r\nstderr: denied" {
		t.Fatalf("LogText() = %q", got)
	}
}
