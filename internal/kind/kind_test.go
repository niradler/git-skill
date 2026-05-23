package kind

import (
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		in      string
		want    Kind
		wantErr bool
	}{
		{"skill", Skill, false},
		{"agent", Agent, false},
		{"SKILL", Skill, false},
		{"Agent", Agent, false},
		{"", 0, true},
		{"runtime", 0, true},
		{"skill ", 0, true},
	}
	for _, tt := range tests {
		got, err := Parse(tt.in)
		if (err != nil) != tt.wantErr {
			t.Errorf("Parse(%q) err = %v, wantErr %v", tt.in, err, tt.wantErr)
		}
		if !tt.wantErr && got != tt.want {
			t.Errorf("Parse(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestString(t *testing.T) {
	if Skill.String() != "skill" {
		t.Errorf("Skill.String() = %q", Skill.String())
	}
	if Agent.String() != "agent" {
		t.Errorf("Agent.String() = %q", Agent.String())
	}
}

func TestAll(t *testing.T) {
	all := All()
	if len(all) != 2 || all[0] != Skill || all[1] != Agent {
		t.Errorf("All() = %v, want [Skill, Agent]", all)
	}
}

func TestResolve_Precedence(t *testing.T) {
	tests := []struct {
		name           string
		sources        Sources
		want           Kind
		wantTier       string
		wantWarnSubstr string
	}{
		{
			name:     "lock wins over all",
			sources:  Sources{Lock: Skill, Trailer: Agent, Frontmatter: Agent, Filename: Agent},
			want:     Skill,
			wantTier: "lock",
		},
		{
			name:           "trailer wins when no lock",
			sources:        Sources{Trailer: Agent, Frontmatter: Skill, Filename: Skill},
			want:           Agent,
			wantTier:       "trailer",
			wantWarnSubstr: "trailer disagrees with frontmatter",
		},
		{
			name:     "frontmatter wins when no lock/trailer",
			sources:  Sources{Frontmatter: Skill, Filename: Agent},
			want:     Skill,
			wantTier: "frontmatter",
		},
		{
			name:     "filename last resort",
			sources:  Sources{Filename: Agent},
			want:     Agent,
			wantTier: "filename",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := Resolve(tt.sources)
			if err != nil {
				t.Fatal(err)
			}
			if r.Kind != tt.want {
				t.Errorf("Kind = %v, want %v", r.Kind, tt.want)
			}
			if r.Tier != tt.wantTier {
				t.Errorf("Tier = %q, want %q", r.Tier, tt.wantTier)
			}
			if tt.wantWarnSubstr != "" {
				found := false
				for _, w := range r.Warnings {
					if strings.Contains(w, tt.wantWarnSubstr) {
						found = true
					}
				}
				if !found {
					t.Errorf("expected warning containing %q, got %v", tt.wantWarnSubstr, r.Warnings)
				}
			}
		})
	}
}

func TestResolve_NoSource(t *testing.T) {
	_, err := Resolve(Sources{})
	if err == nil {
		t.Error("expected error when no source resolves")
	}
}
