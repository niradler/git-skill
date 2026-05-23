package commands

import (
	"errors"
	"flag"
	"fmt"
	"io"

	"github.com/niradler/git-skill/internal/git"
	"github.com/niradler/git-skill/internal/refs"
)

func Fetch(p Profile, args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("fetch", flag.ContinueOnError)
	fs.SetOutput(stderr)
	if err := fs.Parse(args); err != nil {
		return err
	}
	remote := "origin"
	if fs.NArg() > 1 {
		return errors.New("fetch [remote]")
	}
	if fs.NArg() == 1 {
		remote = fs.Arg(0)
	}

	kinds := profileKinds(p)
	if len(kinds) == 0 {
		return errors.New("profile has no kinds to fetch")
	}

	fetchArgs := []string{"fetch", remote}
	for _, k := range kinds {
		fetchArgs = append(fetchArgs, refs.KindFetchRefspec(k), refs.KindFetchTagRefspec(k))
	}
	out, err := git.Run(fetchArgs...)
	if err != nil {
		return err
	}
	if out != "" {
		fmt.Fprintln(stdout, out)
	}
	fmt.Fprintf(stdout, "fetched from %s\n", remote)
	return nil
}
