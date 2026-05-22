package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/niradler/git-skill/internal/kind"
)

func TestRunHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exit := Run(Profile{Name: "git-skill", DefaultKind: kind.Skill, RequireKind: true}, []string{"--help"}, &stdout, &stderr)
	if exit != 0 {
		t.Errorf("Run(--help) exit = %d, stderr=%s", exit, stderr.String())
	}
	if !strings.Contains(stdout.String(), "git-skill") {
		t.Errorf("help output missing program name:\n%s", stdout.String())
	}
}

func TestRunUnknownCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exit := Run(Profile{Name: "git-skill"}, []string{"bogus"}, &stdout, &stderr)
	if exit == 0 {
		t.Errorf("unknown command should exit non-zero")
	}
}
