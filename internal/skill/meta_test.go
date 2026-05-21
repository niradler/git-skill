package skill

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeSKILL(t *testing.T, dir, content string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestParseMeta_Full(t *testing.T) {
	dir := t.TempDir()
	writeSKILL(t, dir, "---\nname: test-skill\ndescription: A test skill\nversion: 1.2.3\nlicense: MIT\n---\n\n# Test\n")

	m, err := ParseMeta(dir)
	if err != nil {
		t.Fatal(err)
	}
	if m.Name != "test-skill" {
		t.Errorf("Name = %q, want test-skill", m.Name)
	}
	if m.Description != "A test skill" {
		t.Errorf("Description = %q, want 'A test skill'", m.Description)
	}
	if m.Version != "1.2.3" {
		t.Errorf("Version = %q, want 1.2.3", m.Version)
	}
	if m.License != "MIT" {
		t.Errorf("License = %q, want MIT", m.License)
	}
}

func TestParseMeta_NoFrontmatter(t *testing.T) {
	dir := t.TempDir()
	writeSKILL(t, dir, "# Just a heading\n\nNo frontmatter here.\n")

	m, err := ParseMeta(dir)
	if err != nil {
		t.Fatal(err)
	}
	if m.Name != "" || m.Version != "" || m.Description != "" || m.License != "" {
		t.Errorf("expected empty meta for no-frontmatter file, got %+v", m)
	}
}

func TestParseMeta_MissingFile(t *testing.T) {
	dir := t.TempDir()
	_, err := ParseMeta(dir)
	if err == nil {
		t.Fatal("expected error for missing SKILL.md, got nil")
	}
}

func TestParseMeta_PartialFrontmatter(t *testing.T) {
	dir := t.TempDir()
	writeSKILL(t, dir, "---\nname: partial-skill\n---\n")

	m, err := ParseMeta(dir)
	if err != nil {
		t.Fatal(err)
	}
	if m.Name != "partial-skill" {
		t.Errorf("Name = %q, want partial-skill", m.Name)
	}
	if m.Version != "" {
		t.Errorf("Version should be empty, got %q", m.Version)
	}
}

func TestParseMeta_IgnoresBodyAfterFrontmatter(t *testing.T) {
	dir := t.TempDir()
	// "name: body-name" appears after closing --- and must be ignored
	writeSKILL(t, dir, "---\nname: header-name\n---\n\nname: body-name\n")

	m, err := ParseMeta(dir)
	if err != nil {
		t.Fatal(err)
	}
	if m.Name != "header-name" {
		t.Errorf("Name = %q, want header-name (body should be ignored)", m.Name)
	}
}

func TestFormatCommitMessage_SubjectOnly(t *testing.T) {
	got := FormatCommitMessage("Fix typo", "", nil)
	if got != "Fix typo" {
		t.Errorf("got %q, want 'Fix typo'", got)
	}
}

func TestFormatCommitMessage_WithBody(t *testing.T) {
	got := FormatCommitMessage("Fix typo", "More details here.", nil)
	want := "Fix typo\n\nMore details here."
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFormatCommitMessage_WithVersion(t *testing.T) {
	m := &Meta{Version: "1.0.0"}
	got := FormatCommitMessage("Release", "", m)
	if !strings.Contains(got, "Skill-Version: 1.0.0") {
		t.Errorf("missing Skill-Version trailer in %q", got)
	}
	if !strings.HasPrefix(got, "Release") {
		t.Errorf("subject missing from %q", got)
	}
}

func TestFormatCommitMessage_WithAll(t *testing.T) {
	m := &Meta{Version: "2.0.0"}
	got := FormatCommitMessage("Release", "Major update.", m)
	if !strings.Contains(got, "Release") {
		t.Errorf("missing subject in %q", got)
	}
	if !strings.Contains(got, "Major update.") {
		t.Errorf("missing body in %q", got)
	}
	if !strings.Contains(got, "Skill-Version: 2.0.0") {
		t.Errorf("missing Skill-Version trailer in %q", got)
	}
}

func TestFormatCommitMessage_NilMetaNoTrailer(t *testing.T) {
	got := FormatCommitMessage("msg", "", nil)
	if strings.Contains(got, "Skill-Version") {
		t.Errorf("nil meta should not emit Skill-Version, got %q", got)
	}
}

func TestFormatCommitMessage_EmptyVersionNoTrailer(t *testing.T) {
	m := &Meta{Version: ""}
	got := FormatCommitMessage("msg", "", m)
	if strings.Contains(got, "Skill-Version") {
		t.Errorf("empty version should not emit Skill-Version, got %q", got)
	}
}

func TestScaffold_CreatesValidSKILL(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "my-skill")

	if err := Scaffold(skillDir, "my-skill", "Does things"); err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(filepath.Join(skillDir, "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(content)
	if !strings.Contains(s, "name: my-skill") {
		t.Errorf("missing name in scaffold output:\n%s", s)
	}
	if !strings.Contains(s, "description: Does things") {
		t.Errorf("missing description in scaffold output:\n%s", s)
	}
	if !strings.Contains(s, "version: 0.1.0") {
		t.Errorf("missing default version in scaffold output:\n%s", s)
	}
}

func TestScaffold_OutputParseable(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "parse-me")

	if err := Scaffold(skillDir, "parse-me", "A skill"); err != nil {
		t.Fatal(err)
	}

	m, err := ParseMeta(skillDir)
	if err != nil {
		t.Fatalf("ParseMeta after Scaffold: %v", err)
	}
	if m.Name != "parse-me" {
		t.Errorf("ParseMeta.Name = %q, want parse-me", m.Name)
	}
	if m.Version != "0.1.0" {
		t.Errorf("ParseMeta.Version = %q, want 0.1.0", m.Version)
	}
}

func TestScaffold_CreatesNestedDirs(t *testing.T) {
	dir := t.TempDir()
	nested := filepath.Join(dir, "a", "b", "c")
	if err := Scaffold(nested, "x", "y"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(nested, "SKILL.md")); err != nil {
		t.Errorf("SKILL.md not created in nested dir: %v", err)
	}
}

func TestScaffold_ExistingDirNoError(t *testing.T) {
	dir := t.TempDir()
	// Directory already exists; Scaffold should still succeed.
	if err := Scaffold(dir, "existing", "desc"); err != nil {
		t.Errorf("Scaffold into existing dir failed: %v", err)
	}
}
