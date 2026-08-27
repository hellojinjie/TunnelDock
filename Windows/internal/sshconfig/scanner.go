package sshconfig

import "strings"

type ExpandedLine struct {
	SourcePath string
	LineNumber int
	Text       string
}

type Scanner struct{}

func (Scanner) DiscoverAliases(lines []ExpandedLine) []string {
	aliases := make([]string, 0)
	seen := make(map[string]struct{})
	for _, line := range lines {
		tokens := Tokens(line.Text)
		if len(tokens) == 0 || !strings.EqualFold(tokens[0], "Host") {
			continue
		}
		for _, token := range tokens[1:] {
			if token == "" || strings.HasPrefix(token, "!") || strings.ContainsAny(token, "*?[]") {
				continue
			}
			if _, exists := seen[token]; exists {
				continue
			}
			seen[token] = struct{}{}
			aliases = append(aliases, token)
		}
	}
	return aliases
}
