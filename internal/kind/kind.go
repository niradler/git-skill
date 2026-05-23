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

// Sources holds optional kind values from each of the 4 tiers (spec L3).
// Zero value (Kind == 0) means "this tier did not resolve."
type Sources struct {
	Lock        Kind
	Trailer     Kind
	Frontmatter Kind
	Filename    Kind
}

type Resolution struct {
	Kind     Kind
	Tier     string   // "lock" | "trailer" | "frontmatter" | "filename"
	Warnings []string // e.g. trailer/frontmatter disagreement
}

// Resolve applies precedence (lock > trailer > frontmatter > filename) and
// surfaces conflict warnings between trailer and frontmatter.
func Resolve(s Sources) (*Resolution, error) {
	r := &Resolution{}
	switch {
	case s.Lock != 0:
		r.Kind, r.Tier = s.Lock, "lock"
	case s.Trailer != 0:
		r.Kind, r.Tier = s.Trailer, "trailer"
	case s.Frontmatter != 0:
		r.Kind, r.Tier = s.Frontmatter, "frontmatter"
	case s.Filename != 0:
		r.Kind, r.Tier = s.Filename, "filename"
	default:
		return nil, fmt.Errorf("cannot determine asset kind: no source resolved")
	}
	if s.Trailer != 0 && s.Frontmatter != 0 && s.Trailer != s.Frontmatter {
		r.Warnings = append(r.Warnings,
			fmt.Sprintf("trailer disagrees with frontmatter (trailer=%s, frontmatter=%s) - was the file hand-edited post-commit?",
				s.Trailer, s.Frontmatter))
	}
	return r, nil
}

// FromFilename maps a marker filename to a Kind (case-insensitive).
// "SKILL.md" → Skill, "AGENT.md" → Agent, anything else → 0.
func FromFilename(basename string) Kind {
	switch strings.ToLower(basename) {
	case "skill.md":
		return Skill
	case "agent.md":
		return Agent
	}
	return 0
}
