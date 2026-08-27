package sshclient

import (
	"fmt"
	"os"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

type Job struct {
	mu     sync.Mutex
	handle windows.Handle
}

func NewJob() (*Job, error) {
	handle, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return nil, fmt.Errorf("create Job Object: %w", err)
	}
	info := killOnCloseInformation()
	if _, err := windows.SetInformationJobObject(
		handle,
		windows.JobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&info)),
		uint32(unsafe.Sizeof(info)),
	); err != nil {
		_ = windows.CloseHandle(handle)
		return nil, fmt.Errorf("configure Job Object: %w", err)
	}
	return &Job{handle: handle}, nil
}

func killOnCloseInformation() windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION {
	var info windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION
	info.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
	return info
}

func (j *Job) Assign(process *os.Process) error {
	if process == nil {
		return fmt.Errorf("assign process to Job Object: process is nil")
	}
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.handle == 0 {
		return fmt.Errorf("assign process to Job Object: Job is closed")
	}
	var assignErr error
	if err := process.WithHandle(func(processHandle uintptr) {
		assignErr = windows.AssignProcessToJobObject(j.handle, windows.Handle(processHandle))
	}); err != nil {
		return fmt.Errorf("access process handle: %w", err)
	}
	if assignErr != nil {
		return fmt.Errorf("assign process to Job Object: %w", assignErr)
	}
	return nil
}

func (j *Job) Close() error {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.handle == 0 {
		return nil
	}
	handle := j.handle
	j.handle = 0
	if err := windows.CloseHandle(handle); err != nil {
		return fmt.Errorf("close Job Object: %w", err)
	}
	return nil
}
