package commands

import (
	"bytes"
	"os"
	"os/exec"
	"testing"
)

func TestPushSendsAssetAndTagRefs(t *testing.T) {
	bare := t.TempDir()
	if err := exec.Command("git", "init", "--bare", bare).Run(); err != nil {
		t.Fatalf("init bare: %v", err)
	}

	work := gitInit(t)
	wd, _ := os.Getwd()
	os.Chdir(work)
	defer os.Chdir(wd)

	if err := exec.Command("git", "remote", "add", "origin", bare).Run(); err != nil {
		t.Fatalf("remote add: %v", err)
	}
	os.MkdirAll("src", 0755)
	os.WriteFile("src/SKILL.md", []byte("---\nname: x\n---"), 0644)
	if err := Commit(profileSkillOnly, []string{"acme/x", "-m", "feat: init", "--path", "src"}, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	if err := Tag(profileSkillOnly, []string{"acme/x", "v1.0.0"}, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatalf("Tag: %v", err)
	}

	var stdout, stderr bytes.Buffer
	if err := Push(profileSkillOnly, nil, &stdout, &stderr); err != nil {
		t.Fatalf("Push: %v stderr=%s", err, stderr.String())
	}

	out, err := exec.Command("git", "--git-dir", bare, "for-each-ref", "--format=%(refname)").Output()
	if err != nil {
		t.Fatalf("for-each-ref: %v", err)
	}
	got := string(out)
	for _, want := range []string{"refs/assets/skill/acme/x", "refs/asset-tags/skill/acme/x/v1.0.0"} {
		if !bytes.Contains([]byte(got), []byte(want)) {
			t.Errorf("missing %s on remote; got:\n%s", want, got)
		}
	}
}
