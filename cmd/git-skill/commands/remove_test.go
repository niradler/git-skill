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

func TestRemoveDeletesEntryAndMaterializedPaths(t *testing.T) {
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
	Add(profileSkillOnly, []string{"acme/x@v1.0.0", "--from", upstream, "--runtime", "claude"}, &bytes.Buffer{}, &bytes.Buffer{})

	if err := Remove(profileSkillOnly, []string{"acme/x"}, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat("skills/acme/x"); !os.IsNotExist(err) {
		t.Errorf("canonical path not removed: %v", err)
	}
	if _, err := os.Stat(".claude/skills/acme/x"); !os.IsNotExist(err) {
		t.Errorf("runtime path not removed: %v", err)
	}
	st, _ := state.Read(consumer)
	if _, ok := st.Get(kind.Skill, "acme/x"); ok {
		t.Errorf("state entry still present")
	}
}
