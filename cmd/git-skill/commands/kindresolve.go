package commands

import (
	"fmt"

	"github.com/niradler/git-skill/internal/git"
	"github.com/niradler/git-skill/internal/kind"
	"github.com/niradler/git-skill/internal/refs"
)

// resolveAssetKind picks the kind for a kind-agnostic command (log/diff/show/tag)
// by inspecting which kind's asset ref exists for the given name. If an explicit
// --kind override is supplied (non-empty), it's used directly. Errors when the
// name exists under both kinds without an override, or under neither.
func resolveAssetKind(name, kindFlag string) (kind.Kind, error) {
	if kindFlag != "" {
		k, err := kind.Parse(kindFlag)
		if err != nil {
			return 0, err
		}
		if _, err := git.ResolveRef(refs.Ref(k, name)); err != nil {
			return 0, fmt.Errorf("%s does not exist", refs.Ref(k, name))
		}
		return k, nil
	}
	_, skillErr := git.ResolveRef(refs.Ref(kind.Skill, name))
	_, agentErr := git.ResolveRef(refs.Ref(kind.Agent, name))
	skillOK := skillErr == nil
	agentOK := agentErr == nil
	switch {
	case skillOK && agentOK:
		return 0, fmt.Errorf("%s exists as both skill and agent; pass --kind to disambiguate", name)
	case skillOK:
		return kind.Skill, nil
	case agentOK:
		return kind.Agent, nil
	default:
		return 0, fmt.Errorf("no asset named %s", name)
	}
}
