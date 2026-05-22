package commands

import "github.com/niradler/git-skill/internal/kind"

type Profile struct {
	Name        string
	DefaultKind kind.Kind
	RequireKind bool
}
