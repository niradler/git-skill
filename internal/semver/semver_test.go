package semver

import "testing"

func TestParse(t *testing.T) {
	tests := []struct {
		in   string
		ok   bool
		want Version
	}{
		{"v1.2.3", true, Version{Major: 1, Minor: 2, Patch: 3}},
		{"v0.0.1", true, Version{Major: 0, Minor: 0, Patch: 1}},
		{"v10.20.30", true, Version{Major: 10, Minor: 20, Patch: 30}},
		{"v1.2.3-beta.1", true, Version{Major: 1, Minor: 2, Patch: 3, Pre: "beta.1"}},
		{"v1.2.3+build.4", true, Version{Major: 1, Minor: 2, Patch: 3, Build: "build.4"}},
		{"v1.2.3-rc.1+exp.sha.5114f85", true, Version{Major: 1, Minor: 2, Patch: 3, Pre: "rc.1", Build: "exp.sha.5114f85"}},
		{"1.2.3", false, Version{}},
		{"v1.2", false, Version{}},
		{"vX.Y.Z", false, Version{}},
		{"", false, Version{}},
	}
	for _, tt := range tests {
		got, err := Parse(tt.in)
		if tt.ok && err != nil {
			t.Errorf("Parse(%q) error: %v", tt.in, err)
			continue
		}
		if !tt.ok && err == nil {
			t.Errorf("Parse(%q): expected error", tt.in)
			continue
		}
		if tt.ok && got != tt.want {
			t.Errorf("Parse(%q) = %+v, want %+v", tt.in, got, tt.want)
		}
	}
}

func TestCompareNumeric(t *testing.T) {
	v2, _ := Parse("v2.0.0")
	v10, _ := Parse("v10.0.0")
	if v2.Compare(v10) >= 0 {
		t.Errorf("v2.0.0 must be < v10.0.0; got Compare = %d", v2.Compare(v10))
	}

	cases := []struct {
		a, b string
		want int
	}{
		{"v1.2.3", "v1.2.3", 0},
		{"v1.2.3", "v1.2.4", -1},
		{"v1.3.0", "v1.2.9", 1},
		{"v2.0.0", "v1.99.99", 1},
		{"v1.0.0", "v1.0.0-rc.1", 1},
		{"v1.0.0-alpha", "v1.0.0-beta", -1},
		{"v1.0.0-alpha.1", "v1.0.0-alpha.2", -1},
		{"v1.0.0-alpha.2", "v1.0.0-alpha.10", -1},
		{"v1.0.0+build1", "v1.0.0+build2", 0},
	}
	for _, c := range cases {
		a, _ := Parse(c.a)
		b, _ := Parse(c.b)
		if got := a.Compare(b); got != c.want {
			t.Errorf("Compare(%s, %s) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestParseSpec(t *testing.T) {
	tests := []struct {
		in   string
		op   Op
		want string
	}{
		{"v1.2.0", OpExact, "v1.2.0"},
		{"=v1.2.0", OpExact, "v1.2.0"},
		{"^v1.2.0", OpCaret, "v1.2.0"},
		{"~v1.2.0", OpTilde, "v1.2.0"},
		{">=v1.2.0", OpGTE, "v1.2.0"},
		{"latest", OpLatest, ""},
		{"*", OpLatest, ""},
	}
	for _, tt := range tests {
		s, err := ParseSpec(tt.in)
		if err != nil {
			t.Errorf("ParseSpec(%q): %v", tt.in, err)
			continue
		}
		if s.Op != tt.op {
			t.Errorf("ParseSpec(%q) op = %v, want %v", tt.in, s.Op, tt.op)
		}
		if tt.want != "" && s.Version.String() != tt.want {
			t.Errorf("ParseSpec(%q) version = %q, want %q", tt.in, s.Version.String(), tt.want)
		}
	}

	if _, err := ParseSpec("not-a-version"); err == nil {
		t.Errorf("expected error on garbage spec")
	}
}

func TestMatch(t *testing.T) {
	mk := func(s string) Version { v, _ := Parse(s); return v }
	mkSpec := func(s string) Spec { sp, _ := ParseSpec(s); return sp }

	cases := []struct {
		spec string
		v    string
		want bool
	}{
		{"v1.2.3", "v1.2.3", true},
		{"v1.2.3", "v1.2.4", false},
		{"^v1.2.3", "v1.2.3", true},
		{"^v1.2.3", "v1.9.0", true},
		{"^v1.2.3", "v1.2.2", false},
		{"^v1.2.3", "v2.0.0", false},
		{"~v1.2.3", "v1.2.4", true},
		{"~v1.2.3", "v1.3.0", false},
		{">=v1.2.3", "v1.2.3", true},
		{">=v1.2.3", "v2.0.0", true},
		{">=v1.2.3", "v1.2.2", false},
		{"latest", "v0.0.1", true},
	}
	for _, c := range cases {
		if got := Match(mkSpec(c.spec), mk(c.v)); got != c.want {
			t.Errorf("Match(%s, %s) = %v, want %v", c.spec, c.v, got, c.want)
		}
	}
}
