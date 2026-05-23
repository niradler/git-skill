package release

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

var bin string

func TestMain(m *testing.M) {
	tmp, err := os.MkdirTemp("", "git-skill-release-bin-*")
	if err != nil {
		fmt.Fprintln(os.Stderr, "tempdir:", err)
		os.Exit(2)
	}
	defer os.RemoveAll(tmp)

	exe := "git-skill"
	if runtime.GOOS == "windows" {
		exe = "git-skill.exe"
	}
	bin = filepath.Join(tmp, exe)

	// Build from the module root. We are inside test/e2e/release; module root is ../../..
	cmd := exec.Command("go", "build", "-o", bin, "./cmd/git-skill")
	cmd.Dir = filepath.Join("..", "..", "..")
	if out, err := cmd.CombinedOutput(); err != nil {
		fmt.Fprintln(os.Stderr, "build failed:", err)
		fmt.Fprintln(os.Stderr, string(out))
		os.Exit(2)
	}
	os.Exit(m.Run())
}
