package commands

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/niradler/git-skill/internal/git"
	"github.com/niradler/git-skill/internal/kind"
	"github.com/niradler/git-skill/internal/state"
)

func makeUpstreamSkill(t *testing.T, name string) (string, string) {
	t.Helper()
	dir := t.TempDir()
	for _, args := range [][]string{
		{"init", "-q"},
		{"config", "user.email", "t@t"},
		{"config", "user.name", "t"},
		{"config", "core.autocrlf", "false"},
	} {
		c := exec.Command("git", args...)
		c.Dir = dir
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("%s", out)
		}
	}
	wd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(wd)
	os.MkdirAll("src", 0755)
	os.WriteFile("src/SKILL.md", []byte("---\nname: "+name+"\n---\n# "+name), 0644)
	os.WriteFile("src/extra.txt", []byte("payload"), 0644)
	tree, _ := git.WriteTreeFromDir(filepath.Join(dir, "src"))
	commit, _ := git.CommitTree(tree, "init")
	_ = git.UpdateRef("refs/assets/skill/"+name, commit)
	return dir, commit
}

func TestInstallFromState(t *testing.T) {
	upstream, commit := makeUpstreamSkill(t, "acme/x")

	consumer := t.TempDir()
	wd, _ := os.Getwd()
	os.Chdir(consumer)
	defer os.Chdir(wd)
	exec.Command("git", "init", "-q").Run()
	exec.Command("git", "config", "core.autocrlf", "false").Run()

	st := state.New()
	st.Set(kind.Skill, "acme/x", state.Entry{
		Spec:      "v1.0.0",
		Remote:    upstream,
		Runtimes:  []string{"claude"},
		Version:   "v1.0.0",
		Commit:    commit,
		Canonical: "skills/acme/x",
	})
	if err := st.Write(consumer); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	if err := Install(profileSkillOnly, nil, &stdout, &stderr); err != nil {
		t.Fatalf("Install: %v stderr=%s", err, stderr.String())
	}
	if _, err := os.Stat("skills/acme/x/SKILL.md"); err != nil {
		t.Errorf("canonical SKILL.md missing: %v", err)
	}
	if _, err := os.Stat(".claude/skills/acme/x/SKILL.md"); err != nil {
		t.Errorf("runtime fan-out missing: %v", err)
	}
}

func makeUpstreamAgent(t *testing.T, name, body string) (string, string) {
	t.Helper()
	dir := t.TempDir()
	for _, args := range [][]string{
		{"init", "-q"},
		{"config", "user.email", "t@t"},
		{"config", "user.name", "t"},
		{"config", "core.autocrlf", "false"},
	} {
		c := exec.Command("git", args...)
		c.Dir = dir
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("%s", out)
		}
	}
	wd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(wd)
	os.MkdirAll("src", 0755)
	os.WriteFile("src/AGENT.md", []byte(body), 0644)
	tree, _ := git.WriteTreeFromDir(filepath.Join(dir, "src"))
	commit, _ := git.CommitTree(tree, "init")
	_ = git.UpdateRef("refs/assets/agent/"+name, commit)
	return dir, commit
}

func TestInstallAgentFansOutAsFile(t *testing.T) {
	body := "---\nname: reviewer\n---\n# reviewer agent body"
	upstream, commit := makeUpstreamAgent(t, "acme/reviewer", body)

	consumer := t.TempDir()
	wd, _ := os.Getwd()
	os.Chdir(consumer)
	defer os.Chdir(wd)
	exec.Command("git", "init", "-q").Run()
	exec.Command("git", "config", "core.autocrlf", "false").Run()

	st := state.New()
	st.Set(kind.Agent, "acme/reviewer", state.Entry{
		Spec:      "v1.0.0",
		Remote:    upstream,
		Runtimes:  []string{"claude"},
		Version:   "v1.0.0",
		Commit:    commit,
		Canonical: "agents/acme/reviewer",
	})
	if err := st.Write(consumer); err != nil {
		t.Fatal(err)
	}

	if err := Install(profileAgentOnly, nil, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatalf("Install: %v", err)
	}

	if info, err := os.Stat("agents/acme/reviewer/AGENT.md"); err != nil || info.IsDir() {
		t.Errorf("canonical AGENT.md missing or not a file: %v", err)
	}

	rtPath := ".claude/agents/acme/reviewer.md"
	info, err := os.Stat(rtPath)
	if err != nil {
		t.Fatalf("runtime agent file missing at %s: %v", rtPath, err)
	}
	if info.IsDir() {
		t.Errorf("%s is a directory; agents must materialize as a single .md file", rtPath)
	}
	got, _ := os.ReadFile(rtPath)
	if string(got) != body {
		t.Errorf("runtime agent body mismatch:\n got=%q\nwant=%q", string(got), body)
	}
}

func TestInstallRejectsMissingCommit(t *testing.T) {
	consumer := t.TempDir()
	wd, _ := os.Getwd()
	os.Chdir(consumer)
	defer os.Chdir(wd)
	exec.Command("git", "init", "-q").Run()

	st := state.New()
	st.Set(kind.Skill, "acme/x", state.Entry{
		Spec:      "v1.0.0",
		Remote:    "https://example.invalid/repo.git",
		Canonical: "skills/acme/x",
	})
	if err := st.Write(consumer); err != nil {
		t.Fatal(err)
	}

	err := Install(profileSkillOnly, nil, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil {
		t.Errorf("expected error on entry with empty commit")
	}
}
