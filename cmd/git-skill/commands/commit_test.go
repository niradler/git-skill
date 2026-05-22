package commands

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/niradler/git-skill/internal/git"
)

func gitInit(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, args := range [][]string{
		{"init", "-q"},
		{"config", "user.email", "t@t"},
		{"config", "user.name", "t"},
	} {
		c := exec.Command("git", args...)
		c.Dir = dir
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("%s", out)
		}
	}
	return dir
}

func TestCommitWritesTrailerAndRef(t *testing.T) {
	dir := gitInit(t)
	wd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(wd)

	if err := os.MkdirAll("src", 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("src/SKILL.md", []byte("---\nname: x\n---\n# x"), 0644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	err := Commit(profileSkillOnly,
		[]string{"acme/x", "-m", "feat: initial", "--path", "src"},
		&stdout, &stderr)
	if err != nil {
		t.Fatalf("Commit: %v (stderr=%s)", err, stderr.String())
	}

	if !git.RefExists("refs/assets/skill/acme/x") {
		t.Errorf("ref refs/assets/skill/acme/x not created")
	}
	body, err := git.Run("log", "-1", "--format=%B", "refs/assets/skill/acme/x")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(body, "Asset-Kind: skill") {
		t.Errorf("trailer missing in commit message:\n%s", body)
	}
}

func TestCommitWarnsOnKindMismatch(t *testing.T) {
	dir := gitInit(t)
	wd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(wd)

	if err := os.MkdirAll("src", 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("src/SKILL.md",
		[]byte("---\nname: x\nkind: agent\n---\n# x"), 0644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	err := Commit(profileAssetGeneric,
		[]string{"acme/x", "--kind", "skill", "-m", "feat: initial", "--path", "src"},
		&stdout, &stderr)
	if err != nil {
		t.Fatalf("Commit: %v (stderr=%s)", err, stderr.String())
	}
	if !strings.Contains(stderr.String(), "overrides frontmatter") {
		t.Errorf("expected mismatch warning on stderr, got: %s", stderr.String())
	}
}
