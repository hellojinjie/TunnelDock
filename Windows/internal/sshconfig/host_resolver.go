package sshconfig

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"strconv"
	"strings"

	"github.com/hellojinjie/TunnelDock/Windows/internal/model"
)

type EffectiveConfig struct {
	Hostname string
	User     string
	Port     uint16
}

var errInvalidEffectiveConfig = errors.New("ssh -G omitted or returned an invalid hostname, user, or port")

func ParseEffectiveConfig(output []byte) (EffectiveConfig, error) {
	values := make(map[string]string)
	for _, line := range strings.Split(string(output), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		key := strings.ToLower(fields[0])
		if _, exists := values[key]; !exists {
			values[key] = strings.Join(fields[1:], " ")
		}
	}
	port, err := strconv.ParseUint(values["port"], 10, 16)
	if err != nil || port == 0 || values["hostname"] == "" || values["user"] == "" {
		return EffectiveConfig{}, errInvalidEffectiveConfig
	}
	return EffectiveConfig{Hostname: values["hostname"], User: values["user"], Port: uint16(port)}, nil
}

type CommandRunner interface {
	Run(ctx context.Context, executable string, args ...string) (stdout, stderr []byte, exitCode int, err error)
}

type ExecRunner struct{}

func (ExecRunner) Run(ctx context.Context, executable string, args ...string) ([]byte, []byte, int, error) {
	command := exec.CommandContext(ctx, executable, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	err := command.Run()
	exitCode := 0
	if command.ProcessState != nil {
		exitCode = command.ProcessState.ExitCode()
	}
	return stdout.Bytes(), stderr.Bytes(), exitCode, err
}

type HostResolver struct {
	executable string
	runner     CommandRunner
}

func NewHostResolver(executable string, runner CommandRunner) HostResolver {
	return HostResolver{executable: executable, runner: runner}
}

func (r HostResolver) Resolve(ctx context.Context, alias string, order int) model.SSHHost {
	stdout, stderr, exitCode, runErr := r.runner.Run(ctx, r.executable, "-G", alias)
	if runErr != nil || exitCode != 0 {
		message := strings.TrimSpace(string(stderr))
		if message == "" && runErr != nil {
			message = runErr.Error()
		}
		return failedHost(alias, order, message)
	}
	config, err := ParseEffectiveConfig(stdout)
	if err != nil {
		return failedHost(alias, order, err.Error())
	}
	return model.SSHHost{
		Alias: alias, Hostname: config.Hostname, User: config.User, Port: config.Port,
		ConfigOrder: order, Availability: model.HostAvailable,
	}
}

func failedHost(alias string, order int, message string) model.SSHHost {
	return model.SSHHost{
		Alias: alias, Hostname: alias, Port: 22, ConfigOrder: order,
		Availability: model.HostConfigurationError, Error: message,
	}
}
