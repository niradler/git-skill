package commands

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/niradler/git-skill/internal/git"
)

func TestDiscoverListsRemoteAssets(t *testing.T) {
	upstream := gitInit(t)
	wd, _ := os.Getwd()
	os.Chdir(upstream)
	defer os.Chdir(wd)

	os.MkdirAll("src", 0755)
	os.WriteFile("src/SKILL.md", []byte("---\nname: acme/x\n---"), 0644)
	tree, _ := git.WriteTreeFromDir(upstream + "/src")
	c1, _ := git.CommitTree(tree, "x")
	git.UpdateRef("refs/assets/skill/acme/x", c1)
	git.UpdateRef("refs/asset-tags/skill/acme/x/v1.0.0", c1)

	os.WriteFile("src/AGENT.md", []byte("---\nname: foo/bar\n---"), 0644)
	os.Remove("src/SKILL.md")
	tree2, _ := git.WriteTreeFromDir(upstream + "/src")
	c2, _ := git.CommitTree(tree2, "y")
	git.UpdateRef("refs/assets/agent/foo/bar", c2)

	var stdout, stderr bytes.Buffer
	if err := Discover(profileAssetGeneric, []string{upstream}, &stdout, &stderr); err != nil {
		t.Fatalf("Discover: %v stderr=%s", err, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "acme/x") {
		t.Errorf("missing acme/x:\n%s", out)
	}
	if !strings.Contains(out, "foo/bar") {
		t.Errorf("missing foo/bar:\n%s", out)
	}
	if !strings.Contains(out, "skill") || !strings.Contains(out, "agent") {
		t.Errorf("expected both kinds in output:\n%s", out)
	}
}
