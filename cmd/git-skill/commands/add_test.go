package commands

import (
	"bytes"
	"os"
	"os/exec"
	"testing"

	"github.com/niradler/git-skill/internal/git"
	"github.com/niradler/git-skill/internal/kind"
	"github.com/niradler/git-skill/internal/state"
)

func TestAddNewSkill(t *testing.T) {
	upstream, _ := makeUpstreamSkill(t, "acme/x")
	wd, _ := os.Getwd()
	os.Chdir(upstream)
	tip, _ := git.ResolveRef("refs/assets/skill/acme/x")
	git.UpdateRef("refs/asset-tags/skill/acme/x/v1.0.0", tip)
	os.Chdir(wd)

	consumer := t.TempDir()
	os.Chdir(consumer)
	defer os.Chdir(wd)
	exec.Command("git", "init", "-q").Run()
	exec.Command("git", "config", "core.autocrlf", "false").Run()
	Init(profileSkillOnly, nil, &bytes.Buffer{}, &bytes.Buffer{})

	var stdout, stderr bytes.Buffer
	err := Add(profileSkillOnly,
		[]string{"acme/x@v1.0.0", "--from", upstream, "--runtime", "claude"},
		&stdout, &stderr)
	if err != nil {
		t.Fatalf("Add: %v stderr=%s", err, stderr.String())
	}

	st, _ := state.Read(consumer)
	e, ok := st.Get(kind.Skill, "acme/x")
	if !ok {
		t.Fatalf("state entry missing")
	}
	if e.Spec == "" {
		t.Errorf("Spec not stored: %+v", e)
	}
	if e.Commit == "" {
		t.Errorf("Commit not resolved: %+v", e)
	}
	if e.Version != "v1.0.0" {
		t.Errorf("Version = %q, want v1.0.0", e.Version)
	}
	if _, err := os.Stat("skills/acme/x/SKILL.md"); err != nil {
		t.Errorf("canonical materialization missing: %v", err)
	}
}
