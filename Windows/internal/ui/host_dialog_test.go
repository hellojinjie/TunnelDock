package ui

import (
	"testing"
)

func TestSSHHostInputValidateAndConfigBlock(t *testing.T) {
	input := SSHHostInput{Alias: "gpu", Hostname: "gpu.example", User: "alice", Port: 2222}
	if err := input.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if got := input.ConfigBlock(); got != "Host gpu\n    HostName gpu.example\n    Port 2222\n    User alice\n" {
		t.Fatalf("ConfigBlock() = %q", got)
	}

	for _, input := range []SSHHostInput{
		{Alias: "", Hostname: "gpu.example", Port: 22},
		{Alias: "gpu host", Hostname: "gpu.example", Port: 22},
		{Alias: "gpu", Hostname: "gpu.example", Port: 0},
	} {
		if err := input.Validate(); err == nil {
			t.Fatalf("Validate(%+v) unexpectedly succeeded", input)
		}
	}
}
