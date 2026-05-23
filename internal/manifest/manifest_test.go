package manifest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/niradler/git-skill/internal/runtimes"
)

func writeManifest(t *testing.T, dir, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, Filename), []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestLoad_AbsentReturnsNilNil(t *testing.T) {
	dir := t.TempDir()
	m, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if m != nil {
		t.Errorf("expected nil manifest when file absent, got %+v", m)
	}
}

func TestLoad_HappyPath(t *testing.T) {
	dir := t.TempDir()
	writeManifest(t, dir, `kind: agent
runtimes:
  claude:
    from: prompts/main.md
    to: .claude/agents/<name>.md
  codex:
    from: prompts/main.toml
    to: .codex/agents/<name>.toml
`)
	m, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if m == nil {
		t.Fatal("manifest nil")
	}
	if m.Kind != "agent" {
		t.Errorf("Kind = %q", m.Kind)
	}
	if got := m.Runtimes["claude"].From; got != "prompts/main.md" {
		t.Errorf("claude.From = %q", got)
	}
	if got := m.Runtimes["codex"].To; got != ".codex/agents/<name>.toml" {
		t.Errorf("codex.To = %q", got)
	}
}

func TestLoad_MalformedYAMLError(t *testing.T) {
	dir := t.TempDir()
	writeManifest(t, dir, "runtimes: { not closed\n")
	_, err := Load(dir)
	if err == nil {
		t.Fatal("expected parse error on malformed YAML")
	}
	if !strings.Contains(err.Error(), "parse") && !strings.Contains(err.Error(), Filename) {
		t.Errorf("error %q should mention parse failure or filename", err.Error())
	}
}

func TestLoad_RejectsEmptyRuntimeName(t *testing.T) {
	dir := t.TempDir()
	writeManifest(t, dir, `runtimes:
  "":
    to: .x/<name>/
`)
	_, err := Load(dir)
	if err == nil {
		t.Fatal("expected error on empty runtime name")
	}
}

func TestLoad_RejectsEntryWithNoFields(t *testing.T) {
	dir := t.TempDir()
	writeManifest(t, dir, `runtimes:
  claude: {}
`)
	_, err := Load(dir)
	if err == nil {
		t.Fatal("expected error on empty entry")
	}
	if !strings.Contains(err.Error(), "from") || !strings.Contains(err.Error(), "to") {
		t.Errorf("error %q should mention 'from' or 'to' must be set", err.Error())
	}
}

func TestApply_NilManifestReturnsBase(t *testing.T) {
	base := runtimes.Mapping{From: ".", To: ".claude/skills/acme/x/"}
	got := Apply(base, nil, "claude", "acme/x")
	if got != base {
		t.Errorf("Apply(nil) = %+v, want %+v", got, base)
	}
}

func TestApply_RuntimeAbsentReturnsBase(t *testing.T) {
	m := &Manifest{Runtimes: map[string]Entry{"codex": {From: "agent.toml", To: ".codex/x.toml"}}}
	base := runtimes.Mapping{From: ".", To: ".claude/skills/acme/x/"}
	got := Apply(base, m, "claude", "acme/x")
	if got != base {
		t.Errorf("Apply unchanged when rt absent = %+v, want %+v", got, base)
	}
}

func TestApply_OverridesAndSubstitutesName(t *testing.T) {
	m := &Manifest{Runtimes: map[string]Entry{
		"claude": {From: "prompts/main.md", To: ".claude/agents/<name>.md"},
	}}
	base := runtimes.Mapping{From: ".", To: ".claude/skills/acme/x/"}
	got := Apply(base, m, "claude", "acme/reviewer")
	want := runtimes.Mapping{From: "prompts/main.md", To: ".claude/agents/acme/reviewer.md"}
	if got != want {
		t.Errorf("Apply = %+v, want %+v", got, want)
	}
}

func TestApply_PartialOverrideKeepsBase(t *testing.T) {
	m := &Manifest{Runtimes: map[string]Entry{
		"claude": {To: ".alt/agents/<name>.md"}, // From not set
	}}
	base := runtimes.Mapping{From: "AGENT.md", To: ".claude/agents/acme/x.md"}
	got := Apply(base, m, "claude", "acme/x")
	if got.From != "AGENT.md" {
		t.Errorf("From should be preserved: got %q", got.From)
	}
	if got.To != ".alt/agents/acme/x.md" {
		t.Errorf("To should be overridden+substituted: got %q", got.To)
	}
}

func TestMapping_AbsentReturnsFalse(t *testing.T) {
	if _, ok := (&Manifest{}).Mapping("claude", "x"); ok {
		t.Error("Mapping should return false for absent runtime")
	}
	if _, ok := (*Manifest)(nil).Mapping("claude", "x"); ok {
		t.Error("Mapping should return false on nil manifest")
	}
}

func TestMapping_SubstitutesName(t *testing.T) {
	m := &Manifest{Runtimes: map[string]Entry{
		"future": {From: "src/<name>.txt", To: ".future/<name>/"},
	}}
	got, ok := m.Mapping("future", "acme/foo")
	if !ok {
		t.Fatal("expected ok=true")
	}
	want := runtimes.Mapping{From: "src/acme/foo.txt", To: ".future/acme/foo/"}
	if got != want {
		t.Errorf("Mapping = %+v, want %+v", got, want)
	}
}
