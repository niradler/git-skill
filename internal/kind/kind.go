// Package kind defines the asset kinds tracked by git-skill.
//
// Two values: Skill and Agent. Kind is determined per-asset by the 4-tier
// discriminator (see Resolve in this package): lock structural position >
// commit trailer > frontmatter > marker filename. See spec L3.
package kind

import (
	"fmt"
	"strings"
)

type Kind uint8

const (
	_ Kind = iota
	Skill
	Agent
)

func Parse(s string) (Kind, error) {
	if s != strings.TrimSpace(s) {
		return 0, fmt.Errorf("kind %q has surrounding whitespace", s)
	}
	switch strings.ToLower(s) {
	case "skill":
		return Skill, nil
	case "agent":
		return Agent, nil
	}
	return 0, fmt.Errorf("unknown kind %q (expected skill or agent)", s)
}

func (k Kind) String() string {
	switch k {
	case Skill:
		return "skill"
	case Agent:
		return "agent"
	}
	return ""
}

func All() []Kind { return []Kind{Skill, Agent} }
