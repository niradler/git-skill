package main

import (
	"path/filepath"
	"strings"

	"github.com/niradler/git-skill/internal/kind"
)

// Profile describes the behavior implied by the executable name.
//
//	git-skill  → DefaultKind=Skill, RequireKind=true   (operates on skills only)
//	git-agent  → DefaultKind=Agent, RequireKind=true   (operates on agents only)
//	git-asset  → DefaultKind=0,     RequireKind=false  (both kinds, must be specified)
//
// See spec L2.
type Profile struct {
	Name        string
	DefaultKind kind.Kind
	RequireKind bool
}

func ProfileFromArgv0(argv0 string) Profile {
	// Normalize Windows separators so a path captured on Windows
	// resolves the same way when tests cross-validate on other OSes.
	normalized := strings.ReplaceAll(argv0, `\`, `/`)
	base := strings.TrimSuffix(filepath.Base(normalized), ".exe")
	switch base {
	case "git-skill":
		return Profile{Name: "git-skill", DefaultKind: kind.Skill, RequireKind: true}
	case "git-agent":
		return Profile{Name: "git-agent", DefaultKind: kind.Agent, RequireKind: true}
	default:
		return Profile{Name: "git-asset", RequireKind: false}
	}
}
