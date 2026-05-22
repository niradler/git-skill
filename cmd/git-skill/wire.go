package main

import (
	"io"

	cmdcommands "github.com/niradler/git-skill/cmd/git-skill/commands"
)

func init() {
	register("init", func(p Profile, args []string, stdout, stderr io.Writer) error {
		return cmdcommands.Init(toCmdProfile(p), args, stdout, stderr)
	})
	register("commit", func(p Profile, args []string, stdout, stderr io.Writer) error {
		return cmdcommands.Commit(toCmdProfile(p), args, stdout, stderr)
	})
	register("tag", func(p Profile, args []string, stdout, stderr io.Writer) error {
		return cmdcommands.Tag(toCmdProfile(p), args, stdout, stderr)
	})
	register("push", func(p Profile, args []string, stdout, stderr io.Writer) error {
		return cmdcommands.Push(toCmdProfile(p), args, stdout, stderr)
	})
	register("fetch", func(p Profile, args []string, stdout, stderr io.Writer) error {
		return cmdcommands.Fetch(toCmdProfile(p), args, stdout, stderr)
	})
	register("list", func(p Profile, args []string, stdout, stderr io.Writer) error {
		return cmdcommands.List(toCmdProfile(p), args, stdout, stderr)
	})
	register("log", func(p Profile, args []string, stdout, stderr io.Writer) error {
		return cmdcommands.Log(toCmdProfile(p), args, stdout, stderr)
	})
	register("diff", func(p Profile, args []string, stdout, stderr io.Writer) error {
		return cmdcommands.Diff(toCmdProfile(p), args, stdout, stderr)
	})
	register("show", func(p Profile, args []string, stdout, stderr io.Writer) error {
		return cmdcommands.Show(toCmdProfile(p), args, stdout, stderr)
	})
	register("install", func(p Profile, args []string, stdout, stderr io.Writer) error {
		return cmdcommands.Install(toCmdProfile(p), args, stdout, stderr)
	})
	register("add", func(p Profile, args []string, stdout, stderr io.Writer) error {
		return cmdcommands.Add(toCmdProfile(p), args, stdout, stderr)
	})
	register("update", func(p Profile, args []string, stdout, stderr io.Writer) error {
		return cmdcommands.Update(toCmdProfile(p), args, stdout, stderr)
	})
	register("remove", func(p Profile, args []string, stdout, stderr io.Writer) error {
		return cmdcommands.Remove(toCmdProfile(p), args, stdout, stderr)
	})
}

func toCmdProfile(p Profile) cmdcommands.Profile {
	return cmdcommands.Profile{Name: p.Name, DefaultKind: p.DefaultKind, RequireKind: p.RequireKind}
}
