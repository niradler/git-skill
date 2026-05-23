package commands

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestShowPrintsAssetMetadataAndTags(t *testing.T) {
	dir := gitInit(t)
	wd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(wd)
	os.MkdirAll("src", 0755)
	os.WriteFile("src/SKILL.md", []byte("---\nname: x\ndescription: hello\n---\nbody"), 0644)
	if err := Commit(profileSkillOnly, []string{"acme/x", "-m", "feat: init", "--path", "src"}, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if err := Tag(profileSkillOnly, []string{"acme/x", "v1.0.0"}, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if err := Tag(profileSkillOnly, []string{"acme/x", "v1.1.0"}, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if err := Show(profileSkillOnly, []string{"acme/x"}, &stdout, &stderr); err != nil {
		t.Fatalf("Show: %v stderr=%s", err, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "acme/x") {
		t.Errorf("expected name in output:\n%s", out)
	}
	if !strings.Contains(out, "v1.0.0") || !strings.Contains(out, "v1.1.0") {
		t.Errorf("expected tag list in output:\n%s", out)
	}
}
