package commands

import (
	"fmt"

	"github.com/niradler/git-skill/internal/kind"
)

// pickKind chooses the asset kind for a command invocation:
//   - explicit --kind flag wins if non-empty
//   - otherwise the profile's DefaultKind (skill/agent for narrow profiles,
//     and the generic git-asset profile MUST pass --kind explicitly)
func pickKind(p Profile, kindFlag string) (kind.Kind, error) {
	if kindFlag != "" {
		return kind.Parse(kindFlag)
	}
	if p.RequireKind {
		return p.DefaultKind, nil
	}
	return 0, fmt.Errorf("kind required (profile %q is generic; pass --kind skill|agent)", p.Name)
}
