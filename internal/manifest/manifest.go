// Package manifest parses the optional asset manifest (git-skill.yaml)
// that ships at the root of a canonical asset tree. The manifest lets
// asset authors declare per-runtime From/To overrides on top of the
// built-in registry, without forcing every consumer to set --target.
//
// Resolution precedence (lowest → highest): built-in registry < user
// runtimes.yaml < project runtimes.yaml < asset manifest < lock entry
// override.
//
// Note: manifest values support "<name>" placeholder substitution in
// both `from` and `to`. The built-in registry only substitutes in `to`;
// manifest authors thus get strictly more templating than registry
// authors, by design.
package manifest

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/niradler/git-skill/internal/runtimes"
)

// Filename is the canonical manifest filename. Looked up relative to the
// root of the asset's canonical tree.
const Filename = "git-skill.yaml"

// Manifest is the parsed asset manifest. A nil/zero manifest means the
// file was absent and the registry default applies.
type Manifest struct {
	// Kind is an optional author hint ("skill" / "agent"). Parsed but not
	// consumed by Phase B - the install path infers kind elsewhere.
	Kind string `yaml:"kind"`
	// Runtimes maps runtime name → mapping override. An entry may set
	// just From, just To, or both. Missing fields fall back to the
	// registry default for that (runtime, kind).
	Runtimes map[string]Entry `yaml:"runtimes"`
}

// Entry is the per-runtime override declared in the manifest.
type Entry struct {
	From string `yaml:"from"`
	To   string `yaml:"to"`
}

// Load reads <canonAbs>/git-skill.yaml. Returns (nil, nil) when the file
// is absent - callers treat that as "no manifest, use registry default".
func Load(canonAbs string) (*Manifest, error) {
	path := filepath.Join(canonAbs, Filename)
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", Filename, err)
	}
	m := &Manifest{}
	if err := yaml.Unmarshal(data, m); err != nil {
		return nil, fmt.Errorf("parse %s: %w", Filename, err)
	}
	for rt, entry := range m.Runtimes {
		if rt == "" {
			return nil, fmt.Errorf("%s: runtime name must not be empty", Filename)
		}
		if entry.From == "" && entry.To == "" {
			return nil, fmt.Errorf("%s: runtime %q: at least one of 'from' or 'to' must be set", Filename, rt)
		}
	}
	return m, nil
}

// Apply layers the manifest's runtime override on top of base. Fields
// set in the manifest override the corresponding fields in base; unset
// fields keep base's value. Any "<name>" placeholders in the manifest
// values are substituted with name. Calling Apply with a nil manifest,
// or with a manifest that has no entry for rt, returns base unchanged.
func Apply(base runtimes.Mapping, m *Manifest, rt, name string) runtimes.Mapping {
	if m == nil {
		return base
	}
	entry, ok := m.Runtimes[rt]
	if !ok {
		return base
	}
	if entry.From != "" {
		base.From = strings.ReplaceAll(entry.From, "<name>", name)
	}
	if entry.To != "" {
		base.To = strings.ReplaceAll(entry.To, "<name>", name)
	}
	return base
}

// Mapping returns the manifest's entry for rt as a runtimes.Mapping with
// "<name>" placeholders substituted. Used when the runtime is not in the
// built-in registry but is declared in the manifest. Both From and To
// must be set in the manifest for a registry-less runtime to materialize;
// the caller validates that.
func (m *Manifest) Mapping(rt, name string) (runtimes.Mapping, bool) {
	if m == nil {
		return runtimes.Mapping{}, false
	}
	entry, ok := m.Runtimes[rt]
	if !ok {
		return runtimes.Mapping{}, false
	}
	return runtimes.Mapping{
		From: strings.ReplaceAll(entry.From, "<name>", name),
		To:   strings.ReplaceAll(entry.To, "<name>", name),
	}, true
}
