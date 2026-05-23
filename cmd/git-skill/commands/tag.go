package commands

import (
	"errors"
	"flag"
	"fmt"
	"io"

	"github.com/niradler/git-skill/internal/git"
	"github.com/niradler/git-skill/internal/kind"
	"github.com/niradler/git-skill/internal/refs"
	"github.com/niradler/git-skill/internal/semver"
)

func Tag(p Profile, args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("tag", flag.ContinueOnError)
	fs.SetOutput(stderr)
	kindFlag := fs.String("kind", "", "asset kind (skill|agent); inferred from existing ref when omitted")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 2 {
		return errors.New("tag <name> v<semver>")
	}
	name, version := fs.Arg(0), fs.Arg(1)
	if _, err := semver.Parse(version); err != nil {
		return err
	}

	k, commit, err := resolveTagKind(name, *kindFlag)
	if err != nil {
		return err
	}

	tagRef := refs.TagRef(k, name, version)
	if err := git.UpdateRef(tagRef, commit); err != nil {
		return fmt.Errorf("update-ref %s: %w", tagRef, err)
	}
	fmt.Fprintf(stdout, "tagged %s %s -> %s\n", k, name, version)
	return nil
}

func resolveTagKind(name, kindFlag string) (kind.Kind, string, error) {
	if kindFlag != "" {
		k, err := kind.Parse(kindFlag)
		if err != nil {
			return 0, "", err
		}
		commit, err := git.ResolveRef(refs.Ref(k, name))
		if err != nil {
			return 0, "", fmt.Errorf("%s does not exist (commit the asset first)", refs.Ref(k, name))
		}
		return k, commit, nil
	}

	skillCommit, skillErr := git.ResolveRef(refs.Ref(kind.Skill, name))
	agentCommit, agentErr := git.ResolveRef(refs.Ref(kind.Agent, name))
	skillOK := skillErr == nil
	agentOK := agentErr == nil

	switch {
	case skillOK && agentOK:
		return 0, "", fmt.Errorf("%s exists as both skill and agent; pass --kind to disambiguate", name)
	case skillOK:
		return kind.Skill, skillCommit, nil
	case agentOK:
		return kind.Agent, agentCommit, nil
	default:
		return 0, "", fmt.Errorf("no asset named %s (commit the asset first)", name)
	}
}
