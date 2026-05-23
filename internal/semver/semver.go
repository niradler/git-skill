// Package semver implements a minimal SemVer 2.0 comparator for git-skill tags.
package semver

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type Version struct {
	Major, Minor, Patch int
	Pre, Build          string
}

func (v Version) String() string {
	s := fmt.Sprintf("v%d.%d.%d", v.Major, v.Minor, v.Patch)
	if v.Pre != "" {
		s += "-" + v.Pre
	}
	if v.Build != "" {
		s += "+" + v.Build
	}
	return s
}

var versionRe = regexp.MustCompile(`^v(\d+)\.(\d+)\.(\d+)(?:-([0-9A-Za-z.-]+))?(?:\+([0-9A-Za-z.-]+))?$`)

func Parse(s string) (Version, error) {
	m := versionRe.FindStringSubmatch(s)
	if m == nil {
		return Version{}, fmt.Errorf("invalid version %q (expected vMAJOR.MINOR.PATCH)", s)
	}
	major, _ := strconv.Atoi(m[1])
	minor, _ := strconv.Atoi(m[2])
	patch, _ := strconv.Atoi(m[3])
	return Version{Major: major, Minor: minor, Patch: patch, Pre: m[4], Build: m[5]}, nil
}

func (v Version) Compare(o Version) int {
	if c := cmpInt(v.Major, o.Major); c != 0 {
		return c
	}
	if c := cmpInt(v.Minor, o.Minor); c != 0 {
		return c
	}
	if c := cmpInt(v.Patch, o.Patch); c != 0 {
		return c
	}
	if v.Pre == "" && o.Pre != "" {
		return 1
	}
	if v.Pre != "" && o.Pre == "" {
		return -1
	}
	if v.Pre == o.Pre {
		return 0
	}
	return comparePre(v.Pre, o.Pre)
}

func cmpInt(a, b int) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	}
	return 0
}

func comparePre(a, b string) int {
	ai := strings.Split(a, ".")
	bi := strings.Split(b, ".")
	for i := 0; i < len(ai) && i < len(bi); i++ {
		an, aErr := strconv.Atoi(ai[i])
		bn, bErr := strconv.Atoi(bi[i])
		if aErr == nil && bErr == nil {
			if c := cmpInt(an, bn); c != 0 {
				return c
			}
			continue
		}
		if aErr == nil {
			return -1
		}
		if bErr == nil {
			return 1
		}
		if c := strings.Compare(ai[i], bi[i]); c != 0 {
			return c
		}
	}
	return cmpInt(len(ai), len(bi))
}

type Op int

const (
	OpExact Op = iota
	OpCaret
	OpTilde
	OpGTE
	OpLatest
)

type Spec struct {
	Op      Op
	Version Version
}

func ParseSpec(s string) (Spec, error) {
	s = strings.TrimSpace(s)
	if s == "latest" || s == "*" {
		return Spec{Op: OpLatest}, nil
	}
	var op Op
	var rest string
	switch {
	case strings.HasPrefix(s, "^"):
		op, rest = OpCaret, s[1:]
	case strings.HasPrefix(s, "~"):
		op, rest = OpTilde, s[1:]
	case strings.HasPrefix(s, ">="):
		op, rest = OpGTE, s[2:]
	case strings.HasPrefix(s, "="):
		op, rest = OpExact, s[1:]
	default:
		op, rest = OpExact, s
	}
	v, err := Parse(rest)
	if err != nil {
		return Spec{}, err
	}
	return Spec{Op: op, Version: v}, nil
}

func Match(spec Spec, v Version) bool {
	switch spec.Op {
	case OpLatest:
		return true
	case OpExact:
		return v.Compare(spec.Version) == 0
	case OpGTE:
		return v.Compare(spec.Version) >= 0
	case OpCaret:
		return v.Major == spec.Version.Major && v.Compare(spec.Version) >= 0
	case OpTilde:
		return v.Major == spec.Version.Major && v.Minor == spec.Version.Minor && v.Compare(spec.Version) >= 0
	}
	return false
}

func Best(spec Spec, versions []Version) *Version {
	var best *Version
	for i := range versions {
		v := versions[i]
		if !Match(spec, v) {
			continue
		}
		if best == nil || v.Compare(*best) > 0 {
			best = &versions[i]
		}
	}
	return best
}
