package runtimes

import (
	"os"
	"path/filepath"
	"testing"
)

// TestMain isolates this package's tests from the developer's real
// ~/.config/git-skill/runtimes.yaml. See cmd/git-skill/commands/main_test.go
// for the rationale.
func TestMain(m *testing.M) {
	if os.Getenv("GIT_SKILL_USER_CONFIG") == "" {
		os.Setenv("GIT_SKILL_USER_CONFIG", filepath.Join(os.TempDir(), "git-skill-no-user-config-"+filepath.Base(os.Args[0]), "runtimes.yaml"))
	}
	os.Exit(m.Run())
}
