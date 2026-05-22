package main

import (
	"io"

	cmdcommands "github.com/niradler/git-skill/cmd/git-skill/commands"
)

func init() {
	register("init", func(p Profile, args []string, stdout, stderr io.Writer) error {
		return cmdcommands.Init(toCmdProfile(p), args, stdout, stderr)
	})
}

func toCmdProfile(p Profile) cmdcommands.Profile {
	return cmdcommands.Profile{Name: p.Name, DefaultKind: p.DefaultKind, RequireKind: p.RequireKind}
}
