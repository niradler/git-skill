package commands

import (
	"errors"
	"flag"
	"fmt"
	"io"

	"github.com/niradler/git-skill/internal/git"
	"github.com/niradler/git-skill/internal/refs"
)

func Diff(p Profile, args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("diff", flag.ContinueOnError)
	fs.SetOutput(stderr)
	kindFlag := fs.String("kind", "", "skill|agent (inferred when omitted)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 3 {
		return errors.New("diff <name> <fromTag> <toTag>")
	}
	name, from, to := fs.Arg(0), fs.Arg(1), fs.Arg(2)
	k, err := resolveAssetKind(name, *kindFlag)
	if err != nil {
		return err
	}
	out, err := git.DiffTree(refs.TagRef(k, name, from), refs.TagRef(k, name, to))
	if err != nil {
		return err
	}
	fmt.Fprintln(stdout, out)
	return nil
}
