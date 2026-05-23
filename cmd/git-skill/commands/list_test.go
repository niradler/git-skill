package commands

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestListShowsAssetsAndLatestTag(t *testing.T) {
	dir := gitInit(t)
	wd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(wd)

	os.MkdirAll("src", 0755)
	os.WriteFile("src/SKILL.md", []byte("---\nname: x\n---"), 0644)
	if err := Commit(profileSkillOnly, []string{"acme/x", "-m", "feat: init", "--path", "src"}, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	for _, v := range []string{"v1.0.0", "v1.2.0", "v1.1.0"} {
		if err := Tag(profileSkillOnly, []string{"acme/x", v}, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
			t.Fatalf("Tag %s: %v", v, err)
		}
	}
	// asset with no tag
	os.WriteFile("src/SKILL.md", []byte("---\nname: y\n---"), 0644)
	if err := Commit(profileSkillOnly, []string{"acme/y", "-m", "feat: y", "--path", "src"}, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	if err := List(profileSkillOnly, nil, &stdout, &stderr); err != nil {
		t.Fatalf("List: %v stderr=%s", err, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "KIND") || !strings.Contains(out, "NAME") {
		t.Errorf("missing header in output:\n%s", out)
	}
	if !strings.Contains(out, "acme/x") {
		t.Errorf("missing acme/x:\n%s", out)
	}
	if !strings.Contains(out, "v1.2.0") {
		t.Errorf("expected latest tag v1.2.0 (highest of v1.0.0/v1.1.0/v1.2.0):\n%s", out)
	}
	if !strings.Contains(out, "acme/y") {
		t.Errorf("missing acme/y:\n%s", out)
	}
	if !strings.Contains(out, "(none)") {
		t.Errorf("expected (none) for untagged asset:\n%s", out)
	}
}
