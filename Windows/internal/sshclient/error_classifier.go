package sshclient

import (
	"errors"
	"strings"
)

type ErrorKind int

const (
	ErrorLocalPortInUse ErrorKind = iota
	ErrorAuthenticationFailed
	ErrorHostVerificationRequired
	ErrorHostNotFound
	ErrorSSHConfiguration
	ErrorConnectionTimedOut
	ErrorCouldNotResolveHost
	ErrorSSHServerRefused
	ErrorUnexpectedExit
	ErrorOpenSSHNotInstalled
)

type ClassifiedError struct {
	Kind      ErrorKind
	Message   string
	RawStderr string
}

// ConnectionFailure retains the actionable classification, raw ssh.exe output,
// and startup cause so the UI can explain a failed tunnel without losing the
// information needed to diagnose it.
type ConnectionFailure struct {
	ClassifiedError
	Cause error
}

func NewConnectionFailure(stderr string, cause error) *ConnectionFailure {
	classified := ClassifyOpenSSHError(stderr, cause)
	if errors.Is(cause, ErrReadinessTimedOut) {
		classified.Kind = ErrorUnexpectedExit
		classified.Message = "The SSH tunnel did not start listening within 5 seconds."
	}
	if errors.Is(cause, ErrProcessExited) && strings.TrimSpace(stderr) == "" {
		classified.Kind = ErrorUnexpectedExit
		classified.Message = "ssh.exe exited before the local tunnel was ready."
	}
	return &ConnectionFailure{ClassifiedError: classified, Cause: cause}
}

func (failure *ConnectionFailure) Error() string { return failure.Message }

func (failure *ConnectionFailure) Unwrap() error { return failure.Cause }

// SuggestedAction provides the next useful user action for every classified
// OpenSSH failure rather than leaving a generic error string in the UI.
func (failure *ConnectionFailure) SuggestedAction() string {
	switch failure.Kind {
	case ErrorLocalPortInUse:
		return "Choose a different Local Port under Advanced, then try again."
	case ErrorAuthenticationFailed:
		return "Verify the SSH user and key, then test `ssh <Host>` in a terminal."
	case ErrorHostVerificationRequired:
		return "Connect to this Host once in a terminal and verify its host key before retrying."
	case ErrorHostNotFound:
		return "Refresh SSH Hosts or open your SSH config and check the Host alias."
	case ErrorSSHConfiguration:
		return "Open your SSH config and correct the reported configuration problem."
	case ErrorConnectionTimedOut:
		return "Check the network connection, host address, and SSH port."
	case ErrorCouldNotResolveHost:
		return "Check the HostName in your SSH config and DNS or network access."
	case ErrorSSHServerRefused:
		return "Check that the SSH service is running and accepting connections on the configured port."
	case ErrorOpenSSHNotInstalled:
		return "Install Windows OpenSSH Client in Optional features, then restart TunnelDock."
	default:
		return "Review the technical details below, then test `ssh <Host>` in a terminal."
	}
}

func (failure *ConnectionFailure) Details() string {
	if stderr := strings.TrimSpace(failure.RawStderr); stderr != "" {
		return stderr
	}
	if failure.Cause != nil {
		return failure.Cause.Error()
	}
	return "No additional output was received from ssh.exe."
}

// RequiresInteractiveSSH indicates that a normal interactive ssh session can
// complete a blocked trust or authentication step that TunnelDock deliberately
// runs in BatchMode.
func (failure *ConnectionFailure) RequiresInteractiveSSH() bool {
	switch failure.Kind {
	case ErrorAuthenticationFailed, ErrorHostVerificationRequired, ErrorUnexpectedExit:
		return true
	default:
		return false
	}
}

func ClassifyOpenSSHError(stderr string, launchErr error) ClassifiedError {
	classified := ClassifiedError{RawStderr: stderr}
	if errors.Is(launchErr, ErrOpenSSHNotInstalled) {
		classified.Kind = ErrorOpenSSHNotInstalled
		classified.Message = OpenSSHNotInstalledMessage
		return classified
	}

	lower := strings.ToLower(stderr)
	switch {
	case containsAny(lower, "address already in use", "cannot listen to port", "could not request local forwarding"):
		classified.Kind = ErrorLocalPortInUse
		classified.Message = "Local port is already in use."
	case containsAny(lower, "permission denied", "no supported authentication methods available", "authentication failed"):
		classified.Kind = ErrorAuthenticationFailed
		classified.Message = "Authentication failed."
	case containsAny(lower, "host key verification failed", "remote host identification has changed", "requested strict checking"):
		classified.Kind = ErrorHostVerificationRequired
		classified.Message = "Host verification required."
	case containsAny(lower, "host alias is no longer present", "host not found"):
		classified.Kind = ErrorHostNotFound
		classified.Message = "Host not found."
	case containsAny(lower, "bad configuration option", "bad owner or permissions on", "could not open user config file", "terminating, "):
		classified.Kind = ErrorSSHConfiguration
		classified.Message = "SSH configuration error."
	case containsAny(lower, "could not resolve hostname", "name or service not known", "no such host is known"):
		classified.Kind = ErrorCouldNotResolveHost
		classified.Message = "Could not resolve host."
	case containsAny(lower, "connection timed out", "operation timed out"):
		classified.Kind = ErrorConnectionTimedOut
		classified.Message = "Connection timed out."
	case strings.Contains(lower, "connection refused"):
		classified.Kind = ErrorSSHServerRefused
		classified.Message = "Connection refused by SSH server."
	default:
		classified.Kind = ErrorUnexpectedExit
		classified.Message = "SSH connection failed.\nSee Log for details."
	}
	return classified
}

func containsAny(value string, candidates ...string) bool {
	for _, candidate := range candidates {
		if strings.Contains(value, candidate) {
			return true
		}
	}
	return false
}
