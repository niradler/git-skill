package commands

import (
	"bytes"
	"os"
	"testing"

	"github.com/niradler/git-skill/internal/git"
)

func TestTagCreatesTagRefAtAssetTip(t *testing.T) {
	dir := gitInit(t)
	wd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(wd)

	os.MkdirAll("src", 0755)
	os.WriteFile("src/SKILL.md", []byte("---\nname: x\n---"), 0644)
	if err := Commit(profileSkillOnly, []string{"acme/x", "-m", "feat: init", "--path", "src"}, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	tip, _ := git.ResolveRef("refs/assets/skill/acme/x")

	var stdout, stderr bytes.Buffer
	if err := Tag(profileSkillOnly, []string{"acme/x", "v1.0.0"}, &stdout, &stderr); err != nil {
		t.Fatalf("Tag: %v stderr=%s", err, stderr.String())
	}

	tag, err := git.ResolveRef("refs/asset-tags/skill/acme/x/v1.0.0")
	if err != nil {
		t.Fatalf("tag ref not found: %v", err)
	}
	if tag != tip {
		t.Errorf("tag points at %s, asset tip is %s", tag, tip)
	}
}

func TestTagRejectsBadSemver(t *testing.T) {
	dir := gitInit(t)
	wd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(wd)
	os.MkdirAll("src", 0755)
	os.WriteFile("src/SKILL.md", []byte("---\nname: x\n---"), 0644)
	Commit(profileSkillOnly, []string{"a/b", "-m", "x", "--path", "src"}, &bytes.Buffer{}, &bytes.Buffer{})

	if err := Tag(profileSkillOnly, []string{"a/b", "not-a-version"}, &bytes.Buffer{}, &bytes.Buffer{}); err == nil {
		t.Errorf("expected error on bad semver")
	}
}
