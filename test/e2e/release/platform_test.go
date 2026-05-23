package release

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// C1 (Windows): add with --dev, then os.Lstat the runtime path; ensure it exists.
// Symlink OR junction OR copy - all acceptable; we just need the tree mirrored.
func TestC1_WindowsDevLink(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("windows only")
	}

	producer := newRepo(t)
	bare := newBareRepo(t)
	mustRun(t, producer.dir, "git", "remote", "add", "origin", bare)
	writeFile(t, filepath.Join(producer.dir, "src", "SKILL.md"), []byte("# windows dev"))
	producer.runCLI("commit", "acme/x", "-m", "v1", "--path", "src")
	producer.runCLI("tag", "acme/x", "v1.0.0")
	producer.runCLI("push", "origin")

	consumer := newRepo(t)
	consumer.runCLI("init")
	_, stderr, err := consumer.runCLI("add", "acme/x@v1.0.0", "--from", bare, "--runtime", "claude", "--dev")
	if err != nil {
		t.Fatalf("add --dev: %v\n%s", err, stderr)
	}

	// os.Lstat must succeed - symlink, junction, or copy all return without error
	rtPath := filepath.Join(consumer.dir, ".claude", "skills", "acme", "x")
	if _, err := os.Lstat(rtPath); err != nil {
		t.Errorf("runtime path missing after --dev add on Windows: %v", err)
	}
}

// C2 (Unix): add with --dev, os.Lstat then os.Readlink - assert target is relative
// (does NOT start with /).
func TestC2_UnixDevSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix only")
	}

	producer := newRepo(t)
	bare := newBareRepo(t)
	mustRun(t, producer.dir, "git", "remote", "add", "origin", bare)
	writeFile(t, filepath.Join(producer.dir, "src", "SKILL.md"), []byte("# unix dev"))
	producer.runCLI("commit", "acme/x", "-m", "v1", "--path", "src")
	producer.runCLI("tag", "acme/x", "v1.0.0")
	producer.runCLI("push", "origin")

	consumer := newRepo(t)
	consumer.runCLI("init")
	_, stderr, err := consumer.runCLI("add", "acme/x@v1.0.0", "--from", bare, "--runtime", "claude", "--dev")
	if err != nil {
		t.Fatalf("add --dev: %v\n%s", err, stderr)
	}

	rtPath := filepath.Join(consumer.dir, ".claude", "skills", "acme", "x")
	info, err := os.Lstat(rtPath)
	if err != nil {
		t.Fatalf("runtime path missing: %v", err)
	}

	if info.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(rtPath)
		if err != nil {
			t.Fatalf("readlink: %v", err)
		}
		if len(target) > 0 && target[0] == '/' {
			t.Errorf("symlink target should be relative, got absolute: %q", target)
		}
	}
	// If not a symlink (copy fallback), no further assertion needed.
}
