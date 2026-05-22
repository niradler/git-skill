package gitops

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/niradler/git-skill/internal/git"
	"github.com/niradler/git-skill/internal/kind"
)

func newRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, args := range [][]string{
		{"init", "-q"},
		{"config", "user.email", "t@t"},
		{"config", "user.name", "t"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %s", args, out)
		}
	}
	return dir
}

func TestWriteCommitWithKindTrailerAndRead(t *testing.T) {
	dir := newRepo(t)
	wd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(wd)

	if err := os.WriteFile(filepath.Join(dir, "f.txt"), []byte("hi"), 0644); err != nil {
		t.Fatal(err)
	}
	tree, err := git.WriteTreeFromDir(dir)
	if err != nil {
		t.Fatal(err)
	}

	commit, err := WriteCommitWithKind(tree, "feat: add f", kind.Skill, "")
	if err != nil {
		t.Fatalf("WriteCommitWithKind: %v", err)
	}

	k, ok, err := ReadKindTrailer(commit)
	if err != nil {
		t.Fatal(err)
	}
	if !ok || k != kind.Skill {
		t.Errorf("ReadKindTrailer = %v ok=%v, want skill true", k, ok)
	}
}

func TestReadKindTrailerMissing(t *testing.T) {
	dir := newRepo(t)
	wd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(wd)

	os.WriteFile(filepath.Join(dir, "f.txt"), []byte("x"), 0644)
	tree, _ := git.WriteTreeFromDir(dir)
	commit, _ := git.CommitTree(tree, "no trailer")

	_, ok, err := ReadKindTrailer(commit)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Errorf("expected ok=false on missing trailer")
	}
}
