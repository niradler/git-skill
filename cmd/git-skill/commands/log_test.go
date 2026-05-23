package commands

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestLogShowsAssetHistory(t *testing.T) {
	dir := gitInit(t)
	wd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(wd)
	os.MkdirAll("src", 0755)
	os.WriteFile("src/SKILL.md", []byte("---\nname: x\n---"), 0644)
	if err := Commit(profileSkillOnly, []string{"acme/x", "-m", "feat: initial", "--path", "src"}, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	os.WriteFile("src/SKILL.md", []byte("---\nname: x\n---\nv2"), 0644)
	if err := Commit(profileSkillOnly, []string{"acme/x", "-m", "feat: second", "--path", "src"}, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if err := Log(profileSkillOnly, []string{"acme/x"}, &stdout, &stderr); err != nil {
		t.Fatalf("Log: %v stderr=%s", err, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "feat: initial") || !strings.Contains(out, "feat: second") {
		t.Errorf("expected both commits in log output:\n%s", out)
	}
}
