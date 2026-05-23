package state

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/niradler/git-skill/internal/kind"
)

func TestReadMissingReturnsDefaults(t *testing.T) {
	s, err := Read(t.TempDir())
	if err != nil {
		t.Fatalf("Read missing: %v", err)
	}
	if s.Version != SupportedVersion {
		t.Errorf("Version = %d, want %d", s.Version, SupportedVersion)
	}
	if s.Config.SkillsRoot != "skills" {
		t.Errorf("SkillsRoot default = %q", s.Config.SkillsRoot)
	}
	if s.Config.AgentsRoot != "agents" {
		t.Errorf("AgentsRoot default = %q", s.Config.AgentsRoot)
	}
	if len(s.Assets) != 0 {
		t.Errorf("Assets should be empty, got %v", s.Assets)
	}
}

func TestReadWriteRoundTrip(t *testing.T) {
	dir := t.TempDir()
	in := New()
	in.Config = Config{SkillsRoot: "skills", AgentsRoot: "agents"}
	in.Set(kind.Skill, "acme/api-conventions", Entry{
		Spec:   "^v1.2.0",
		Remote: "https://github.com/acme/skills",
		Runtimes: map[string]RuntimeOverride{
			"claude": {},
			"cursor": {To: ".custom/cursor/<name>/"},
		},
		Version:   "v1.2.4",
		Commit:    "abc123def4567890abc123def4567890abc12345",
		Canonical: "skills/acme/api-conventions",
	})
	in.Set(kind.Agent, "nir/reviewer", Entry{
		Spec:      "v0.3.0",
		Remote:    "https://skillhub.example/nir/agents.git",
		Runtimes:  map[string]RuntimeOverride{"claude": {}},
		Requires:  []string{"skill/acme/api-conventions"},
		Version:   "v0.3.0",
		Commit:    "fedcba9876543210fedcba9876543210fedcba98",
		Canonical: "agents/nir/reviewer",
	})

	if err := in.Write(dir); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, Filename)); err != nil {
		t.Fatalf("state not written: %v", err)
	}

	out, err := Read(dir)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if !reflect.DeepEqual(in.Assets, out.Assets) {
		t.Errorf("round-trip mismatch:\n in = %+v\nout = %+v", in.Assets, out.Assets)
	}
	if out.Config.SkillsRoot != "skills" {
		t.Errorf("SkillsRoot lost: %q", out.Config.SkillsRoot)
	}
}

func TestWriteUsesSpecFieldNames(t *testing.T) {
	dir := t.TempDir()
	s := New()
	s.Set(kind.Skill, "acme/x", Entry{
		Spec: "v1.0.0", Remote: "r", Commit: "c", Canonical: "skills/acme/x",
	})
	if err := s.Write(dir); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, Filename))
	if err != nil {
		t.Fatal(err)
	}
	body := string(raw)
	if !strings.Contains(body, `"version": 1`) && !strings.Contains(body, `"version":1`) {
		t.Errorf(`expected on-disk JSON to contain "version": 1, got:\n%s`, body)
	}
	if strings.Contains(body, "stateVersion") {
		t.Errorf(`on-disk JSON contains legacy "stateVersion" field; spec requires "version":\n%s`, body)
	}
}

func TestRejectUnsupportedVersion(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, Filename),
		[]byte(`{"version":99,"assets":{}}`), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := Read(dir); err == nil {
		t.Errorf("expected error on version=99")
	}
}

func TestRejectBadSpec(t *testing.T) {
	dir := t.TempDir()
	body := `{"version":1,"assets":{"skill":{"x":{"spec":"not-a-version","remote":"r","commit":"c","canonical":"skills/x"}}}}`
	if err := os.WriteFile(filepath.Join(dir, Filename), []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := Read(dir); err == nil {
		t.Errorf("expected error on garbage spec")
	}
}

func TestRejectLegacyRuntimesArray(t *testing.T) {
	dir := t.TempDir()
	body := `{"version":1,"assets":{"skill":{"acme/x":{"spec":"v1.0.0","remote":"r","commit":"c","canonical":"skills/acme/x","runtimes":["claude","cursor"]}}}}`
	if err := os.WriteFile(filepath.Join(dir, Filename), []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := Read(dir)
	if err == nil {
		t.Fatal("expected error rejecting []string runtimes form")
	}
	if !strings.Contains(err.Error(), "runtimes") || !strings.Contains(err.Error(), "object") {
		t.Errorf("error %q should explain the new object shape", err.Error())
	}
}

func TestRejectEmptyRuntimeName(t *testing.T) {
	dir := t.TempDir()
	body := `{"version":1,"assets":{"skill":{"acme/x":{"spec":"v1.0.0","remote":"r","commit":"c","canonical":"skills/acme/x","runtimes":{"":{}}}}}}`
	if err := os.WriteFile(filepath.Join(dir, Filename), []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := Read(dir)
	if err == nil {
		t.Fatal("expected error on empty runtime name")
	}
	if !strings.Contains(err.Error(), "must not be empty") {
		t.Errorf("error %q should explain empty name is rejected", err.Error())
	}
}

func TestReadAcceptsNewRuntimesObject(t *testing.T) {
	dir := t.TempDir()
	body := `{"version":1,"assets":{"skill":{"acme/x":{"spec":"v1.0.0","remote":"r","commit":"c","canonical":"skills/acme/x","runtimes":{"claude":{},"codex":{"to":".custom/<name>/"}}}}}}`
	if err := os.WriteFile(filepath.Join(dir, Filename), []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	s, err := Read(dir)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	e, ok := s.Get(kind.Skill, "acme/x")
	if !ok {
		t.Fatal("entry missing")
	}
	if len(e.Runtimes) != 2 {
		t.Fatalf("Runtimes len = %d, want 2: %+v", len(e.Runtimes), e.Runtimes)
	}
	if _, ok := e.Runtimes["claude"]; !ok {
		t.Errorf("claude missing")
	}
	if e.Runtimes["codex"].To != ".custom/<name>/" {
		t.Errorf("codex.to = %q", e.Runtimes["codex"].To)
	}
}

func TestRejectBadRequires(t *testing.T) {
	dir := t.TempDir()
	body := `{"version":1,"assets":{"skill":{"x":{"spec":"v1.0.0","remote":"r","commit":"c","canonical":"skills/x","requires":["not-a-kind/name"]}}}}`
	if err := os.WriteFile(filepath.Join(dir, Filename), []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := Read(dir); err == nil {
		t.Errorf("expected error on requires entry with bad kind prefix")
	}
}

func TestGetSetDelete(t *testing.T) {
	s := New()
	s.Set(kind.Skill, "a/b", Entry{Spec: "v1.0.0", Remote: "r", Commit: "c", Canonical: "skills/a/b"})
	got, ok := s.Get(kind.Skill, "a/b")
	if !ok || got.Remote != "r" {
		t.Fatalf("Get failed: %+v ok=%v", got, ok)
	}
	s.Delete(kind.Skill, "a/b")
	if _, ok := s.Get(kind.Skill, "a/b"); ok {
		t.Errorf("Delete did not remove entry")
	}
}

func TestRoot(t *testing.T) {
	s := New()
	if s.Root(kind.Skill) != "skills" {
		t.Errorf("Root(skill) = %q", s.Root(kind.Skill))
	}
	if s.Root(kind.Agent) != "agents" {
		t.Errorf("Root(agent) = %q", s.Root(kind.Agent))
	}
}
