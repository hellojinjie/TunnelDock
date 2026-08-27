package sshconfig

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type DiagnosticKind string

const (
	DiagnosticIncludeCycle   DiagnosticKind = "include-cycle"
	DiagnosticUnreadableFile DiagnosticKind = "unreadable-file"
)

type Diagnostic struct {
	Kind    DiagnosticKind
	Path    string
	Message string
}

type ExpandedConfig struct {
	Lines              []ExpandedLine
	WatchedFiles       []string
	WatchedDirectories []string
	Diagnostics        []Diagnostic
}

type IncludeResolver struct {
	sshDirectory  string
	homeDirectory string
}

func NewIncludeResolver(sshDirectory string) IncludeResolver {
	sshDirectory = cleanAbsolute(sshDirectory)
	return IncludeResolver{
		sshDirectory:  sshDirectory,
		homeDirectory: filepath.Dir(sshDirectory),
	}
}

func (r IncludeResolver) Resolve(root string) (ExpandedConfig, error) {
	accumulator := resolverAccumulator{
		watchedFiles:       make(map[string]struct{}),
		watchedDirectories: map[string]struct{}{r.sshDirectory: {}},
		active:             make(map[string]struct{}),
	}
	if err := r.expand(cleanAbsolute(root), true, &accumulator); err != nil {
		return ExpandedConfig{}, err
	}
	return accumulator.result(), nil
}

func (r IncludeResolver) expand(path string, root bool, accumulator *resolverAccumulator) error {
	path = cleanAbsolute(path)
	if _, active := accumulator.active[path]; active {
		accumulator.diagnostics = append(accumulator.diagnostics, Diagnostic{
			Kind: DiagnosticIncludeCycle, Path: path, Message: "recursive Include cycle",
		})
		return nil
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) && root {
		return nil
	}
	if err != nil {
		accumulator.diagnostics = append(accumulator.diagnostics, Diagnostic{
			Kind: DiagnosticUnreadableFile, Path: path, Message: err.Error(),
		})
		return nil
	}
	accumulator.active[path] = struct{}{}
	defer delete(accumulator.active, path)
	accumulator.watchedFiles[path] = struct{}{}

	for index, text := range strings.Split(string(data), "\n") {
		tokens := Tokens(strings.TrimSuffix(text, "\r"))
		if len(tokens) == 0 || !strings.EqualFold(tokens[0], "Include") {
			accumulator.lines = append(accumulator.lines, ExpandedLine{
				SourcePath: path, LineNumber: index + 1, Text: strings.TrimSuffix(text, "\r"),
			})
			continue
		}
		for _, pattern := range tokens[1:] {
			expandedPattern := r.expandPattern(pattern)
			accumulator.watchedDirectories[globDirectory(expandedPattern)] = struct{}{}
			matches, globErr := filepath.Glob(expandedPattern)
			if globErr != nil {
				return fmt.Errorf("expand Include %q: %w", pattern, globErr)
			}
			sort.Strings(matches)
			for _, match := range matches {
				if err := r.expand(match, false, accumulator); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (r IncludeResolver) expandPattern(pattern string) string {
	pattern = filepath.FromSlash(pattern)
	if pattern == "~" {
		return r.homeDirectory
	}
	if strings.HasPrefix(pattern, "~"+string(filepath.Separator)) {
		return filepath.Join(r.homeDirectory, strings.TrimPrefix(pattern, "~"+string(filepath.Separator)))
	}
	if filepath.IsAbs(pattern) {
		return filepath.Clean(pattern)
	}
	return filepath.Join(r.sshDirectory, pattern)
}

func globDirectory(pattern string) string {
	first := strings.IndexAny(pattern, "*?[")
	if first < 0 {
		return filepath.Dir(pattern)
	}
	prefix := pattern[:first]
	if !strings.HasSuffix(prefix, string(filepath.Separator)) {
		prefix = filepath.Dir(prefix)
	}
	return filepath.Clean(prefix)
}

func cleanAbsolute(path string) string {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return filepath.Clean(path)
	}
	return filepath.Clean(absolute)
}

type resolverAccumulator struct {
	lines              []ExpandedLine
	watchedFiles       map[string]struct{}
	watchedDirectories map[string]struct{}
	diagnostics        []Diagnostic
	active             map[string]struct{}
}

func (a resolverAccumulator) result() ExpandedConfig {
	files := mapKeys(a.watchedFiles)
	directories := mapKeys(a.watchedDirectories)
	return ExpandedConfig{
		Lines: a.lines, WatchedFiles: files, WatchedDirectories: directories, Diagnostics: a.diagnostics,
	}
}

func mapKeys(values map[string]struct{}) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
