package gitops

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/niradler/git-skill/internal/git"
)

// makeUpstream builds a working repo with one commit on refs/assets/skill/a/b.
func makeUpstream(t *testing.T) (path, commit string) {
	t.Helper()
	work := t.TempDir()
	for _, args := range [][]string{
		{"init", "-q"},
		{"config", "user.email", "t@t"},
		{"config", "user.name", "t"},
	} {
		c := exec.Command("git", args...)
		c.Dir = work
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("%s", out)
		}
	}
	if err := os.WriteFile(filepath.Join(work, "SKILL.md"), []byte("# x"), 0644); err != nil {
		t.Fatal(err)
	}
	wd, _ := os.Getwd()
	os.Chdir(work)
	defer os.Chdir(wd)
	tree, err := git.WriteTreeFromDir(work)
	if err != nil {
		t.Fatal(err)
	}
	commit, err = git.CommitTree(tree, "init")
	if err != nil {
		t.Fatal(err)
	}
	if err := git.UpdateRef("refs/assets/skill/a/b", commit); err != nil {
		t.Fatal(err)
	}
	return work, commit
}

func TestFetchPinnedCommit(t *testing.T) {
	upstream, commit := makeUpstream(t)
	consumer := t.TempDir()
	wd, _ := os.Getwd()
	os.Chdir(consumer)
	defer os.Chdir(wd)
	if _, err := git.Run("init", "-q"); err != nil {
		t.Fatal(err)
	}

	if err := FetchPinnedCommit(upstream, "refs/assets/skill/a/b", commit); err != nil {
		t.Fatalf("FetchPinnedCommit: %v", err)
	}
	if _, err := git.Run("cat-file", "-e", commit); err != nil {
		t.Errorf("commit not present in consumer: %v", err)
	}
}
