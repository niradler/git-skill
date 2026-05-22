package commands

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/niradler/git-skill/internal/kind"
)

var profileSkillOnly = Profile{Name: "git-skill", DefaultKind: kind.Skill, RequireKind: true}
var profileAgentOnly = Profile{Name: "git-agent", DefaultKind: kind.Agent, RequireKind: true}
var profileAssetGeneric = Profile{Name: "git-asset", RequireKind: false}

func TestInitCreatesStateAndGitignore(t *testing.T) {
	dir := t.TempDir()
	wd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(wd)

	var stdout, stderr bytes.Buffer
	if err := Init(profileSkillOnly, nil, &stdout, &stderr); err != nil {
		t.Fatalf("Init: %v (stderr=%s)", err, stderr.String())
	}

	if _, err := os.Stat("assets.json"); err != nil {
		t.Errorf("assets.json not created: %v", err)
	}
	gi, err := os.ReadFile(".gitignore")
	if err != nil {
		t.Fatalf("read .gitignore: %v", err)
	}
	if !strings.Contains(string(gi), gitignoreBlockMarker) {
		t.Errorf(".gitignore missing block marker:\n%s", gi)
	}
}

func TestInitIdempotent(t *testing.T) {
	dir := t.TempDir()
	wd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(wd)

	var b1, b2 bytes.Buffer
	if err := Init(profileSkillOnly, nil, &b1, &b1); err != nil {
		t.Fatal(err)
	}
	gi1, _ := os.ReadFile(".gitignore")

	if err := Init(profileSkillOnly, nil, &b2, &b2); err != nil {
		t.Fatal(err)
	}
	gi2, _ := os.ReadFile(".gitignore")

	if string(gi1) != string(gi2) {
		t.Errorf(".gitignore changed on second Init")
	}
}
