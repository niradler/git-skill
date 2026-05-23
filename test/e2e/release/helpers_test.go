package release

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

type repo struct {
	t   *testing.T
	dir string
}

func newRepo(t *testing.T) *repo {
	t.Helper()
	dir := t.TempDir()
	mustRun(t, dir, "git", "init", "-q")
	mustRun(t, dir, "git", "config", "user.email", "t@t")
	mustRun(t, dir, "git", "config", "user.name", "t")
	mustRun(t, dir, "git", "config", "core.autocrlf", "false")
	return &repo{t: t, dir: dir}
}

func newBareRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	mustRun(t, dir, "git", "init", "--bare", "-q")
	return dir
}

func mustRun(t *testing.T, dir, name string, args ...string) {
	t.Helper()
	c := exec.Command(name, args...)
	c.Dir = dir
	if out, err := c.CombinedOutput(); err != nil {
		t.Fatalf("%s %s: %v\n%s", name, strings.Join(args, " "), err, out)
	}
}

// runCLI invokes the shipped binary inside r.dir. Pass the subcommand as args[0].
func (r *repo) runCLI(args ...string) (stdout, stderr string, err error) {
	r.t.Helper()
	c := exec.Command(bin, args...)
	c.Dir = r.dir
	var so, se bytes.Buffer
	c.Stdout = &so
	c.Stderr = &se
	err = c.Run()
	return so.String(), se.String(), err
}

// runCLIAs invokes a copy of the binary named profileName (e.g. "git-agent" or
// "git-asset") so argv[0] dispatch can be exercised. On Windows the binary
// name needs ".exe".
func (r *repo) runCLIAs(profileName string, args ...string) (stdout, stderr string, err error) {
	r.t.Helper()
	exe := profileName
	if runtime.GOOS == "windows" {
		exe = profileName + ".exe"
	}
	copyPath := filepath.Join(r.t.TempDir(), exe)
	if cerr := copyFile(bin, copyPath); cerr != nil {
		return "", "", cerr
	}
	c := exec.Command(copyPath, args...)
	c.Dir = r.dir
	var so, se bytes.Buffer
	c.Stdout = &so
	c.Stderr = &se
	err = c.Run()
	return so.String(), se.String(), err
}

func copyFile(src, dst string) error {
	in, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, in, 0755)
}

func writeFile(t *testing.T, path string, body []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, body, 0644); err != nil {
		t.Fatal(err)
	}
}

func assertFileExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Errorf("expected %s to exist: %v", path, err)
	}
}

func assertFileAbsent(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("expected %s to be absent, got err=%v", path, err)
	}
}

func assertFileContains(t *testing.T, path, want string) {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if !strings.Contains(string(body), want) {
		t.Errorf("%s does not contain %q\n--- got ---\n%s", path, want, body)
	}
}

func fileHash(t *testing.T, path string) string {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

// gitRunLines runs git in the given dir and returns stdout lines, trimming empty.
func gitRunLines(t *testing.T, dir string, args ...string) []string {
	t.Helper()
	c := exec.Command("git", args...)
	c.Dir = dir
	out, err := c.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	var lines []string
	for _, l := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if l != "" {
			lines = append(lines, l)
		}
	}
	return lines
}
