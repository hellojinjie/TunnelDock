package sshconfig

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/hellojinjie/TunnelDock/Windows/internal/model"
)

func TestParseEffectiveConfigUsesFirstRequiredValues(t *testing.T) {
	output := []byte("hostname 192.0.2.10\nuser alice\nport 2222\nhostname ignored\nproxyjump jump\n")
	got, err := ParseEffectiveConfig(output)
	if err != nil {
		t.Fatalf("ParseEffectiveConfig() error: %v", err)
	}
	want := EffectiveConfig{Hostname: "192.0.2.10", User: "alice", Port: 2222}
	if got != want {
		t.Fatalf("ParseEffectiveConfig() = %#v, want %#v", got, want)
	}
}

func TestHostResolverRunsSSHGDedicatedArguments(t *testing.T) {
	runner := &recordingRunner{stdout: []byte("hostname server\nuser bob\nport 22\n")}
	resolver := NewHostResolver(`C:\Windows\System32\OpenSSH\ssh.exe`, runner)
	host := resolver.Resolve(context.Background(), "gpu", 7)

	if runner.executable != `C:\Windows\System32\OpenSSH\ssh.exe` || !reflect.DeepEqual(runner.args, []string{"-G", "gpu"}) {
		t.Fatalf("runner call = %q %#v", runner.executable, runner.args)
	}
	if host.Alias != "gpu" || host.Hostname != "server" || host.User != "bob" || host.Port != 22 || host.ConfigOrder != 7 || host.Availability != model.HostAvailable {
		t.Fatalf("Resolve() = %#v", host)
	}
}

func TestHostResolverReportsConfigurationError(t *testing.T) {
	runner := &recordingRunner{stderr: []byte("Bad configuration option"), exitCode: 255, err: errors.New("exit status 255")}
	host := NewHostResolver("ssh.exe", runner).Resolve(context.Background(), "broken", 1)
	if host.Availability != model.HostConfigurationError || host.Error != "Bad configuration option" {
		t.Fatalf("Resolve() = %#v", host)
	}
}

type recordingRunner struct {
	executable string
	args       []string
	stdout     []byte
	stderr     []byte
	exitCode   int
	err        error
}

func (r *recordingRunner) Run(_ context.Context, executable string, args ...string) ([]byte, []byte, int, error) {
	r.executable = executable
	r.args = append([]string(nil), args...)
	return r.stdout, r.stderr, r.exitCode, r.err
}
