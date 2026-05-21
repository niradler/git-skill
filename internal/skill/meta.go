package skill

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Meta struct {
	Name        string
	Description string
	Version     string
	License     string
}

func ParseMeta(dir string) (*Meta, error) {
	path := filepath.Join(dir, "SKILL.md")
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("no SKILL.md in %s: %w", dir, err)
	}
	defer f.Close()

	m := &Meta{}
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
			m.Name = val
		case "description":
			m.Description = val
		case "version":
			m.Version = val
		case "license":
			m.License = val
		}
	}
	return m, scanner.Err()
}

func FormatCommitMessage(subject, body string, meta *Meta) string {
	var b strings.Builder
	b.WriteString(subject)
	if body != "" {
		b.WriteString("\n\n")
		b.WriteString(body)
	}
	if meta != nil && meta.Version != "" {
		b.WriteString("\n\nSkill-Version: " + meta.Version)
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
