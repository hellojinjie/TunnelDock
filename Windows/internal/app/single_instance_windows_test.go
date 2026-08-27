package app

import (
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestAcquireSingleInstanceRejectsSecondOwner(t *testing.T) {
	name := fmt.Sprintf("Local\\TunnelDock.Test.%d", time.Now().UnixNano())
	first, err := AcquireSingleInstance(name)
	if err != nil {
		t.Fatal(err)
	}
	defer first.Close()
	second, err := AcquireSingleInstance(name)
	if !errors.Is(err, ErrAlreadyRunning) {
		t.Fatalf("second acquire error = %v", err)
	}
	if second != nil {
		t.Fatal("second owner = non-nil")
	}
}
