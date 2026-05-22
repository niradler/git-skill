package kind

import "testing"

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
