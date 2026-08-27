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
