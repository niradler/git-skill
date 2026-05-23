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
	"maps"
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
	return resolveIn(registry, runtime, k, asset)
}

func resolveIn(table map[string]map[kind.Kind]Mapping, runtime string, k kind.Kind, asset string) (Mapping, error) {
	entry, ok := table[runtime]
	if !ok {
		return Mapping{}, fmt.Errorf("unknown runtime %q (known: %s)", runtime, strings.Join(knownIn(table), ", "))
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

func Known() []string { return knownIn(registry) }

func knownIn(table map[string]map[kind.Kind]Mapping) []string {
	names := make([]string, 0, len(table))
	for n := range table {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// Registry is a layered view of the built-in mappings plus optional
// user (~/.config/git-skill/runtimes.yaml) and project
// (<repoRoot>/.git-skill/runtimes.yaml) configs. Build one with
// LoadRegistry; use Resolve to look up a (runtime, kind) pair just like
// the package-level Resolve.
type Registry struct {
	table map[string]map[kind.Kind]Mapping
}

// LoadRegistry returns the built-in registry deep-merged with the user
// runtimes.yaml (lower precedence) and the project runtimes.yaml
// (higher precedence). Missing files are silently skipped. A malformed
// file surfaces as an error so the user sees the problem instead of
// silently falling back to the built-in defaults.
func LoadRegistry(repoRoot string) (*Registry, error) {
	r := &Registry{table: cloneTable(registry)}
	if p := userConfigPath(); p != "" {
		c, err := LoadConfig(p)
		if err != nil {
			return nil, err
		}
		r.merge(c)
	}
	if p := projectConfigPath(repoRoot); p != "" {
		c, err := LoadConfig(p)
		if err != nil {
			return nil, err
		}
		r.merge(c)
	}
	return r, nil
}

// Resolve mirrors the package-level Resolve but consults the layered
// table on the receiver.
func (r *Registry) Resolve(runtime string, k kind.Kind, asset string) (Mapping, error) {
	return resolveIn(r.table, runtime, k, asset)
}

// Known returns the sorted list of runtime names in the layered registry.
func (r *Registry) Known() []string { return knownIn(r.table) }

// merge applies c on top of the receiver. Adds new runtimes/kinds and
// overrides existing entries field-by-field: a set From/To wins, an
// empty one preserves the existing value. The receiver's table is
// mutated in place.
func (r *Registry) merge(c *Config) {
	if c == nil {
		return
	}
	for rt, kinds := range c.Runtimes {
		if r.table[rt] == nil {
			r.table[rt] = map[kind.Kind]Mapping{}
		}
		for ks, m := range kinds {
			k, _ := kind.Parse(ks) // LoadConfig has already validated this
			existing := r.table[rt][k]
			if m.From != "" {
				existing.From = m.From
			}
			if m.To != "" {
				existing.To = m.To
			}
			r.table[rt][k] = existing
		}
	}
}

func cloneTable(src map[string]map[kind.Kind]Mapping) map[string]map[kind.Kind]Mapping {
	out := make(map[string]map[kind.Kind]Mapping, len(src))
	for rt, kinds := range src {
		inner := make(map[kind.Kind]Mapping, len(kinds))
		maps.Copy(inner, kinds)
		out[rt] = inner
	}
	return out
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
