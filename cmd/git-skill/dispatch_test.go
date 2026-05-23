package main

import (
	"testing"

	"github.com/niradler/git-skill/internal/kind"
)

func TestProfileFromArgv0(t *testing.T) {
	tests := []struct {
		argv0       string
		wantName    string
		wantKind    kind.Kind
		wantRequire bool
	}{
		{"git-skill", "git-skill", kind.Skill, true},
		{"git-skill.exe", "git-skill", kind.Skill, true},
		{"git-agent", "git-agent", kind.Agent, true},
		{"git-agent.exe", "git-agent", kind.Agent, true},
		{"git-asset", "git-asset", 0, false},
		{"git-asset.exe", "git-asset", 0, false},
		{"/usr/local/bin/git-skill", "git-skill", kind.Skill, true},
		{`C:\path\git-agent.exe`, "git-agent", kind.Agent, true},
	}
	for _, tt := range tests {
		p := ProfileFromArgv0(tt.argv0)
		if p.Name != tt.wantName {
			t.Errorf("argv0=%q Name=%q want %q", tt.argv0, p.Name, tt.wantName)
		}
		if p.DefaultKind != tt.wantKind {
			t.Errorf("argv0=%q DefaultKind=%v want %v", tt.argv0, p.DefaultKind, tt.wantKind)
		}
		if p.RequireKind != tt.wantRequire {
			t.Errorf("argv0=%q RequireKind=%v want %v", tt.argv0, p.RequireKind, tt.wantRequire)
		}
	}
}

func TestProfileFromArgv0Unknown(t *testing.T) {
	p := ProfileFromArgv0("git-weird")
	if p.Name != "git-asset" {
		t.Errorf("unknown argv0 should fall through to git-asset, got %q", p.Name)
	}
}
