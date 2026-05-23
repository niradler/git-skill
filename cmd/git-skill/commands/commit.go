package commands

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/niradler/git-skill/internal/git"
	"github.com/niradler/git-skill/internal/gitops"
	"github.com/niradler/git-skill/internal/kind"
	"github.com/niradler/git-skill/internal/refs"
	"github.com/niradler/git-skill/internal/skill"
)

func Commit(p Profile, args []string, stdout, stderr io.Writer) error {
	// The name argument may appear before or after the flags (e.g. `commit <name> -m …`).
	// Standard flag.Parse stops at the first non-flag token, so we separate the name
	// from the flag arguments up-front: collect all non-flag tokens as positional args
	// and pass the rest to flag.Parse.
	var positional []string
	var flagArgs []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			positional = append(positional, args[i+1:]...)
			break
		}
		if len(a) > 1 && a[0] == '-' {
			// This is a flag token. If it is a known value flag (-m, --path, --kind)
			// without an embedded "=", the next token is its value.
			flagArgs = append(flagArgs, a)
			name := strings.TrimLeft(a, "-")
			if idx := strings.Index(name, "="); idx >= 0 {
				// value embedded; nothing extra to consume
			} else if name == "m" || name == "path" || name == "kind" {
				if i+1 < len(args) {
					i++
					flagArgs = append(flagArgs, args[i])
				}
			}
		} else {
			positional = append(positional, a)
		}
	}

	fs := flag.NewFlagSet("commit", flag.ContinueOnError)
	fs.SetOutput(stderr)
	msg := fs.String("m", "", "commit message (required)")
	path := fs.String("path", ".", "directory whose tree is committed")
	kindFlag := fs.String("kind", "", "asset kind (skill|agent); inferred when omitted")
	if err := fs.Parse(flagArgs); err != nil {
		return err
	}
	if len(positional) != 1 {
		return errors.New("commit requires exactly one asset name argument")
	}
	if *msg == "" {
		return errors.New("commit requires -m <message>")
	}
	name := positional[0]

	k, warnings, err := resolveCommitKind(p, *kindFlag, *path, name)
	if err != nil {
		return err
	}
	for _, w := range warnings {
		fmt.Fprintf(stderr, "warning: %s\n", w)
	}

	abs, err := absPath(*path)
	if err != nil {
		return err
	}

	tree, err := git.WriteTreeFromDir(abs)
	if err != nil {
		return fmt.Errorf("write tree: %w", err)
	}

	ref := refs.Ref(k, name)
	parent, _ := git.ResolveRef(ref)

	commit, err := gitops.WriteCommitWithKind(tree, *msg, k, parent)
	if err != nil {
		return fmt.Errorf("commit-tree: %w", err)
	}

	if err := git.UpdateRef(ref, commit); err != nil {
		return fmt.Errorf("update-ref %s: %w", ref, err)
	}
	fmt.Fprintf(stdout, "[%s %s] %s\n", k, name, commit[:10])
	return nil
}

// resolveCommitKind implements the 4-tier kind discriminator per spec L3/L5.
//  1. Existing ref kind (refs/assets/<kind>/<name>) — locks kind for life of the asset
//  2. --kind flag
//  3. Frontmatter kind: field
//  4. Marker filename (SKILL.md / AGENT.md)
//  5. Profile default (last resort)
func resolveCommitKind(p Profile, kindFlag, path, name string) (kind.Kind, []string, error) {
	var warnings []string
	abs, err := absPath(path)
	if err != nil {
		return 0, nil, err
	}

	var refKind kind.Kind
	var refFound bool
	for _, k := range kind.All() {
		if _, err := git.ResolveRef(refs.Ref(k, name)); err == nil {
			if refFound && refKind != k {
				return 0, warnings, fmt.Errorf(
					"name %q exists under both kinds (%s and %s); rename one before committing",
					name, refKind, k)
			}
			refKind, refFound = k, true
		}
	}

	var fmKind, fileKind kind.Kind
	var fmFound, fileFound bool
	if meta, err := skill.ReadFrontmatter(abs); err == nil && meta.Kind != "" {
		if k, err := kind.Parse(meta.Kind); err == nil {
			fmKind, fmFound = k, true
		}
	}
	if k, ok := skill.KindFromMarkerFile(abs); ok {
		fileKind, fileFound = k, true
	}
	if fmFound && fileFound && fmKind != fileKind {
		warnings = append(warnings, fmt.Sprintf(
			"frontmatter kind=%s disagrees with marker filename kind=%s; using frontmatter",
			fmKind, fileKind))
	}

	if refFound {
		if kindFlag != "" {
			if picked, err := kind.Parse(kindFlag); err == nil && picked != refKind {
				warnings = append(warnings, fmt.Sprintf(
					"--kind=%s ignored: existing ref pins kind=%s for %q",
					picked, refKind, name))
			}
		}
		if fmFound && fmKind != refKind {
			warnings = append(warnings, fmt.Sprintf(
				"frontmatter kind=%s ignored: existing ref pins kind=%s",
				fmKind, refKind))
		}
		return refKind, warnings, nil
	}

	if kindFlag != "" {
		picked, err := kind.Parse(kindFlag)
		if err != nil {
			return 0, warnings, err
		}
		if fmFound && fmKind != picked {
			warnings = append(warnings, fmt.Sprintf(
				"--kind=%s overrides frontmatter kind=%s", picked, fmKind))
		}
		return picked, warnings, nil
	}
	if fmFound {
		return fmKind, warnings, nil
	}
	if fileFound {
		return fileKind, warnings, nil
	}
	if p.RequireKind {
		warnings = append(warnings, fmt.Sprintf(
			"no kind signal in asset; defaulting to profile kind=%s", p.DefaultKind))
		return p.DefaultKind, warnings, nil
	}
	return 0, warnings, fmt.Errorf("cannot infer kind for %s (pass --kind)", name)
}

func absPath(p string) (string, error) {
	if p == "" || p == "." {
		return os.Getwd()
	}
	if filepath.IsAbs(p) {
		return p, nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.Join(cwd, p), nil
}
