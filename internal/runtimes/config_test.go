package runtimes

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/niradler/git-skill/internal/kind"
)

func writeConfig(t *testing.T, dir, body string) string {
	t.Helper()
	p := filepath.Join(dir, ConfigFilename)
	if err := os.WriteFile(p, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLoadConfig_AbsentReturnsNilNil(t *testing.T) {
	c, err := LoadConfig(filepath.Join(t.TempDir(), "missing.yaml"))
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if c != nil {
		t.Errorf("expected nil config for missing file, got %+v", c)
	}
}

func TestLoadConfig_HappyPath(t *testing.T) {
	dir := t.TempDir()
	p := writeConfig(t, dir, `runtimes:
  myfuture:
    skill:
      to: .myfuture/skills/<name>/
    agent:
      from: AGENT.md
      to: .myfuture/agents/<name>.md
`)
	c, err := LoadConfig(p)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if c == nil {
		t.Fatal("config nil")
	}
	if got := c.Runtimes["myfuture"]["skill"].To; got != ".myfuture/skills/<name>/" {
		t.Errorf("skill.to = %q", got)
	}
	if got := c.Runtimes["myfuture"]["agent"].From; got != "AGENT.md" {
		t.Errorf("agent.from = %q", got)
	}
}

func TestLoadConfig_RejectsUnknownKind(t *testing.T) {
	dir := t.TempDir()
	p := writeConfig(t, dir, `runtimes:
  myfuture:
    plugin:
      to: .myfuture/<name>/
`)
	_, err := LoadConfig(p)
	if err == nil {
		t.Fatal("expected error on unknown kind")
	}
	if !strings.Contains(err.Error(), "plugin") {
		t.Errorf("error should mention bad kind: %q", err.Error())
	}
}

func TestLoadConfig_RejectsEmptyTo(t *testing.T) {
	dir := t.TempDir()
	p := writeConfig(t, dir, `runtimes:
  myfuture:
    skill:
      from: .
`)
	_, err := LoadConfig(p)
	if err == nil {
		t.Fatal("expected error when 'to' is missing")
	}
	if !strings.Contains(err.Error(), "to") {
		t.Errorf("error %q should mention 'to'", err.Error())
	}
}

func TestLoadConfig_RejectsMalformedYAML(t *testing.T) {
	dir := t.TempDir()
	p := writeConfig(t, dir, "runtimes: { not closed\n")
	_, err := LoadConfig(p)
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestLoadRegistry_NoConfigFilesMatchesBuiltin(t *testing.T) {
	t.Setenv("GIT_SKILL_USER_CONFIG", filepath.Join(t.TempDir(), "missing.yaml"))
	repo := t.TempDir() // no .git-skill/ inside
	r, err := LoadRegistry(repo)
	if err != nil {
		t.Fatalf("LoadRegistry: %v", err)
	}
	got, err := r.Resolve("claude", kind.Skill, "acme/foo")
	if err != nil {
		t.Fatal(err)
	}
	want := Mapping{From: ".", To: ".claude/skills/acme/foo/"}
	if got != want {
		t.Errorf("Resolve = %+v, want %+v", got, want)
	}
}

func TestLoadRegistry_UserConfigAddsNewRuntime(t *testing.T) {
	userDir := t.TempDir()
	userPath := writeConfig(t, userDir, `runtimes:
  myfuture:
    skill:
      to: .myfuture/skills/<name>/
`)
	t.Setenv("GIT_SKILL_USER_CONFIG", userPath)

	r, err := LoadRegistry(t.TempDir())
	if err != nil {
		t.Fatalf("LoadRegistry: %v", err)
	}
	got, err := r.Resolve("myfuture", kind.Skill, "acme/foo")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.To != ".myfuture/skills/acme/foo/" {
		t.Errorf("user-config runtime To = %q", got.To)
	}
}

func TestLoadRegistry_ProjectConfigOverridesUser(t *testing.T) {
	userDir := t.TempDir()
	userPath := writeConfig(t, userDir, `runtimes:
  claude:
    skill:
      to: .user-claude/<name>/
`)
	t.Setenv("GIT_SKILL_USER_CONFIG", userPath)

	repo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repo, ".git-skill"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".git-skill", ConfigFilename), []byte(`runtimes:
  claude:
    skill:
      to: .project-claude/<name>/
`), 0644); err != nil {
		t.Fatal(err)
	}

	r, err := LoadRegistry(repo)
	if err != nil {
		t.Fatalf("LoadRegistry: %v", err)
	}
	got, err := r.Resolve("claude", kind.Skill, "acme/foo")
	if err != nil {
		t.Fatal(err)
	}
	if got.To != ".project-claude/acme/foo/" {
		t.Errorf("project should override user: To = %q", got.To)
	}
}

func TestLoadRegistry_PartialOverridePreservesBuiltin(t *testing.T) {
	// User overrides only 'to'; 'from' should keep the built-in value.
	userDir := t.TempDir()
	userPath := writeConfig(t, userDir, `runtimes:
  claude:
    agent:
      to: .alt/agents/<name>.md
`)
	t.Setenv("GIT_SKILL_USER_CONFIG", userPath)

	r, err := LoadRegistry(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	got, err := r.Resolve("claude", kind.Agent, "nir/x")
	if err != nil {
		t.Fatal(err)
	}
	if got.From != "AGENT.md" {
		t.Errorf("From should be preserved from built-in, got %q", got.From)
	}
	if got.To != ".alt/agents/nir/x.md" {
		t.Errorf("To should be overridden, got %q", got.To)
	}
}

func TestLoadRegistry_MalformedConfigBubblesUp(t *testing.T) {
	userDir := t.TempDir()
	userPath := writeConfig(t, userDir, "runtimes: { not closed\n")
	t.Setenv("GIT_SKILL_USER_CONFIG", userPath)

	_, err := LoadRegistry(t.TempDir())
	if err == nil {
		t.Fatal("expected error from malformed user config")
	}
}
