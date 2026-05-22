package commands

import (
	"errors"
	"flag"
	"fmt"
	"io"

	"github.com/niradler/git-skill/internal/git"
	"github.com/niradler/git-skill/internal/refs"
)

func Log(p Profile, args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("log", flag.ContinueOnError)
	fs.SetOutput(stderr)
	kindFlag := fs.String("kind", "", "skill|agent (inferred when omitted)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return errors.New("log <name>")
	}
	name := fs.Arg(0)
	k, err := resolveAssetKind(name, *kindFlag)
	if err != nil {
		return err
	}
	out, err := git.Log(refs.Ref(k, name), "%h %s", 0)
	if err != nil {
		return err
	}
	fmt.Fprintln(stdout, out)
	return nil
}
