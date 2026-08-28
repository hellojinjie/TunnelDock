package ui

import (
	"fmt"
	"testing"
)

func TestReconcileRowsKeepsIdentityAcrossReorder(t *testing.T) {
	ops := ReconcileRows([]string{"a", "b", "c"}, []string{"c", "a", "d"})
	want := []ReconcileOperation{
		{Kind: ReconcileRemove, ID: "b", From: 1, To: -1},
		{Kind: ReconcileMove, ID: "c", From: 1, To: 0},
		{Kind: ReconcileKeep, ID: "a", From: 1, To: 1},
		{Kind: ReconcileInsert, ID: "d", From: -1, To: 2},
	}
	if diff := compareReconcileOperations(ops, want); diff != "" {
		t.Fatal(diff)
	}
}

func TestReconcileRowsRemovesFromHighestIndexFirst(t *testing.T) {
	ops := ReconcileRows([]string{"a", "b", "c", "d"}, []string{"b"})
	for i, wantFrom := range []int{3, 2, 0} {
		if ops[i].Kind != ReconcileRemove || ops[i].From != wantFrom {
			t.Fatalf("operation %d = %#v, want removal from %d", i, ops[i], wantFrom)
		}
	}
}

func compareReconcileOperations(got, want []ReconcileOperation) string {
	if len(got) != len(want) {
		return fmt.Sprintf("len(got) = %d, want %d; got %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			return fmt.Sprintf("operation %d = %#v, want %#v", i, got[i], want[i])
		}
	}
	return ""
}
