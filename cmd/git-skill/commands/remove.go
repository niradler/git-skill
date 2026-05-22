package commands

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

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
	cwd, _ := os.Getwd()
	st, err := state.Read(cwd)
	if err != nil {
		return err
	}
	entry, ok := st.Get(k, name)
	if !ok {
		return fmt.Errorf("%s/%s not in %s", k, name, state.Filename)
	}
	if err := os.RemoveAll(filepath.Join(cwd, entry.Canonical)); err != nil {
		return fmt.Errorf("remove canonical: %w", err)
	}
	for _, rt := range entry.Runtimes {
		target, err := runtimes.Resolve(rt, k, name)
		if err != nil {
			continue
		}
		_ = os.RemoveAll(filepath.Join(cwd, target))
	}
	st.Delete(k, name)
	if err := st.Write(cwd); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "removed %s %s\n", k, name)
	return nil
}
