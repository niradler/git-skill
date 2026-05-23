package runtimes

import (
	"strings"
	"testing"

	"github.com/niradler/git-skill/internal/kind"
)

func TestResolve_ClaudeSkill(t *testing.T) {
	got, err := Resolve("claude", kind.Skill, "acme/foo")
	if err != nil {
		t.Fatal(err)
	}
	want := Mapping{From: ".", To: ".claude/skills/acme/foo/"}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
	if !got.IsDir() {
		t.Errorf("expected dir mapping (trailing /)")
	}
}

func TestResolve_ClaudeAgent(t *testing.T) {
	got, err := Resolve("claude", kind.Agent, "nir/code-reviewer")
	if err != nil {
		t.Fatal(err)
	}
	want := Mapping{From: "AGENT.md", To: ".claude/agents/nir/code-reviewer.md"}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
	if got.IsDir() {
		t.Errorf("expected file mapping (no trailing /)")
	}
}

func TestResolve_CodexAgent(t *testing.T) {
	got, err := Resolve("codex", kind.Agent, "nir/code-reviewer")
	if err != nil {
		t.Fatal(err)
	}
	want := Mapping{From: "agent.toml", To: ".codex/agents/nir/code-reviewer.toml"}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestResolve_CodexSkill(t *testing.T) {
	got, err := Resolve("codex", kind.Skill, "acme/foo")
	if err != nil {
		t.Fatal(err)
	}
	want := Mapping{From: ".", To: ".agents/skills/acme/foo/"}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestResolve_UnknownRuntime(t *testing.T) {
	_, err := Resolve("nope", kind.Skill, "foo")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "unknown runtime") {
		t.Errorf("error = %q", err.Error())
	}
}

func TestResolve_UnsupportedKind(t *testing.T) {
	_, err := Resolve("cursor", kind.Agent, "foo")
	if err == nil {
		t.Fatal("expected error for runtime that doesn't support agent kind")
	}
}

func TestKnown(t *testing.T) {
	names := Known()
	if len(names) < 4 {
		t.Errorf("expected at least 4 runtimes, got %v", names)
	}
}

func TestGitignoreLines_DerivesDirPrefixFromTo(t *testing.T) {
	lines := GitignoreLines()
	want := map[string]bool{
		".claude/skills/": true,
		".claude/agents/": true,
		".cursor/rules/":  true,
		".agents/skills/": true,
		".codex/agents/":  true,
	}
	have := map[string]bool{}
	for _, l := range lines {
		have[l] = true
	}
	for w := range want {
		if !have[w] {
			t.Errorf("expected %q in GitignoreLines, got %v", w, lines)
		}
	}
}
