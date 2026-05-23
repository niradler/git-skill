package commands

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/niradler/git-skill/internal/assetignore"
	xfs "github.com/niradler/git-skill/internal/fs"
	"github.com/niradler/git-skill/internal/git"
	"github.com/niradler/git-skill/internal/gitops"
	"github.com/niradler/git-skill/internal/kind"
	"github.com/niradler/git-skill/internal/manifest"
	"github.com/niradler/git-skill/internal/refs"
	"github.com/niradler/git-skill/internal/runtimes"
	"github.com/niradler/git-skill/internal/state"
)

func Install(p Profile, args []string, stdout, stderr io.Writer) error {
	if len(args) > 0 {
		return Add(p, args, stdout, stderr)
	}
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	st, err := state.Read(cwd)
	if err != nil {
		return fmt.Errorf("read %s: %w", state.Filename, err)
	}
	reg, err := runtimes.LoadRegistry(cwd)
	if err != nil {
		return fmt.Errorf("load runtimes config: %w", err)
	}
	for _, k := range kind.All() {
		if p.RequireKind && k != p.DefaultKind {
			continue
		}
		for name, entry := range st.Assets[k] {
			if err := installOne(cwd, st, k, name, entry, reg, stdout); err != nil {
				return fmt.Errorf("install %s/%s: %w", k, name, err)
			}
		}
	}
	return nil
}

func installOne(repoRoot string, st *state.State, k kind.Kind, name string, e state.Entry, reg *runtimes.Registry, stdout io.Writer) error {
	if e.Commit == "" {
		return fmt.Errorf("entry has no commit pin (run 'update %s/%s' first)", k, name)
	}
	var fallbackRefs []string
	if e.Version != "" {
		fallbackRefs = append(fallbackRefs, refs.TagRef(k, name, e.Version))
	}
	if err := gitops.FetchPinnedCommit(e.Remote, refs.Ref(k, name), e.Commit, fallbackRefs...); err != nil {
		return fmt.Errorf("fetch %s: %w", e.Remote, err)
	}
	canonAbs := filepath.Join(repoRoot, e.Canonical)
	// In dev mode, preserve local edits to the canonical tree: only checkout
	// when the canonical path does not yet exist. Non-dev installs always
	// refresh the canonical tree to match the pinned commit.
	canonExists := false
	if info, err := os.Stat(canonAbs); err == nil && info.IsDir() {
		canonExists = true
	}
	if !(e.Dev && canonExists) {
		if err := os.RemoveAll(canonAbs); err != nil {
			return err
		}
		if err := os.MkdirAll(canonAbs, 0755); err != nil {
			return err
		}
		if err := git.ReadTreeToDir(e.Commit, canonAbs); err != nil {
			return fmt.Errorf("checkout tree: %w", err)
		}
	}
	matcher, err := assetignore.LoadFromTree(canonAbs)
	if err != nil {
		return fmt.Errorf("load .assetignore: %w", err)
	}
	mf, err := manifest.Load(canonAbs)
	if err != nil {
		return fmt.Errorf("load %s: %w", manifest.Filename, err)
	}
	// Iterate in deterministic order so logs and side effects are stable.
	for _, rt := range sortedRuntimes(e.Runtimes) {
		override := e.Runtimes[rt]
		mapping, err := resolveMapping(reg, rt, k, name, mf, override)
		if err != nil {
			return err
		}
		if err := materialize(canonAbs, repoRoot, mapping, e.Dev, matcher); err != nil {
			return fmt.Errorf("runtime %s: %w", rt, err)
		}
	}
	fmt.Fprintf(stdout, "  %s %s @ %s\n", k, name, e.Version)
	return nil
}

// resolveMapping returns the effective Mapping for a runtime by layering,
// from lowest to highest precedence: built-in registry → user/project
// runtimes.yaml (already merged into reg) → asset manifest
// (git-skill.yaml) → lock entry override (assets.json runtimes.<name>).
//
// "Manifest-only" runtime: declared in neither the built-in registry
// nor the user/project runtimes.yaml. In that case the asset manifest
// must supply at least a To; an empty From defaults to "." (the whole
// canonical tree), matching registry convention. If no source declares
// the runtime, resolution fails with reg's "unknown runtime" error.
//
// "<name>" placeholders in override values are substituted with name
// to match registry behavior, so consumers can write paths like
// ".custom/<name>/" in --target or assets.json.
func resolveMapping(reg *runtimes.Registry, rt string, k kind.Kind, name string, mf *manifest.Manifest, override state.RuntimeOverride) (runtimes.Mapping, error) {
	base, regErr := reg.Resolve(rt, k, name)
	if regErr != nil {
		mapping, ok := mf.Mapping(rt, name)
		if !ok {
			return runtimes.Mapping{}, regErr
		}
		if mapping.To == "" {
			return runtimes.Mapping{}, fmt.Errorf("%s declares runtime %q without 'to'; cannot materialize", manifest.Filename, rt)
		}
		base = mapping
	} else {
		base = manifest.Apply(base, mf, rt, name)
	}
	if override.From != "" {
		base.From = strings.ReplaceAll(override.From, "<name>", name)
	}
	if override.To != "" {
		base.To = strings.ReplaceAll(override.To, "<name>", name)
	}
	return base, nil
}

// materialize realizes a single Mapping from canonAbs into repoRoot.
// Trailing "/" on To selects directory fanout; otherwise single-file.
// In dev mode files are symlinked/junctioned instead of copied.
func materialize(canonAbs, repoRoot string, m runtimes.Mapping, dev bool, matcher *assetignore.Matcher) error {
	from := m.From
	if from == "" {
		from = "."
	}
	srcAbs := filepath.Join(canonAbs, from)
	dstAbs := filepath.Join(repoRoot, m.To)
	info, err := os.Stat(srcAbs)
	if err != nil {
		return fmt.Errorf("source %q missing in canonical: %w", from, err)
	}
	if err := os.MkdirAll(filepath.Dir(dstAbs), 0755); err != nil {
		return err
	}
	if err := os.RemoveAll(dstAbs); err != nil {
		return err
	}
	if m.IsDir() {
		if !info.IsDir() {
			return fmt.Errorf("mapping target %q ends with '/' but source %q is a file", m.To, from)
		}
		if dev {
			if _, err := xfs.EnsureLink(srcAbs, dstAbs, true); err != nil {
				return fmt.Errorf("link %s: %w", dstAbs, err)
			}
			return nil
		}
		filter := func(rel string) bool { return matcher.Match(rel) }
		if err := xfs.CopyTree(srcAbs, dstAbs, filter); err != nil {
			return fmt.Errorf("copy %s: %w", dstAbs, err)
		}
		return nil
	}
	// Single-file mapping.
	if info.IsDir() {
		return fmt.Errorf("mapping target %q is a file but source %q is a directory", m.To, from)
	}
	if dev {
		if _, err := xfs.EnsureLink(srcAbs, dstAbs, false); err != nil {
			return fmt.Errorf("link %s: %w", dstAbs, err)
		}
		return nil
	}
	if err := xfs.CopyFile(srcAbs, dstAbs); err != nil {
		return fmt.Errorf("copy %s: %w", dstAbs, err)
	}
	return nil
}

func sortedRuntimes(m map[string]state.RuntimeOverride) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
