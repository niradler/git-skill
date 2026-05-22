// Package main is the git-skill / git-agent / git-asset CLI entrypoint.
//
// One binary, three invocation names (see Profile in dispatch.go). The body
// of every subcommand lives in cmd/git-skill/commands/<name>.go.
package main

import (
	"fmt"
	"io"
	"os"
)

func main() {
	profile := ProfileFromArgv0(os.Args[0])
	os.Exit(Run(profile, os.Args[1:], os.Stdout, os.Stderr))
}

// Run executes one CLI invocation. Returned int is the process exit code.
// Split out so it is testable.
func Run(profile Profile, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		printHelp(profile, stdout)
		return 0
	}
	cmd, args := args[0], args[1:]
	h, ok := commands[cmd]
	if !ok {
		fmt.Fprintf(stderr, "%s: unknown command %q (try --help)\n", profile.Name, cmd)
		return 2
	}
	if err := h(profile, args, stdout, stderr); err != nil {
		fmt.Fprintf(stderr, "%s: %s: %s\n", profile.Name, cmd, err)
		return 1
	}
	return 0
}

type handler func(p Profile, args []string, stdout, stderr io.Writer) error

// commands is populated by per-command files via init() registrations.
var commands = map[string]handler{}

func register(name string, h handler) {
	if _, dup := commands[name]; dup {
		panic("duplicate command: " + name)
	}
	commands[name] = h
}

func printHelp(p Profile, w io.Writer) {
	fmt.Fprintf(w, "usage: %s <command> [options]\n\ncommands:\n", p.Name)
	for name := range commands {
		fmt.Fprintf(w, "  %s\n", name)
	}
}

// stubNames is the canonical command list. Phase 5/6 tasks each replace one
// stub with a real handler — they MUST also delete that name from this slice
// to avoid the duplicate-registration panic at startup.
var stubNames = []string{
	"commit", "tag", "push", "fetch", "list",
	"log", "diff", "show", "install", "add", "update",
	"remove", "discover",
}

func init() {
	for _, name := range stubNames {
		n := name
		register(n, func(p Profile, args []string, stdout, stderr io.Writer) error {
			return fmt.Errorf("%s not implemented yet", n)
		})
	}
}
