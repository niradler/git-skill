package commands

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestDiffShowsChangesBetweenTags(t *testing.T) {
	dir := gitInit(t)
	wd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(wd)
	os.MkdirAll("src", 0755)
	os.WriteFile("src/SKILL.md", []byte("---\nname: x\n---\nv1"), 0644)
	if err := Commit(profileSkillOnly, []string{"acme/x", "-m", "feat: v1", "--path", "src"}, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if err := Tag(profileSkillOnly, []string{"acme/x", "v1.0.0"}, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	os.WriteFile("src/SKILL.md", []byte("---\nname: x\n---\nv2 body"), 0644)
	if err := Commit(profileSkillOnly, []string{"acme/x", "-m", "feat: v2", "--path", "src"}, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if err := Tag(profileSkillOnly, []string{"acme/x", "v2.0.0"}, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if err := Diff(profileSkillOnly, []string{"acme/x", "v1.0.0", "v2.0.0"}, &stdout, &stderr); err != nil {
		t.Fatalf("Diff: %v stderr=%s", err, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "v2 body") && !strings.Contains(out, "SKILL.md") {
		t.Errorf("expected diff output to mention SKILL.md or new content:\n%s", out)
	}
}
