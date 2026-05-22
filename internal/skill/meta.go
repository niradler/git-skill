package skill

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/niradler/git-skill/internal/kind"
)

type Frontmatter struct {
	Name        string
	Description string
	Version     string
	License     string
	Kind        string // optional; one of "skill" / "agent" when set
	MarkerFile  string // basename of the marker file found (e.g. "SKILL.md", "AGENT.md")
}

// Meta is the previous name. Kept as an alias only as long as the rewrite of
// callers in main.go is incomplete (Task 14 finishes that). DELETE after.
type Meta = Frontmatter

// candidateMarkers lists the marker filenames tried in order. Case-insensitive.
var candidateMarkers = []string{"SKILL.md", "AGENT.md"}

func findMarker(dir string) (path string, basename string, err error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", "", err
	}
	want := make(map[string]struct{}, len(candidateMarkers))
	for _, m := range candidateMarkers {
		want[strings.ToLower(m)] = struct{}{}
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if _, ok := want[strings.ToLower(e.Name())]; ok {
			return filepath.Join(dir, e.Name()), e.Name(), nil
		}
	}
	return "", "", fmt.Errorf("no marker file (SKILL.md / AGENT.md) in %s", dir)
}

func ParseFrontmatter(dir string) (*Frontmatter, error) {
	path, base, err := findMarker(dir)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	fm := &Frontmatter{MarkerFile: base}
	scanner := bufio.NewScanner(f)
	inFrontmatter := false
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "---" {
			if inFrontmatter {
				break
			}
			inFrontmatter = true
			continue
		}
		if !inFrontmatter {
			continue
		}
		key, val, ok := parseYAMLLine(line)
		if !ok {
			continue
		}
		switch key {
		case "name":
			fm.Name = val
		case "description":
			fm.Description = val
		case "version":
			fm.Version = val
		case "license":
			fm.License = val
		case "kind":
			fm.Kind = val
		}
	}
	return fm, scanner.Err()
}

// ParseMeta is the previous entrypoint. Forwards to ParseFrontmatter.
// DELETE after Task 14.
func ParseMeta(dir string) (*Meta, error) {
	return ParseFrontmatter(dir)
}

// ReadFrontmatter is the public read-and-return-value entrypoint used by
// commit/install code. Returns a zero-value Frontmatter (not nil) when the
// marker file has no YAML block, so callers can switch on fm.Kind safely.
func ReadFrontmatter(dir string) (Frontmatter, error) {
	fm, err := ParseFrontmatter(dir)
	if err != nil {
		return Frontmatter{}, err
	}
	if fm == nil {
		return Frontmatter{}, nil
	}
	return *fm, nil
}

// KindFromMarkerFile inspects the marker filename in dir (SKILL.md / AGENT.md,
// case-insensitive) and returns the implied kind. Returns (0, false) when no
// marker file is present or the name doesn't map to a known kind.
//
// Imported only where avoiding a kind package dep would create a cycle; the
// canonical path is the L3 discriminator chain.
func KindFromMarkerFile(dir string) (kind.Kind, bool) {
	_, base, err := findMarker(dir)
	if err != nil {
		return 0, false
	}
	switch strings.ToLower(base) {
	case "skill.md":
		return kind.Skill, true
	case "agent.md":
		return kind.Agent, true
	}
	return 0, false
}

func FormatCommitMessage(subject, body string, fm *Frontmatter) string {
	var b strings.Builder
	b.WriteString(subject)
	if body != "" {
		b.WriteString("\n\n")
		b.WriteString(body)
	}
	if fm != nil && fm.Version != "" {
		b.WriteString("\n\nSkill-Version: " + fm.Version)
	}
	return b.String()
}

func Scaffold(dir, name, description string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	content := fmt.Sprintf(`---
name: %s
description: %s
version: 0.1.0
kind: skill
---

# %s

TODO: Describe what this skill does.

## Instructions

TODO: Add the instructions that guide the AI agent.
`, name, description, name)
	return os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0644)
}

func parseYAMLLine(line string) (string, string, bool) {
	idx := strings.Index(line, ":")
	if idx < 0 {
		return "", "", false
	}
	key := strings.TrimSpace(line[:idx])
	val := strings.TrimSpace(line[idx+1:])
	return key, val, true
}
