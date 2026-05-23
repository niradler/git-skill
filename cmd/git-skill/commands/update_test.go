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

func TestUpdateBumpsCommitWhenNewerTagPublished(t *testing.T) {
	upstream, _ := makeUpstreamSkill(t, "acme/x")
	wd, _ := os.Getwd()
	os.Chdir(upstream)
	tip1, _ := git.ResolveRef("refs/assets/skill/acme/x")
	git.UpdateRef("refs/asset-tags/skill/acme/x/v1.0.0", tip1)
	os.Chdir(wd)

	consumer := t.TempDir()
	os.Chdir(consumer)
	defer os.Chdir(wd)
	exec.Command("git", "init", "-q").Run()
	exec.Command("git", "config", "core.autocrlf", "false").Run()
	Init(profileSkillOnly, nil, &bytes.Buffer{}, &bytes.Buffer{})

	if err := Add(profileSkillOnly,
		[]string{"acme/x@^v1.0.0", "--from", upstream, "--runtime", "claude"},
		&bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}

	os.Chdir(upstream)
	os.WriteFile("src/SKILL.md", []byte("---\nname: acme/x\n---\n# v1.1.0"), 0644)
	tree, _ := git.WriteTreeFromDir(upstream + "/src")
	tip2, _ := git.CommitTree(tree, "v1.1.0")
	_ = git.UpdateRef("refs/assets/skill/acme/x", tip2)
	_ = git.UpdateRef("refs/asset-tags/skill/acme/x/v1.1.0", tip2)
	os.Chdir(consumer)

	st0, _ := state.Read(consumer)
	c0, _ := st0.Get(kind.Skill, "acme/x")
	if c0.Version != "v1.0.0" {
		t.Fatalf("pre-update version = %q", c0.Version)
	}

	if err := Update(profileSkillOnly, nil, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	st1, _ := state.Read(consumer)
	c1, _ := st1.Get(kind.Skill, "acme/x")
	if c1.Version != "v1.1.0" {
		t.Errorf("Version = %q, want v1.1.0", c1.Version)
	}
	if c1.Commit == c0.Commit {
		t.Errorf("Commit not updated: still %s", c1.Commit)
	}
}
