package lock

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadEmpty(t *testing.T) {
	dir := t.TempDir()
	lk, err := Read(dir)
	if err != nil {
		t.Fatal(err)
	}
	if lk.Version != 2 {
		t.Errorf("expected version 2, got %d", lk.Version)
	}
	if len(lk.Skills) != 0 {
		t.Errorf("expected empty skills, got %d", len(lk.Skills))
	}
}

func TestWriteRead(t *testing.T) {
	dir := t.TempDir()
	lk := &Lock{Version: 2, Skills: map[string]Entry{
		"my-skill": {
			Remote:    "https://example.com/my-skill.git",
			Version:   "v1.0.0",
			Commit:    "abc123",
			Canonical: ".skills/my-skill",
			Agents:    map[string]string{"claude": ".claude/skills/my-skill"},
			Dev:       false,
		},
	}}
	if err := lk.Write(dir); err != nil {
		t.Fatal(err)
	}

	got, err := Read(dir)
	if err != nil {
		t.Fatal(err)
	}
	e := got.Skills["my-skill"]
	if e.Canonical != ".skills/my-skill" {
		t.Errorf("canonical = %q", e.Canonical)
	}
	if e.Agents["claude"] != ".claude/skills/my-skill" {
		t.Errorf("agents.claude = %q", e.Agents["claude"])
	}
}

func TestV1Migration(t *testing.T) {
	dir := t.TempDir()
	v1 := `{
  "lockfileVersion": 1,
  "skills": {
    "old-skill": {
      "remote": "https://example.com/old-skill.git",
      "commit": "def456",
      "dest": ".claude/skills/old-skill"
    }
  }
}`
	if err := os.WriteFile(filepath.Join(dir, Filename), []byte(v1), 0644); err != nil {
		t.Fatal(err)
	}

	lk, err := Read(dir)
	if err != nil {
		t.Fatal(err)
	}
	e := lk.Skills["old-skill"]
	if e.Canonical != ".claude/skills/old-skill" {
		t.Errorf("v1 migration: canonical = %q, want .claude/skills/old-skill", e.Canonical)
	}
	if e.Commit != "def456" {
		t.Errorf("commit = %q", e.Commit)
	}
}
