// Package runtimes is the built-in registry mapping runtime names (e.g.
// "claude", "cursor") to per-kind install path templates. See spec L9.6.
//
// Authors of git-skill add new runtimes by editing this file. Consumers who
// need a custom path override the lock entry's runtimes map directly (the
// lock holds full explicit paths; this registry only seeds the initial values).
package runtimes

import (
	"fmt"
	"sort"
	"strings"

	"github.com/niradler/git-skill/internal/kind"
)

type pathTemplate struct {
	prefix string
	suffix string
}

var registry = map[string]map[kind.Kind]pathTemplate{
	"claude": {
		kind.Skill: {prefix: ".claude/skills/", suffix: ""},
		kind.Agent: {prefix: ".claude/agents/", suffix: ".md"},
	},
	"cursor": {
		kind.Skill: {prefix: ".cursor/rules/", suffix: ""},
	},
	"codex": {
		kind.Skill: {prefix: ".agents/skills/", suffix: ""},
	},
	"opencode": {
		kind.Skill: {prefix: ".agents/skills/", suffix: ""},
	},
}

func Resolve(runtime string, k kind.Kind, asset string) (string, error) {
	entry, ok := registry[runtime]
	if !ok {
		return "", fmt.Errorf("unknown runtime %q (known: %s)", runtime, strings.Join(Known(), ", "))
	}
	tpl, ok := entry[k]
	if !ok {
		return "", fmt.Errorf("runtime %q does not support kind %s", runtime, k)
	}
	return tpl.prefix + asset + tpl.suffix, nil
}

func Known() []string {
	names := make([]string, 0, len(registry))
	for n := range registry {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

func GitignoreLines() []string {
	seen := map[string]struct{}{}
	for _, kinds := range registry {
		for _, tpl := range kinds {
			seen[tpl.prefix] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for p := range seen {
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}
