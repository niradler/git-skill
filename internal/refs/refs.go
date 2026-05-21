package refs

import (
	"fmt"
	"regexp"
	"strings"
)

const (
	// Prefix is the ref namespace for all skills.
	Prefix = "refs/skills/"

	// TagPrefix is for versioned skill releases.
	TagPrefix = "refs/skill-tags/"
)

// nameSegment matches a single segment of a skill name:
// lowercase letters, digits, hyphens, underscores; must start with a letter or digit.
var nameSegment = regexp.MustCompile(`^[a-z0-9][a-z0-9_\-]*$`)

// ValidateName rejects skill names that would create unsafe filesystem paths,
// produce invalid git refs, or trip path traversal. Allowed: one or more
// `[a-z0-9][a-z0-9_-]*` segments joined by `/`.
//
// Examples:
//
//	"frontend-design"        → ok
//	"acme/onboarding"        → ok (namespaced)
//	".."                     → rejected
//	"foo/../bar"             → rejected
//	"My Skill"               → rejected (uppercase, space)
func ValidateName(name string) error {
	if name == "" {
		return fmt.Errorf("skill name is empty")
	}
	if len(name) > 128 {
		return fmt.Errorf("skill name too long (max 128 chars)")
	}
	if strings.HasPrefix(name, "/") || strings.HasSuffix(name, "/") {
		return fmt.Errorf("skill name must not start or end with %q", "/")
	}
	if strings.Contains(name, "//") {
		return fmt.Errorf("skill name must not contain empty segments")
	}
	for _, seg := range strings.Split(name, "/") {
		if seg == "." || seg == ".." {
			return fmt.Errorf("skill name segment %q is not allowed", seg)
		}
		if !nameSegment.MatchString(seg) {
			return fmt.Errorf("skill name segment %q must match [a-z0-9][a-z0-9_-]*", seg)
		}
	}
	return nil
}

// semverRe accepts vX.Y.Z with an optional pre-release tag like "-rc1" or "-beta.2".
// Matches `git skill tag` input both with and without the leading `v`.
var semverRe = regexp.MustCompile(`^v?\d+\.\d+\.\d+(?:-[A-Za-z0-9.\-]+)?$`)

// ValidateVersion rejects version strings that are not SemVer-shaped.
// Accepts `1.2.3`, `v1.2.3`, `1.2.3-rc1`, `v1.2.3-beta.2`.
func ValidateVersion(version string) error {
	if version == "" {
		return fmt.Errorf("version is empty")
	}
	if !semverRe.MatchString(version) {
		return fmt.Errorf("version %q is not SemVer (expected vX.Y.Z, e.g. v1.0.0)", version)
	}
	return nil
}

// Ref returns the full ref path for a skill.
// Caller must have validated the name via ValidateName first.
//
//	"frontend-design" → "refs/skills/frontend-design"
//	"nir/boxy"        → "refs/skills/nir/boxy"
func Ref(name string) string {
	return Prefix + name
}

// TagRef returns the ref for a tagged skill version.
// Caller must have validated name and version first.
//
//	("frontend-design", "1.2.0") → "refs/skill-tags/frontend-design/v1.2.0"
func TagRef(name, version string) string {
	v := version
	if !strings.HasPrefix(v, "v") {
		v = "v" + v
	}
	return fmt.Sprintf("%s%s/%s", TagPrefix, name, v)
}

// Pattern returns a glob for listing all skills.
func Pattern() string {
	return Prefix + "*"
}

// Refspec returns push/fetch refspecs for skills.
func PushRefspec() string {
	return Prefix + "*:" + Prefix + "*"
}

func FetchRefspec() string {
	return "+" + Prefix + "*:" + Prefix + "*"
}

func FetchTagRefspec() string {
	return "+" + TagPrefix + "*:" + TagPrefix + "*"
}
