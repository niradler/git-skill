package assetignore

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMatch_Patterns(t *testing.T) {
	m, err := ParseString(`
# comment
eval/
tests/
*.draft
!important.draft
scripts/*.sh
`)
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		path string
		dir  bool
		want bool
	}{
		{"eval/foo.txt", false, true},
		{"eval/", true, true},
		{"eval", false, false},
		{"tests/", true, true},
		{"src/main.go", false, false},
		{"thoughts.draft", false, true},
		{"important.draft", false, false},
		{"scripts/run.sh", false, true},
		{"scripts/lib/run.sh", false, false},
		{"SKILL.md", false, false},
	}
	for _, c := range cases {
		if got := m.MatchPath(c.path, c.dir); got != c.want {
			t.Errorf("MatchPath(%q, dir=%v) = %v, want %v", c.path, c.dir, got, c.want)
		}
	}
}

func TestParseFile_RepoRoot(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".assetignore"),
		[]byte("eval/\n"), 0644); err != nil {
		t.Fatal(err)
	}
	m, err := ParseFile(filepath.Join(dir, ".assetignore"))
	if err != nil {
		t.Fatal(err)
	}
	if !m.MatchPath("eval/x", false) {
		t.Error("expected eval/x to match")
	}
}

func TestDiscover_PerAssetOverridesRoot(t *testing.T) {
	repo := t.TempDir()
	if err := os.WriteFile(filepath.Join(repo, ".assetignore"),
		[]byte("eval/\n"), 0644); err != nil {
		t.Fatal(err)
	}
	asset := filepath.Join(repo, "skills", "foo")
	if err := os.MkdirAll(asset, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(asset, ".assetignore"),
		[]byte("docs/\n"), 0644); err != nil {
		t.Fatal(err)
	}
	m, err := Discover(repo, asset)
	if err != nil {
		t.Fatal(err)
	}
	if m.MatchPath("eval/x", false) {
		t.Error("per-asset matcher should not inherit eval/ from root")
	}
	if !m.MatchPath("docs/x", false) {
		t.Error("per-asset docs/ should match")
	}
}

func TestDiscover_RootOnly(t *testing.T) {
	repo := t.TempDir()
	if err := os.WriteFile(filepath.Join(repo, ".assetignore"),
		[]byte("eval/\n"), 0644); err != nil {
		t.Fatal(err)
	}
	asset := filepath.Join(repo, "skills", "foo")
	if err := os.MkdirAll(asset, 0755); err != nil {
		t.Fatal(err)
	}
	m, err := Discover(repo, asset)
	if err != nil {
		t.Fatal(err)
	}
	if !m.MatchPath("eval/x", false) {
		t.Error("root matcher should apply when no per-asset file")
	}
}
