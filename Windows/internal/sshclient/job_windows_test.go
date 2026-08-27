package sshclient

import (
	"os"
	"os/exec"
	"testing"
	"time"

	"golang.org/x/sys/windows"
)

func TestKillOnCloseInformationSetsOnlyRequiredLimit(t *testing.T) {
	info := killOnCloseInformation()
	if info.BasicLimitInformation.LimitFlags != windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE {
		t.Fatalf("LimitFlags = %#x", info.BasicLimitInformation.LimitFlags)
	}
}

func TestJobCloseTerminatesAssignedChild(t *testing.T) {
	if os.Getenv("TUNNELDOCK_JOB_HELPER") == "1" {
		for {
			time.Sleep(time.Hour)
		}
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	command := exec.Command(executable, "-test.run=TestJobCloseTerminatesAssignedChild")
	command.Env = append(os.Environ(), "TUNNELDOCK_JOB_HELPER=1")
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if command.ProcessState == nil {
			_ = command.Process.Kill()
			_, _ = command.Process.Wait()
		}
	}()

	job, err := NewJob()
	if err != nil {
		t.Fatal(err)
	}
	if err := job.Assign(command.Process); err != nil {
		_ = job.Close()
		t.Fatal(err)
	}
	if err := job.Close(); err != nil {
		t.Fatal(err)
	}

	done := make(chan error, 1)
	go func() {
		done <- command.Wait()
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("assigned child survived Job close")
	}
}
