package commands

import (
	"fmt"
	"io"
	"os"

	"github.com/niradler/git-skill/internal/gitops"
	"github.com/niradler/git-skill/internal/kind"
	"github.com/niradler/git-skill/internal/runtimes"
	"github.com/niradler/git-skill/internal/state"
)

func Update(p Profile, args []string, stdout, stderr io.Writer) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	st, err := state.Read(cwd)
	if err != nil {
		return err
	}
	reg, err := runtimes.LoadRegistry(cwd)
	if err != nil {
		return fmt.Errorf("load runtimes config: %w", err)
	}
	want := stringSet(args)

	for _, k := range kind.All() {
		if p.RequireKind && k != p.DefaultKind {
			continue
		}
		for name, entry := range st.Assets[k] {
			if len(want) > 0 && !want[name] {
				continue
			}
			remoteAssets, err := gitops.ListRemote(entry.Remote)
			if err != nil {
				return fmt.Errorf("ls-remote %s: %w", entry.Remote, err)
			}
			resolved, err := ResolveSpec(remoteAssets, k, name, entry.Spec)
			if err != nil {
				return fmt.Errorf("resolve %s/%s: %w", k, name, err)
			}
			entry.Version = resolved.Version
			entry.Commit = resolved.Commit
			st.Set(k, name, entry)
			if err := installOne(cwd, k, name, entry, reg, stdout); err != nil {
				return fmt.Errorf("install %s/%s: %w", k, name, err)
			}
		}
	}
	return st.Write(cwd)
}

func stringSet(ss []string) map[string]bool {
	if len(ss) == 0 {
		return nil
	}
	m := make(map[string]bool, len(ss))
	for _, s := range ss {
		m[s] = true
	}
	return m
}
