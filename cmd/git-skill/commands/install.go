package commands

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/niradler/git-skill/internal/assetignore"
	xfs "github.com/niradler/git-skill/internal/fs"
	"github.com/niradler/git-skill/internal/git"
	"github.com/niradler/git-skill/internal/gitops"
	"github.com/niradler/git-skill/internal/kind"
	"github.com/niradler/git-skill/internal/runtimes"
	"github.com/niradler/git-skill/internal/state"
)

func Install(p Profile, args []string, stdout, stderr io.Writer) error {
	if len(args) > 0 {
		// Task 26 will replace this with: return Add(p, args, stdout, stderr).
		return errors.New("install <name> not yet supported (use 'add')")
	}
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	st, err := state.Read(cwd)
	if err != nil {
		return fmt.Errorf("read %s: %w", state.Filename, err)
	}
	for _, k := range kind.All() {
		if p.RequireKind && k != p.DefaultKind {
			continue
		}
		for name, entry := range st.Assets[k] {
			if err := installOne(cwd, st, k, name, entry, stdout); err != nil {
				return fmt.Errorf("install %s/%s: %w", k, name, err)
			}
		}
	}
	return nil
}

func installOne(repoRoot string, st *state.State, k kind.Kind, name string, e state.Entry, stdout io.Writer) error {
	if e.Commit == "" {
		return fmt.Errorf("entry has no commit pin (run 'update %s/%s' first)", k, name)
	}
	if err := gitops.FetchPinnedCommit(e.Remote, "refs/assets/"+k.String()+"/"+name, e.Commit); err != nil {
		return fmt.Errorf("fetch %s: %w", e.Remote, err)
	}
	canonAbs := filepath.Join(repoRoot, e.Canonical)
	if err := os.RemoveAll(canonAbs); err != nil {
		return err
	}
	if err := os.MkdirAll(canonAbs, 0755); err != nil {
		return err
	}
	if err := git.ReadTreeToDir(e.Commit, canonAbs); err != nil {
		return fmt.Errorf("checkout tree: %w", err)
	}
	matcher, err := assetignore.LoadFromTree(canonAbs)
	if err != nil {
		return fmt.Errorf("load .assetignore: %w", err)
	}
	for _, rt := range e.Runtimes {
		target, err := runtimes.Resolve(rt, k, name)
		if err != nil {
			return fmt.Errorf("runtime %s: %w", rt, err)
		}
		linkAbs := filepath.Join(repoRoot, target)
		switch k {
		case kind.Skill:
			if err := os.MkdirAll(filepath.Dir(linkAbs), 0755); err != nil {
				return err
			}
			if err := os.RemoveAll(linkAbs); err != nil {
				return err
			}
			if e.Dev {
				if _, err := xfs.EnsureLink(canonAbs, linkAbs, true); err != nil {
					return fmt.Errorf("link %s: %w", linkAbs, err)
				}
			} else {
				// matcher.Match returns true = should be ignored/excluded.
				// CopyTree filter returns true = skip. The semantics align directly.
				filter := func(rel string) bool { return matcher.Match(rel) }
				if err := xfs.CopyTree(canonAbs, linkAbs, filter); err != nil {
					return fmt.Errorf("copy %s: %w", linkAbs, err)
				}
			}
		case kind.Agent:
			markerSrc := filepath.Join(canonAbs, "AGENT.md")
			if _, err := os.Stat(markerSrc); err != nil {
				return fmt.Errorf("agent marker AGENT.md missing in %s", canonAbs)
			}
			if err := os.MkdirAll(filepath.Dir(linkAbs), 0755); err != nil {
				return err
			}
			if err := os.RemoveAll(linkAbs); err != nil {
				return err
			}
			if e.Dev {
				if _, err := xfs.EnsureLink(markerSrc, linkAbs, false); err != nil {
					return fmt.Errorf("link %s: %w", linkAbs, err)
				}
			} else {
				if err := xfs.CopyFile(markerSrc, linkAbs); err != nil {
					return fmt.Errorf("copy %s: %w", linkAbs, err)
				}
			}
		}
	}
	fmt.Fprintf(stdout, "  %s %s @ %s\n", k, name, e.Version)
	return nil
}
