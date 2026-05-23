// Package runtimes is the built-in registry of per-runtime, per-kind
// asset→destination mappings. A Mapping declares which subtree of the
// canonical asset (From) materializes to which path in the consumer (To).
//
// To template:
//   - trailing "/" → directory materialization (the From subtree fans out
//     into the directory)
//   - no trailing "/" → single-file materialization (one file at To)
//   - "<name>" placeholder is replaced with the asset name
//
// From template:
//   - "" or "." → the whole canonical tree
//   - otherwise a relative path inside the canonical tree (file or dir)
//
// Consumers override mappings per-entry in the lock file (assets.json
// runtimes map). Higher-precedence sources (asset manifest, runtimes.yaml
// config) will be layered on in later phases.
package runtimes

import (
	"fmt"
	"sort"
	"strings"

	"github.com/niradler/git-skill/internal/kind"
)

type Mapping struct {
	From string
	To   string
}

// IsDir reports whether the mapping materializes as a directory (To ends with "/").
func (m Mapping) IsDir() bool { return strings.HasSuffix(m.To, "/") }

var registry = map[string]map[kind.Kind]Mapping{
	"claude": {
		kind.Skill: {From: ".", To: ".claude/skills/<name>/"},
		kind.Agent: {From: "AGENT.md", To: ".claude/agents/<name>.md"},
	},
	"cursor": {
		kind.Skill: {From: ".", To: ".cursor/rules/<name>/"},
	},
	"codex": {
		kind.Skill: {From: ".", To: ".agents/skills/<name>/"},
		kind.Agent: {From: "agent.toml", To: ".codex/agents/<name>.toml"},
	},
	"opencode": {
		kind.Skill: {From: ".", To: ".agents/skills/<name>/"},
	},
}

// Resolve returns the built-in Mapping for (runtime, kind) with the
// "<name>" placeholder substituted. The From field is returned verbatim
// (no placeholder expansion in v1).
func Resolve(runtime string, k kind.Kind, asset string) (Mapping, error) {
	entry, ok := registry[runtime]
	if !ok {
		return Mapping{}, fmt.Errorf("unknown runtime %q (known: %s)", runtime, strings.Join(Known(), ", "))
	}
	tpl, ok := entry[k]
	if !ok {
		return Mapping{}, fmt.Errorf("runtime %q does not support kind %s", runtime, k)
	}
	return Mapping{
		From: tpl.From,
		To:   strings.ReplaceAll(tpl.To, "<name>", asset),
	}, nil
}

func Known() []string {
	names := make([]string, 0, len(registry))
	for n := range registry {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// GitignoreLines returns the set of destination prefixes (the directory
// portion of each registered To template, with the "<name>" placeholder
// stripped) for seeding .gitignore.
func GitignoreLines() []string {
	seen := map[string]struct{}{}
	for _, kinds := range registry {
		for _, tpl := range kinds {
			seen[gitignorePrefix(tpl.To)] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for p := range seen {
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}

// gitignorePrefix returns the parent directory of a To template, suitable
// for a .gitignore entry. Examples:
//
//	".claude/skills/<name>/"   → ".claude/skills/"
//	".claude/agents/<name>.md" → ".claude/agents/"
//	".tools/agents/static.md"  → ".tools/agents/" (no placeholder; LastIndex fallback)
func gitignorePrefix(to string) string {
	if i := strings.Index(to, "<name>"); i >= 0 {
		return to[:i]
	}
	if i := strings.LastIndex(to, "/"); i >= 0 {
		return to[:i+1]
	}
	return to
}
