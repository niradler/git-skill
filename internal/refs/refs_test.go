package refs

import (
	"strings"
	"testing"
)

func TestRef(t *testing.T) {
	tests := []struct{ name, want string }{
		{"frontend-design", "refs/skills/frontend-design"},
		{"nir/boxy", "refs/skills/nir/boxy"},
		{"a", "refs/skills/a"},
		{"org/team/skill", "refs/skills/org/team/skill"},
	}
	for _, tt := range tests {
		if got := Ref(tt.name); got != tt.want {
			t.Errorf("Ref(%q) = %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestTagRef(t *testing.T) {
	tests := []struct{ name, ver, want string }{
		{"frontend-design", "1.0.0", "refs/skill-tags/frontend-design/v1.0.0"},
		{"frontend-design", "v1.0.0", "refs/skill-tags/frontend-design/v1.0.0"},
		{"nir/boxy", "2.3.4", "refs/skill-tags/nir/boxy/v2.3.4"},
		{"a", "v0.1.0", "refs/skill-tags/a/v0.1.0"},
		{"x", "10.20.30", "refs/skill-tags/x/v10.20.30"},
	}
	for _, tt := range tests {
		if got := TagRef(tt.name, tt.ver); got != tt.want {
			t.Errorf("TagRef(%q, %q) = %q, want %q", tt.name, tt.ver, got, tt.want)
		}
	}
}

func TestTagRef_AlreadyPrefixed(t *testing.T) {
	// v-prefixed input must not double-prefix
	got := TagRef("skill", "v2.0.0")
	want := "refs/skill-tags/skill/v2.0.0"
	if got != want {
		t.Errorf("TagRef with v-prefix: got %q, want %q", got, want)
	}
}

func TestPattern(t *testing.T) {
	want := "refs/skills/*"
	if got := Pattern(); got != want {
		t.Errorf("Pattern() = %q, want %q", got, want)
	}
}

func TestPushRefspec(t *testing.T) {
	want := "refs/skills/*:refs/skills/*"
	if got := PushRefspec(); got != want {
		t.Errorf("PushRefspec() = %q, want %q", got, want)
	}
}

func TestFetchRefspec(t *testing.T) {
	want := "+refs/skills/*:refs/skills/*"
	if got := FetchRefspec(); got != want {
		t.Errorf("FetchRefspec() = %q, want %q", got, want)
	}
}

func TestFetchTagRefspec(t *testing.T) {
	want := "+refs/skill-tags/*:refs/skill-tags/*"
	if got := FetchTagRefspec(); got != want {
		t.Errorf("FetchTagRefspec() = %q, want %q", got, want)
	}
}

func TestPrefixConstants(t *testing.T) {
	if Prefix != "refs/skills/" {
		t.Errorf("Prefix = %q", Prefix)
	}
	if TagPrefix != "refs/skill-tags/" {
		t.Errorf("TagPrefix = %q", TagPrefix)
	}
}

func TestValidateName(t *testing.T) {
	ok := []string{
		"a",
		"frontend-design",
		"acme/onboarding",
		"org/team/skill-name",
		"with_under_score",
		"123-skill",
	}
	for _, n := range ok {
		if err := ValidateName(n); err != nil {
			t.Errorf("ValidateName(%q) unexpected error: %v", n, err)
		}
	}

	bad := []string{
		"",
		"..",
		".",
		"foo/../bar",
		"foo/./bar",
		"/leading",
		"trailing/",
		"foo//bar",
		"UPPER",
		"with space",
		"with.dot",
		"emoji-💀",
		"-leading-hyphen",
		"_leading-under",
	}
	for _, n := range bad {
		if err := ValidateName(n); err == nil {
			t.Errorf("ValidateName(%q) expected error, got nil", n)
		}
	}
}

func TestValidateName_TooLong(t *testing.T) {
	long := strings.Repeat("a", 129)
	if err := ValidateName(long); err == nil {
		t.Error("expected error for >128 char name")
	}
}

func TestValidateVersion(t *testing.T) {
	ok := []string{
		"1.0.0",
		"v1.0.0",
		"10.20.30",
		"v1.0.0-rc1",
		"1.2.3-beta.2",
		"v0.1.0-alpha",
	}
	for _, v := range ok {
		if err := ValidateVersion(v); err != nil {
			t.Errorf("ValidateVersion(%q) unexpected error: %v", v, err)
		}
	}

	bad := []string{
		"",
		"1.0",
		"v1.0",
		"1",
		"latest",
		"vX.Y.Z",
		"1.0.0 ",
		" 1.0.0",
		"1.0.0/extra",
	}
	for _, v := range bad {
		if err := ValidateVersion(v); err == nil {
			t.Errorf("ValidateVersion(%q) expected error, got nil", v)
		}
	}
}
