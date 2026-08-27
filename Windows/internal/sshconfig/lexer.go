package sshconfig

import "strings"

func Tokens(line string) []string {
	var tokens []string
	var current strings.Builder
	var quote rune
	escaping := false
	finish := func() {
		if current.Len() > 0 {
			tokens = append(tokens, current.String())
			current.Reset()
		}
	}

	for _, character := range line {
		if escaping {
			current.WriteRune(character)
			escaping = false
			continue
		}
		if character == '\\' {
			escaping = true
			continue
		}
		if quote != 0 {
			if character == quote {
				quote = 0
			} else {
				current.WriteRune(character)
			}
			continue
		}
		switch {
		case character == '\'' || character == '"':
			quote = character
		case character == '#':
			finish()
			return normalizeKeywordEquals(tokens)
		case character == '=' && len(tokens) == 0 && isConfigKeyword(current.String()):
			finish()
		case character == ' ' || character == '\t' || character == '\r' || character == '\n':
			finish()
		default:
			current.WriteRune(character)
		}
	}
	if escaping {
		current.WriteRune('\\')
	}
	finish()
	return normalizeKeywordEquals(tokens)
}

func normalizeKeywordEquals(tokens []string) []string {
	if len(tokens) < 2 || !isConfigKeyword(tokens[0]) {
		return tokens
	}
	if tokens[1] == "=" {
		return append(tokens[:1], tokens[2:]...)
	}
	if strings.HasPrefix(tokens[1], "=") {
		tokens[1] = strings.TrimPrefix(tokens[1], "=")
		if tokens[1] == "" {
			return append(tokens[:1], tokens[2:]...)
		}
	}
	return tokens
}

func isConfigKeyword(value string) bool {
	return strings.EqualFold(value, "Host") || strings.EqualFold(value, "Include")
}
