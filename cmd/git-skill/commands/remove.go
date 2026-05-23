package commands

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/niradler/git-skill/internal/manifest"
	"github.com/niradler/git-skill/internal/runtimes"
	"github.com/niradler/git-skill/internal/state"
)

func Remove(p Profile, args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("remove", flag.ContinueOnError)
	fs.SetOutput(stderr)
	kindFlag := fs.String("kind", "", "asset kind (skill|agent); inferred from profile when omitted")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return errors.New("remove <name>")
	}
	name := fs.Arg(0)
	k, err := pickKind(p, *kindFlag)
	if err != nil {
		return err
	}
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	st, err := state.Read(cwd)
	if err != nil {
		return err
	}
	entry, ok := st.Get(k, name)
	if !ok {
		return fmt.Errorf("%s/%s not in %s", k, name, state.Filename)
	}
	canonAbs := filepath.Join(cwd, entry.Canonical)
	// Load the manifest before deleting canonical, so we can clean up
	// fan-out paths declared by manifest-only runtimes. Manifest errors
	// at this stage are non-fatal — the lock entry still authoritatively
	// lists which runtimes were installed, and stale fan-out is cheap.
	mf, _ := manifest.Load(canonAbs)
	if err := os.RemoveAll(canonAbs); err != nil {
		return fmt.Errorf("remove canonical: %w", err)
	}
	for rt, override := range entry.Runtimes {
		mapping, regErr := runtimes.Resolve(rt, k, name)
		if regErr != nil {
			if mf == nil {
				continue
			}
			mfm, ok := mf.Mapping(rt, name)
			if !ok || mfm.To == "" {
				continue
			}
			mapping = mfm
		} else {
			mapping = manifest.Apply(mapping, mf, rt, name)
		}
		if override.To != "" {
			mapping.To = strings.ReplaceAll(override.To, "<name>", name)
		}
		_ = os.RemoveAll(filepath.Join(cwd, mapping.To))
	}
	st.Delete(k, name)
	if err := st.Write(cwd); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "removed %s %s\n", k, name)
	return nil
}
