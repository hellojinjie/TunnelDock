package sshconfig

import (
	"reflect"
	"testing"
)

func TestScannerDiscoversExplicitAliasesOnceInOrder(t *testing.T) {
	lines := []ExpandedLine{
		{Text: "Host *"},
		{Text: "Host gpu gpu-server lab-gpu"},
		{Text: "Host gpu-* lab-? [abc] !blocked"},
		{Text: "Host gpu"},
		{Text: `Host "hash#alias" plain # comment`},
	}

	got := Scanner{}.DiscoverAliases(lines)
	want := []string{"gpu", "gpu-server", "lab-gpu", "hash#alias", "plain"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("DiscoverAliases() = %#v, want %#v", got, want)
	}
}

func TestLexerAcceptsKeywordEqualsAndQuotedValues(t *testing.T) {
	tests := map[string][]string{
		"Host=eqalias":         {"Host", "eqalias"},
		`hOsT="space alias"`:   {"hOsT", "space alias"},
		"Host = spaced":        {"Host", "spaced"},
		`Include="config d/*"`: {"Include", "config d/*"},
	}
	for line, want := range tests {
		if got := Tokens(line); !reflect.DeepEqual(got, want) {
			t.Errorf("Tokens(%q) = %#v, want %#v", line, got, want)
		}
	}
}
