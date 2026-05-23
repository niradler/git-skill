package git

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const emptyTreeSHA = "4b825dc642cb6eb9a060e54bf8d69288fbee4904"

func Run(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), commandOutput(stdout.String(), stderr.String()))
	}
	return strings.TrimSpace(stdout.String()), nil
}

func commandOutput(stdout, stderr string) string {
	out := strings.TrimSpace(stderr)
	if extra := strings.TrimSpace(stdout); extra != "" {
		if out != "" {
			out += "\n"
		}
		out += extra
	}
	return out
}

func RunLines(args ...string) ([]string, error) {
	out, err := Run(args...)
	if err != nil || out == "" {
		return nil, err
	}
	return strings.Split(out, "\n"), nil
}

func IsRepo() bool {
	_, err := Run("rev-parse", "--is-inside-work-tree")
	return err == nil
}

func TopLevel() (string, error) {
	return Run("rev-parse", "--show-toplevel")
}

func RefExists(ref string) bool {
	_, err := Run("rev-parse", "--verify", "--quiet", ref)
	return err == nil
}

func ResolveRef(ref string) (string, error) {
	return Run("rev-parse", ref)
}

func HashBlob(path string) (string, error) {
	return Run("hash-object", "-w", path)
}

func MkTree(entries string) (string, error) {
	cmd := exec.Command("git", "mktree")
	cmd.Stdin = strings.NewReader(entries)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("mktree: %s", stderr.String())
	}
	return strings.TrimSpace(stdout.String()), nil
}

func WriteTreeFromDir(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("readdir %s: %w", dir, err)
	}
	var lines []string
	for _, e := range entries {
		name := e.Name()
		if name == ".git" {
			continue
		}
		path := filepath.Join(dir, name)
		if e.IsDir() {
			sub, err := WriteTreeFromDir(path)
			if err != nil {
				return "", err
			}
			lines = append(lines, fmt.Sprintf("040000 tree %s\t%s", sub, name))
		} else {
			blob, err := HashBlob(path)
			if err != nil {
				return "", err
			}
			info, err := os.Lstat(path)
			if err != nil {
				return "", fmt.Errorf("lstat %s: %w", path, err)
			}
			mode := "100644"
			if info.Mode()&0111 != 0 {
				mode = "100755"
			}
			lines = append(lines, fmt.Sprintf("%s blob %s\t%s", mode, blob, name))
		}
	}
	if len(lines) == 0 {
		return emptyTreeSHA, nil
	}
	return MkTree(strings.Join(lines, "\n") + "\n")
}

func CommitTree(tree, message string, parents ...string) (string, error) {
	args := []string{"commit-tree", tree}
	for _, p := range parents {
		args = append(args, "-p", p)
	}
	args = append(args, "-m", message)
	return Run(args...)
}

func UpdateRef(ref, commit string) error {
	_, err := Run("update-ref", ref, commit)
	return err
}

// UpdateRefCAS performs a compare-and-set ref update. It tells git to refuse
// the update unless the ref currently points at oldCommit, preventing two
// concurrent writers from silently overwriting each other.
func UpdateRefCAS(ref, newCommit, oldCommit string) error {
	_, err := Run("update-ref", ref, newCommit, oldCommit)
	return err
}

func DeleteRef(ref string) error {
	_, err := Run("update-ref", "-d", ref)
	return err
}

func ForEachRef(pattern, format string) ([]string, error) {
	return RunLines("for-each-ref", "--format="+format, pattern)
}

func Log(ref string, fmtStr string, maxCount int) (string, error) {
	args := []string{"log", "--format=" + fmtStr}
	if maxCount > 0 {
		args = append(args, fmt.Sprintf("-%d", maxCount))
	}
	args = append(args, ref)
	return Run(args...)
}

func ReadTreeToDir(tree, dest string) error {
	tmp, err := os.CreateTemp("", "git-skill-idx-*")
	if err != nil {
		return fmt.Errorf("create temp index: %w", err)
	}
	tmp.Close()
	defer os.Remove(tmp.Name())

	env := append(os.Environ(), "GIT_INDEX_FILE="+tmp.Name())

	cmd := exec.Command("git", "read-tree", tree)
	cmd.Env = env
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("read-tree: %s", out)
	}

	if !strings.HasSuffix(dest, "/") {
		dest += "/"
	}
	cmd = exec.Command("git", "checkout-index", "--all", "--prefix="+dest)
	cmd.Env = env
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("checkout-index: %s", out)
	}
	return nil
}

func DiffTree(a, b string) (string, error) {
	return Run("diff-tree", "-p", "--stat", a, b)
}

func CatFile(typ, hash string) (string, error) {
	return Run("cat-file", typ, hash)
}

// CommitTreeWithMessageFile is like CommitTree but reads the message from a
// tempfile via -F, which (unlike -m) preserves multi-line bodies and trailers
// verbatim. Used by the asset commit path to embed the Asset-Kind trailer.
func CommitTreeWithMessageFile(tree, message string, parents ...string) (string, error) {
	tmp, err := os.CreateTemp("", "git-skill-msg-*")
	if err != nil {
		return "", fmt.Errorf("create message tempfile: %w", err)
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.WriteString(message); err != nil {
		tmp.Close()
		return "", fmt.Errorf("write message tempfile: %w", err)
	}
	tmp.Close()

	args := []string{"commit-tree", tree}
	for _, p := range parents {
		args = append(args, "-p", p)
	}
	args = append(args, "-F", tmp.Name())
	return Run(args...)
}

// runIn shells `git` with cwd=dir. Internal helper for tests that operate on a
// specific repo without chdir-ing.
func runIn(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), commandOutput(stdout.String(), stderr.String()))
	}
	return strings.TrimSpace(stdout.String()), nil
}
