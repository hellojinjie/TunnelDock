package ui

import "testing"

func TestDialogValidationKeepsFormOpen(t *testing.T) {
	state := dialogValidationState{}
	state.Reject("Port must be between 1 and 65535", "port")
	if state.Accepted || state.Message == "" || state.FocusField != "port" {
		t.Fatalf("state = %#v", state)
	}
}

func TestDialogValidationAcceptClearsPreviousError(t *testing.T) {
	state := dialogValidationState{}
	state.Reject("invalid", "name")
	state.Accept()
	if !state.Accepted || state.Message != "" || state.FocusField != "" {
		t.Fatalf("state = %#v", state)
	}
}
