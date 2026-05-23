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

func commitFile(t *testing.T, repo, body string) string {
	t.Helper()
	if err := os.WriteFile(filepath.Join(repo, "SKILL.md"), []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	wd, _ := os.Getwd()
	os.Chdir(repo)
	defer os.Chdir(wd)
	tree, err := git.WriteTreeFromDir(repo)
	if err != nil {
		t.Fatal(err)
	}
	commit, err := git.CommitTree(tree, "commit")
	if err != nil {
		t.Fatal(err)
	}
	return commit
}

func updateRefIn(t *testing.T, repo, ref, commit string) {
	t.Helper()
	wd, _ := os.Getwd()
	os.Chdir(repo)
	defer os.Chdir(wd)
	if err := git.UpdateRef(ref, commit); err != nil {
		t.Fatal(err)
	}
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

func TestFetchPinnedCommitFallsBackToTagRef(t *testing.T) {
	upstream, oldCommit := makeUpstream(t)
	updateRefIn(t, upstream, "refs/asset-tags/skill/a/b/v1.0.0", oldCommit)
	newCommit := commitFile(t, upstream, "# new")
	updateRefIn(t, upstream, "refs/assets/skill/a/b", newCommit)

	consumer := t.TempDir()
	wd, _ := os.Getwd()
	os.Chdir(consumer)
	defer os.Chdir(wd)
	if _, err := git.Run("init", "-q"); err != nil {
		t.Fatal(err)
	}

	err := FetchPinnedCommit(
		upstream,
		"refs/assets/skill/a/b",
		oldCommit,
		"refs/asset-tags/skill/a/b/v1.0.0",
	)
	if err != nil {
		t.Fatalf("FetchPinnedCommit fallback: %v", err)
	}
	if _, err := git.Run("cat-file", "-e", oldCommit); err != nil {
		t.Errorf("tagged commit not present in consumer: %v", err)
	}
}
