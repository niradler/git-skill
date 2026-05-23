package refs

import (
	"strings"
	"testing"

	"github.com/niradler/git-skill/internal/kind"
)

func TestRef(t *testing.T) {
	tests := []struct {
		k    kind.Kind
		name string
		want string
	}{
		{kind.Skill, "frontend-design", "refs/assets/skill/frontend-design"},
		{kind.Skill, "acme/security-review", "refs/assets/skill/acme/security-review"},
		{kind.Agent, "nir/code-reviewer", "refs/assets/agent/nir/code-reviewer"},
		{kind.Agent, "a", "refs/assets/agent/a"},
	}
	for _, tt := range tests {
		if got := Ref(tt.k, tt.name); got != tt.want {
			t.Errorf("Ref(%v, %q) = %q, want %q", tt.k, tt.name, got, tt.want)
		}
	}
}

func TestTagRef(t *testing.T) {
	tests := []struct {
		k    kind.Kind
		name string
		ver  string
		want string
	}{
		{kind.Skill, "foo", "1.2.0", "refs/asset-tags/skill/foo/v1.2.0"},
		{kind.Skill, "foo", "v1.2.0", "refs/asset-tags/skill/foo/v1.2.0"},
		{kind.Agent, "nir/code-reviewer", "0.3.0", "refs/asset-tags/agent/nir/code-reviewer/v0.3.0"},
	}
	for _, tt := range tests {
		if got := TagRef(tt.k, tt.name, tt.ver); got != tt.want {
			t.Errorf("TagRef(%v, %q, %q) = %q, want %q", tt.k, tt.name, tt.ver, got, tt.want)
		}
	}
}

func TestPrefixes(t *testing.T) {
	if Prefix != "refs/assets/" {
		t.Errorf("Prefix = %q", Prefix)
	}
	if TagPrefix != "refs/asset-tags/" {
		t.Errorf("TagPrefix = %q", TagPrefix)
	}
	if KindPrefix(kind.Skill) != "refs/assets/skill/" {
		t.Errorf("KindPrefix(skill) = %q", KindPrefix(kind.Skill))
	}
	if KindTagPrefix(kind.Agent) != "refs/asset-tags/agent/" {
		t.Errorf("KindTagPrefix(agent) = %q", KindTagPrefix(kind.Agent))
	}
}

func TestRefspecs(t *testing.T) {
	if PushRefspec() != "refs/assets/*:refs/assets/*" {
		t.Errorf("PushRefspec = %q", PushRefspec())
	}
	if FetchRefspec() != "+refs/assets/*:refs/assets/*" {
		t.Errorf("FetchRefspec = %q", FetchRefspec())
	}
	if PushTagRefspec() != "refs/asset-tags/*:refs/asset-tags/*" {
		t.Errorf("PushTagRefspec = %q", PushTagRefspec())
	}
	if FetchTagRefspec() != "+refs/asset-tags/*:refs/asset-tags/*" {
		t.Errorf("FetchTagRefspec = %q", FetchTagRefspec())
	}
	if KindPushRefspec(kind.Skill) != "refs/assets/skill/*:refs/assets/skill/*" {
		t.Errorf("KindPushRefspec(skill) = %q", KindPushRefspec(kind.Skill))
	}
}

func TestParseRef(t *testing.T) {
	k, name, err := ParseRef("refs/assets/skill/acme/foo")
	if err != nil || k != kind.Skill || name != "acme/foo" {
		t.Errorf("ParseRef = (%v, %q, %v)", k, name, err)
	}
	k, name, err = ParseRef("refs/assets/agent/bar")
	if err != nil || k != kind.Agent || name != "bar" {
		t.Errorf("ParseRef agent = (%v, %q, %v)", k, name, err)
	}
	if _, _, err := ParseRef("refs/heads/main"); err == nil {
		t.Error("ParseRef should reject non-asset ref")
	}
	if _, _, err := ParseRef("refs/assets/unknown/x"); err == nil {
		t.Error("ParseRef should reject unknown kind")
	}
}

func TestParseTagRef(t *testing.T) {
	k, name, ver, err := ParseTagRef("refs/asset-tags/skill/foo/v1.2.0")
	if err != nil || k != kind.Skill || name != "foo" || ver != "v1.2.0" {
		t.Errorf("ParseTagRef = (%v, %q, %q, %v)", k, name, ver, err)
	}
	k, name, ver, err = ParseTagRef("refs/asset-tags/agent/nir/code-reviewer/v0.3.0")
	if err != nil || k != kind.Agent || name != "nir/code-reviewer" || ver != "v0.3.0" {
		t.Errorf("ParseTagRef nested = (%v, %q, %q, %v)", k, name, ver, err)
	}
}

func TestValidateName(t *testing.T) {
	good := []string{"a", "frontend-design", "acme/onboarding", "org/team/skill"}
	for _, n := range good {
		if err := ValidateName(n); err != nil {
			t.Errorf("ValidateName(%q): %v", n, err)
		}
	}
	bad := []string{"", "..", "foo/../bar", "/leading", "trailing/", "UPPER", " spaces ", "with.dot"}
	for _, n := range bad {
		if err := ValidateName(n); err == nil {
			t.Errorf("ValidateName(%q) expected error", n)
		}
	}
	if err := ValidateName(strings.Repeat("a", 129)); err == nil {
		t.Error("expected error for 129-char name")
	}
}

func TestValidateVersion(t *testing.T) {
	good := []string{"1.0.0", "v1.0.0", "10.20.30", "v1.0.0-rc1", "1.2.3-beta.2"}
	for _, v := range good {
		if err := ValidateVersion(v); err != nil {
			t.Errorf("ValidateVersion(%q): %v", v, err)
		}
	}
	bad := []string{"", "1.0", "latest", "vX.Y.Z", " 1.0.0", "1.0.0 "}
	for _, v := range bad {
		if err := ValidateVersion(v); err == nil {
			t.Errorf("ValidateVersion(%q) expected error", v)
		}
	}
}
