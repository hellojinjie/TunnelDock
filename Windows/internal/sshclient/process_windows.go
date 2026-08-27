package sshclient

import (
	"context"
	"io"
	"os"
	"os/exec"
	"syscall"

	"golang.org/x/sys/windows"
)

type ExecLauncher struct{}

func (ExecLauncher) Start(ctx context.Context, command Command) (LaunchedProcess, error) {
	cmd := exec.CommandContext(ctx, command.Executable, command.Args...)
	cmd.Stdin = nil
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: windows.CREATE_NO_WINDOW,
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return &execLaunchedProcess{cmd: cmd, stdout: stdout, stderr: stderr}, nil
}

type execLaunchedProcess struct {
	cmd    *exec.Cmd
	stdout io.ReadCloser
	stderr io.ReadCloser
}

func (p *execLaunchedProcess) Stdout() io.ReadCloser { return p.stdout }
func (p *execLaunchedProcess) Stderr() io.ReadCloser { return p.stderr }
func (p *execLaunchedProcess) Process() *os.Process  { return p.cmd.Process }
func (p *execLaunchedProcess) Wait() (int, error) {
	err := p.cmd.Wait()
	if p.cmd.ProcessState == nil {
		return -1, err
	}
	return p.cmd.ProcessState.ExitCode(), err
}
func (p *execLaunchedProcess) Kill() error {
	if p.cmd.Process == nil {
		return os.ErrProcessDone
	}
	return p.cmd.Process.Kill()
}
