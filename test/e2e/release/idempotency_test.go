package release

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// E1: double install. After two installs, content hash of canonical SKILL.md and
// runtime SKILL.md are identical pre/post second install.
func TestE1_DoubleInstall(t *testing.T) {
	producer := newRepo(t)
	bare := newBareRepo(t)
	mustRun(t, producer.dir, "git", "remote", "add", "origin", bare)
	writeFile(t, filepath.Join(producer.dir, "src", "SKILL.md"), []byte("# idempotent content"))
	producer.runCLI("commit", "acme/x", "-m", "v1", "--path", "src")
	producer.runCLI("tag", "acme/x", "v1.0.0")
	producer.runCLI("push", "origin")

	consumer := newRepo(t)
	consumer.runCLI("init")
	_, stderr, err := consumer.runCLI("add", "acme/x@v1.0.0", "--from", bare, "--runtime", "claude")
	if err != nil {
		t.Fatalf("add: %v\n%s", err, stderr)
	}

	canonical := filepath.Join(consumer.dir, "skills", "acme", "x", "SKILL.md")
	runtime := filepath.Join(consumer.dir, ".claude", "skills", "acme", "x", "SKILL.md")

	// First install (add already installed; run install explicitly)
	_, stderr, err = consumer.runCLI("install")
	if err != nil {
		t.Fatalf("first install: %v\n%s", err, stderr)
	}
	canonHash1 := fileHash(t, canonical)
	rtHash1 := fileHash(t, runtime)

	// Second install
	_, stderr, err = consumer.runCLI("install")
	if err != nil {
		t.Fatalf("second install: %v\n%s", err, stderr)
	}
	canonHash2 := fileHash(t, canonical)
	rtHash2 := fileHash(t, runtime)

	if canonHash1 != canonHash2 {
		t.Errorf("canonical SKILL.md changed after second install (not idempotent)")
	}
	if rtHash1 != rtHash2 {
		t.Errorf("runtime SKILL.md changed after second install (not idempotent)")
	}
}

// E2: double init. After two inits, .gitignore contains the managed block marker
// exactly once. assets.json mtime must be unchanged after second init.
func TestE2_DoubleInit(t *testing.T) {
	consumer := newRepo(t)

	_, stderr, err := consumer.runCLI("init")
	if err != nil {
		t.Fatalf("first init: %v\n%s", err, stderr)
	}

	assetsPath := filepath.Join(consumer.dir, "assets.json")
	info1, err := os.Stat(assetsPath)
	if err != nil {
		t.Fatalf("stat assets.json after first init: %v", err)
	}
	mtime1 := info1.ModTime()

	// Second init
	_, stderr, err = consumer.runCLI("init")
	if err != nil {
		t.Fatalf("second init: %v\n%s", err, stderr)
	}

	// .gitignore: marker must appear exactly once
	giPath := filepath.Join(consumer.dir, ".gitignore")
	giData, err := os.ReadFile(giPath)
	if err != nil {
		t.Fatalf("read .gitignore: %v", err)
	}
	marker := "# >>> git-skill managed"
	count := strings.Count(string(giData), marker)
	if count != 1 {
		t.Errorf(".gitignore contains managed marker %d times (expected exactly 1):\n%s", count, giData)
	}

	// assets.json mtime must be unchanged
	info2, err := os.Stat(assetsPath)
	if err != nil {
		t.Fatalf("stat assets.json after second init: %v", err)
	}
	if !info2.ModTime().Equal(mtime1) {
		t.Errorf("assets.json mtime changed on second init: before=%v after=%v", mtime1, info2.ModTime())
	}
}
