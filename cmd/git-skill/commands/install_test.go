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
		Runtimes:  map[string]state.RuntimeOverride{"claude": {}},
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
	return makeUpstreamAgentFiles(t, name, map[string]string{"AGENT.md": body})
}

// makeUpstreamAgentFiles seeds an upstream agent repo with arbitrary
// files at the root of the asset tree (e.g. AGENT.md + agent.toml).
func makeUpstreamAgentFiles(t *testing.T, name string, files map[string]string) (string, string) {
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
	for path, body := range files {
		full := filepath.Join("src", path)
		if d := filepath.Dir(full); d != "." {
			os.MkdirAll(d, 0755)
		}
		os.WriteFile(full, []byte(body), 0644)
	}
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
		Runtimes:  map[string]state.RuntimeOverride{"claude": {}},
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

// A6: an agent installed for two runtimes (claude + codex) fans out to
// both destinations using the registry's per-runtime From mapping.
func TestInstallAgentDualRuntimes_ClaudeAndCodex(t *testing.T) {
	mdBody := "---\nname: reviewer\n---\n# reviewer"
	tomlBody := "name = \"reviewer\"\n"
	upstream, commit := makeUpstreamAgentFiles(t, "acme/reviewer", map[string]string{
		"AGENT.md":   mdBody,
		"agent.toml": tomlBody,
	})

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
		Runtimes:  map[string]state.RuntimeOverride{"claude": {}, "codex": {}},
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

	got, err := os.ReadFile(".claude/agents/acme/reviewer.md")
	if err != nil {
		t.Fatalf("claude fan-out missing: %v", err)
	}
	if string(got) != mdBody {
		t.Errorf("claude body mismatch: %q", string(got))
	}
	got, err = os.ReadFile(".codex/agents/acme/reviewer.toml")
	if err != nil {
		t.Fatalf("codex fan-out missing: %v", err)
	}
	if string(got) != tomlBody {
		t.Errorf("codex body mismatch: %q", string(got))
	}
}

// Install respects a per-entry "to" override and writes to the
// overridden path instead of the registry default.
func TestInstall_RespectsToOverride(t *testing.T) {
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
		Runtimes:  map[string]state.RuntimeOverride{"claude": {To: ".custom/claude/acme/x/"}},
		Version:   "v1.0.0",
		Commit:    commit,
		Canonical: "skills/acme/x",
	})
	if err := st.Write(consumer); err != nil {
		t.Fatal(err)
	}

	if err := Install(profileSkillOnly, nil, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatalf("Install: %v", err)
	}
	if _, err := os.Stat(".custom/claude/acme/x/SKILL.md"); err != nil {
		t.Errorf("override path not materialized: %v", err)
	}
	if _, err := os.Stat(".claude/skills/acme/x/SKILL.md"); err == nil {
		t.Errorf("registry default path should not exist when overridden")
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
