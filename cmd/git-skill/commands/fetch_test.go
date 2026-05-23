package commands

import (
	"bytes"
	"os"
	"os/exec"
	"testing"

	"github.com/niradler/git-skill/internal/git"
)

func TestFetchPullsAssetAndTagRefs(t *testing.T) {
	// 1. Producer repo — create asset + tag and push to bare remote.
	bare := t.TempDir()
	if err := exec.Command("git", "init", "--bare", bare).Run(); err != nil {
		t.Fatalf("init bare: %v", err)
	}

	producer := gitInit(t)
	wd, _ := os.Getwd()
	os.Chdir(producer)
	if err := exec.Command("git", "remote", "add", "origin", bare).Run(); err != nil {
		t.Fatalf("remote add (producer): %v", err)
	}
	os.MkdirAll("src", 0755)
	os.WriteFile("src/SKILL.md", []byte("---\nname: x\n---"), 0644)
	if err := Commit(profileSkillOnly, []string{"acme/x", "-m", "feat: init", "--path", "src"}, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	if err := Tag(profileSkillOnly, []string{"acme/x", "v1.0.0"}, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatalf("Tag: %v", err)
	}
	if err := Push(profileSkillOnly, nil, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatalf("Push: %v", err)
	}

	// 2. Consumer repo — separate clone-like repo, fetch from same bare.
	consumer := gitInit(t)
	os.Chdir(consumer)
	defer os.Chdir(wd)
	if err := exec.Command("git", "remote", "add", "origin", bare).Run(); err != nil {
		t.Fatalf("remote add (consumer): %v", err)
	}

	var stdout, stderr bytes.Buffer
	if err := Fetch(profileSkillOnly, nil, &stdout, &stderr); err != nil {
		t.Fatalf("Fetch: %v stderr=%s", err, stderr.String())
	}

	for _, want := range []string{"refs/assets/skill/acme/x", "refs/asset-tags/skill/acme/x/v1.0.0"} {
		if _, err := git.ResolveRef(want); err != nil {
			t.Errorf("expected local ref %s after fetch, got: %v", want, err)
		}
	}
}
