package runtimes

import (
	"strings"
	"testing"

	"github.com/niradler/git-skill/internal/kind"
)

func TestResolve_Skill(t *testing.T) {
	got, err := Resolve("claude", kind.Skill, "acme/foo")
	if err != nil {
		t.Fatal(err)
	}
	want := ".claude/skills/acme/foo"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestResolve_Agent(t *testing.T) {
	got, err := Resolve("claude", kind.Agent, "nir/code-reviewer")
	if err != nil {
		t.Fatal(err)
	}
	want := ".claude/agents/nir/code-reviewer.md"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
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
