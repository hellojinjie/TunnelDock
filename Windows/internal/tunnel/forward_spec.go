package tunnel

import (
	"fmt"
	"strings"
	"unicode"
)

func FormatForwardSpec(localAddress string, localPort uint16, remoteHost string, remotePort uint16) (string, error) {
	if err := validateForwardHost(localAddress); err != nil {
		return "", fmt.Errorf("local address: %w", err)
	}
	if err := validateForwardHost(remoteHost); err != nil {
		return "", fmt.Errorf("remote host: %w", err)
	}
	if localPort == 0 || remotePort == 0 {
		return "", fmt.Errorf("forward ports must be between 1 and 65535")
	}
	return fmt.Sprintf("%s:%d:%s:%d", formatForwardHost(localAddress), localPort, formatForwardHost(remoteHost), remotePort), nil
}

func validateForwardHost(value string) error {
	if value == "" || strings.IndexFunc(value, func(character rune) bool {
		return unicode.IsSpace(character) || unicode.IsControl(character)
	}) >= 0 {
		return fmt.Errorf("host is empty or contains whitespace/control characters")
	}
	return nil
}

func formatForwardHost(value string) string {
	value = strings.TrimPrefix(strings.TrimSuffix(value, "]"), "[")
	if strings.Contains(value, ":") {
		return "[" + value + "]"
	}
	return value
}
