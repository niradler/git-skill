// Package refs builds and parses git ref paths for assets.
//
// Layout (spec L7):
//
//	refs/assets/<kind>/<name>                  // asset branch
//	refs/asset-tags/<kind>/<name>/v<semver>    // immutable tagged release
//
// Kind is "skill" or "agent" (singular). See internal/kind.
package refs

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/niradler/git-skill/internal/kind"
)

const (
	Prefix    = "refs/assets/"
	TagPrefix = "refs/asset-tags/"
)

func KindPrefix(k kind.Kind) string    { return Prefix + k.String() + "/" }
func KindTagPrefix(k kind.Kind) string { return TagPrefix + k.String() + "/" }

func Ref(k kind.Kind, name string) string {
	return KindPrefix(k) + name
}

func TagRef(k kind.Kind, name, version string) string {
	v := version
	if !strings.HasPrefix(v, "v") {
		v = "v" + v
	}
	return KindTagPrefix(k) + name + "/" + v
}

func PushRefspec() string     { return Prefix + "*:" + Prefix + "*" }
func FetchRefspec() string    { return "+" + Prefix + "*:" + Prefix + "*" }
func PushTagRefspec() string  { return TagPrefix + "*:" + TagPrefix + "*" }
func FetchTagRefspec() string { return "+" + TagPrefix + "*:" + TagPrefix + "*" }

func KindPushRefspec(k kind.Kind) string {
	return KindPrefix(k) + "*:" + KindPrefix(k) + "*"
}
func KindFetchRefspec(k kind.Kind) string {
	return "+" + KindPrefix(k) + "*:" + KindPrefix(k) + "*"
}
func KindPushTagRefspec(k kind.Kind) string {
	return KindTagPrefix(k) + "*:" + KindTagPrefix(k) + "*"
}
func KindFetchTagRefspec(k kind.Kind) string {
	return "+" + KindTagPrefix(k) + "*:" + KindTagPrefix(k) + "*"
}

// ParseRef extracts kind + name from a ref under refs/assets/.
// Returns an error if the ref is not under the asset prefix or the kind segment
// is not recognized.
func ParseRef(ref string) (kind.Kind, string, error) {
	rest, ok := strings.CutPrefix(ref, Prefix)
	if !ok {
		return 0, "", fmt.Errorf("ref %q is not under %s", ref, Prefix)
	}
	seg, name, ok := strings.Cut(rest, "/")
	if !ok || name == "" {
		return 0, "", fmt.Errorf("ref %q missing kind or name", ref)
	}
	k, err := kind.Parse(seg)
	if err != nil {
		return 0, "", fmt.Errorf("ref %q: %w", ref, err)
	}
	return k, name, nil
}

// ParseTagRef extracts kind + name + version from a tag ref.
func ParseTagRef(ref string) (kind.Kind, string, string, error) {
	rest, ok := strings.CutPrefix(ref, TagPrefix)
	if !ok {
		return 0, "", "", fmt.Errorf("ref %q is not under %s", ref, TagPrefix)
	}
	seg, rest, ok := strings.Cut(rest, "/")
	if !ok {
		return 0, "", "", fmt.Errorf("tag ref %q missing kind", ref)
	}
	k, err := kind.Parse(seg)
	if err != nil {
		return 0, "", "", fmt.Errorf("tag ref %q: %w", ref, err)
	}
	// Last "/v..." is the version.
	idx := strings.LastIndex(rest, "/v")
	if idx < 0 {
		return 0, "", "", fmt.Errorf("tag ref %q missing version", ref)
	}
	name := rest[:idx]
	ver := rest[idx+1:]
	if name == "" || ver == "" {
		return 0, "", "", fmt.Errorf("tag ref %q has empty name or version", ref)
	}
	return k, name, ver, nil
}

var nameSegment = regexp.MustCompile(`^[a-z0-9][a-z0-9_\-]*$`)

func ValidateName(name string) error {
	if name == "" {
		return fmt.Errorf("asset name is empty")
	}
	if len(name) > 128 {
		return fmt.Errorf("asset name too long (max 128 chars)")
	}
	if strings.HasPrefix(name, "/") || strings.HasSuffix(name, "/") {
		return fmt.Errorf("asset name must not start or end with /")
	}
	if strings.Contains(name, "//") {
		return fmt.Errorf("asset name must not contain empty segments")
	}
	for _, seg := range strings.Split(name, "/") {
		if seg == "." || seg == ".." {
			return fmt.Errorf("asset name segment %q is not allowed", seg)
		}
		if !nameSegment.MatchString(seg) {
			return fmt.Errorf("asset name segment %q must match [a-z0-9][a-z0-9_-]*", seg)
		}
	}
	return nil
}

var semverRe = regexp.MustCompile(`^v?\d+\.\d+\.\d+(?:-[A-Za-z0-9.\-]+)?$`)

func ValidateVersion(version string) error {
	if version == "" {
		return fmt.Errorf("version is empty")
	}
	if !semverRe.MatchString(version) {
		return fmt.Errorf("version %q is not SemVer (expected vX.Y.Z, e.g. v1.0.0)", version)
	}
	return nil
}
