package commands

import (
	"os"
	"path/filepath"
	"testing"
)

// TestMain isolates the test process from the developer's real
// ~/.config/git-skill/runtimes.yaml. Without this guard, a test that
// forgets to call t.Setenv("GIT_SKILL_USER_CONFIG", ...) would silently
// merge that file into the registry and pass locally but break on a
// machine without (or with a different) user config. Individual tests
// can still t.Setenv to a real path when they need to exercise the
// user-config layer.
func TestMain(m *testing.M) {
	if os.Getenv("GIT_SKILL_USER_CONFIG") == "" {
		// A path that cannot exist (TempDir is created on demand, never
		// pre-populated). LoadConfig treats missing files as "no config".
		os.Setenv("GIT_SKILL_USER_CONFIG", filepath.Join(os.TempDir(), "git-skill-no-user-config-"+t0(), "runtimes.yaml"))
	}
	os.Exit(m.Run())
}

// t0 returns a per-process suffix so the sentinel path is unique even
// if multiple test binaries run on the same machine.
func t0() string {
	return filepath.Base(os.Args[0])
}
