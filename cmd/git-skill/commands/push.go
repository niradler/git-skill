package commands

import (
	"errors"
	"flag"
	"fmt"
	"io"

	"github.com/niradler/git-skill/internal/git"
	"github.com/niradler/git-skill/internal/kind"
	"github.com/niradler/git-skill/internal/refs"
)

func Push(p Profile, args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("push", flag.ContinueOnError)
	fs.SetOutput(stderr)
	if err := fs.Parse(args); err != nil {
		return err
	}
	remote := "origin"
	if fs.NArg() > 1 {
		return errors.New("push [remote]")
	}
	if fs.NArg() == 1 {
		remote = fs.Arg(0)
	}

	kinds := profileKinds(p)
	if len(kinds) == 0 {
		return errors.New("profile has no kinds to push")
	}

	pushArgs := []string{"push", remote}
	for _, k := range kinds {
		pushArgs = append(pushArgs, refs.KindPushRefspec(k), refs.KindPushTagRefspec(k))
	}
	out, err := git.Run(pushArgs...)
	if err != nil {
		return err
	}
	if out != "" {
		fmt.Fprintln(stdout, out)
	}
	fmt.Fprintf(stdout, "pushed to %s\n", remote)
	return nil
}

// profileKinds returns the kinds this profile operates on.
// Skill-only/agent-only profiles return their single kind; the generic
// git-asset profile returns both.
func profileKinds(p Profile) []kind.Kind {
	if p.RequireKind {
		return []kind.Kind{p.DefaultKind}
	}
	return []kind.Kind{kind.Skill, kind.Agent}
}
