package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// setupRepo creates a temporary git repository and changes the process working
// directory to it for the duration of the test. All git package functions rely
// on the current working directory being inside a repo.
func setupRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	must := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %s", args, out)
		}
	}
	must("init", dir)
	must("-C", dir, "config", "user.email", "test@example.com")
	must("-C", dir, "config", "user.name", "Test User")

	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(orig) }) //nolint:errcheck

	return dir
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

// ── Run / RunLines ────────────────────────────────────────────────────────────

func TestRun_Success(t *testing.T) {
	setupRepo(t)
	out, err := Run("rev-parse", "--is-inside-work-tree")
	if err != nil {
		t.Fatal(err)
	}
	if out != "true" {
		t.Errorf("got %q, want true", out)
	}
}

func TestRun_Error(t *testing.T) {
	setupRepo(t)
	_, err := Run("cat-file", "-t", "nosuchobject123")
	if err == nil {
		t.Fatal("expected error for unknown object")
	}
}

func TestRunLines_MultipleLines(t *testing.T) {
	setupRepo(t)
	lines, err := RunLines("config", "--list")
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) == 0 {
		t.Error("expected at least one config line")
	}
}

func TestRunLines_EmptyOutput(t *testing.T) {
	setupRepo(t)
	// for-each-ref on an empty repo returns no output
	lines, err := RunLines("for-each-ref", "--format=%(refname)", "refs/skills/")
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 0 {
		t.Errorf("expected nil/empty slice, got %v", lines)
	}
}

// ── IsRepo / TopLevel ─────────────────────────────────────────────────────────

func TestIsRepo_Inside(t *testing.T) {
	setupRepo(t)
	if !IsRepo() {
		t.Error("IsRepo() = false inside a git repo")
	}
}

func TestTopLevel(t *testing.T) {
	dir := setupRepo(t)
	top, err := TopLevel()
	if err != nil {
		t.Fatal(err)
	}
	// EvalSymlinks because macOS TempDir returns /var/… which is a symlink to /private/var/…
	got, _ := filepath.EvalSymlinks(top)
	want, _ := filepath.EvalSymlinks(dir)
	if got != want {
		t.Errorf("TopLevel() = %q, want %q", got, want)
	}
}

// ── HashBlob ──────────────────────────────────────────────────────────────────

func TestHashBlob(t *testing.T) {
	dir := setupRepo(t)
	f := filepath.Join(dir, "test.txt")
	writeFile(t, f, "hello git")

	sha, err := HashBlob(f)
	if err != nil {
		t.Fatal(err)
	}
	if len(sha) != 40 {
		t.Errorf("blob SHA should be 40 hex chars, got %d: %q", len(sha), sha)
	}

	// Hashing the same content twice must yield the same SHA.
	f2 := filepath.Join(dir, "test2.txt")
	writeFile(t, f2, "hello git")
	sha2, err := HashBlob(f2)
	if err != nil {
		t.Fatal(err)
	}
	if sha != sha2 {
		t.Errorf("identical content produced different SHAs: %q vs %q", sha, sha2)
	}
}

// ── WriteTreeFromDir ──────────────────────────────────────────────────────────

func TestWriteTreeFromDir(t *testing.T) {
	dir := setupRepo(t)
	skillDir := filepath.Join(dir, ".skills", "test")
	writeFile(t, filepath.Join(skillDir, "SKILL.md"), "---\nname: test\n---\n")

	sha, err := WriteTreeFromDir(skillDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(sha) != 40 {
		t.Errorf("tree SHA should be 40 chars, got %q", sha)
	}

	// Must be a tree object.
	typ, err := CatFile("-t", sha)
	if err != nil {
		t.Fatal(err)
	}
	if typ != "tree" {
		t.Errorf("object type = %q, want tree", typ)
	}
}

func TestWriteTreeFromDir_Empty(t *testing.T) {
	dir := setupRepo(t)
	emptyDir := filepath.Join(dir, "empty")
	if err := os.MkdirAll(emptyDir, 0755); err != nil {
		t.Fatal(err)
	}

	sha, err := WriteTreeFromDir(emptyDir)
	if err != nil {
		t.Fatal(err)
	}
	if sha != emptyTreeSHA {
		t.Errorf("empty dir returned %q, want emptyTreeSHA %q", sha, emptyTreeSHA)
	}
}

func TestWriteTreeFromDir_Nested(t *testing.T) {
	dir := setupRepo(t)
	skillDir := filepath.Join(dir, "nested-skill")
	writeFile(t, filepath.Join(skillDir, "SKILL.md"), "root file")
	writeFile(t, filepath.Join(skillDir, "references", "guide.md"), "guide content")

	sha, err := WriteTreeFromDir(skillDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(sha) != 40 {
		t.Errorf("nested tree SHA should be 40 chars, got %q", sha)
	}
}

func TestWriteTreeFromDir_SkipsGitDir(t *testing.T) {
	dir := setupRepo(t)
	// A .git directory inside the skill dir should be skipped silently.
	skillDir := filepath.Join(dir, "has-git")
	writeFile(t, filepath.Join(skillDir, "SKILL.md"), "content")
	if err := os.MkdirAll(filepath.Join(skillDir, ".git"), 0755); err != nil {
		t.Fatal(err)
	}

	// Should not error; .git is simply ignored.
	_, err := WriteTreeFromDir(skillDir)
	if err != nil {
		t.Errorf("unexpected error with .git subdir: %v", err)
	}
}

// ── CommitTree ────────────────────────────────────────────────────────────────

func TestCommitTree_NoParent(t *testing.T) {
	dir := setupRepo(t)
	skillDir := filepath.Join(dir, "skill")
	writeFile(t, filepath.Join(skillDir, "SKILL.md"), "hello")

	tree, err := WriteTreeFromDir(skillDir)
	if err != nil {
		t.Fatal(err)
	}
	sha, err := CommitTree(tree, "Initial commit")
	if err != nil {
		t.Fatal(err)
	}
	if len(sha) != 40 {
		t.Errorf("commit SHA = %q (len %d), want 40-char hex", sha, len(sha))
	}

	typ, err := CatFile("-t", sha)
	if err != nil {
		t.Fatal(err)
	}
	if typ != "commit" {
		t.Errorf("object type = %q, want commit", typ)
	}
}

func TestCommitTree_WithParent(t *testing.T) {
	dir := setupRepo(t)
	skillDir := filepath.Join(dir, "skill")
	writeFile(t, filepath.Join(skillDir, "SKILL.md"), "v1")

	tree1, _ := WriteTreeFromDir(skillDir)
	parent, _ := CommitTree(tree1, "first")

	writeFile(t, filepath.Join(skillDir, "SKILL.md"), "v2")
	tree2, _ := WriteTreeFromDir(skillDir)
	child, err := CommitTree(tree2, "second", parent)
	if err != nil {
		t.Fatal(err)
	}

	// The child's parent must be the first commit.
	out, err := Run("rev-parse", child+"^")
	if err != nil {
		t.Fatal(err)
	}
	if out != parent {
		t.Errorf("child parent = %q, want %q", out, parent)
	}
}

// ── UpdateRef / RefExists / ResolveRef / DeleteRef ────────────────────────────

func TestUpdateRefAndRefExists(t *testing.T) {
	dir := setupRepo(t)
	writeFile(t, filepath.Join(dir, "skill", "SKILL.md"), "hello")
	tree, _ := WriteTreeFromDir(filepath.Join(dir, "skill"))
	commit, _ := CommitTree(tree, "init")

	ref := "refs/skills/test-skill"
	if RefExists(ref) {
		t.Error("ref should not exist before UpdateRef")
	}
	if err := UpdateRef(ref, commit); err != nil {
		t.Fatal(err)
	}
	if !RefExists(ref) {
		t.Error("ref should exist after UpdateRef")
	}
}

func TestResolveRef(t *testing.T) {
	dir := setupRepo(t)
	writeFile(t, filepath.Join(dir, "skill", "SKILL.md"), "hello")
	tree, _ := WriteTreeFromDir(filepath.Join(dir, "skill"))
	commit, _ := CommitTree(tree, "init")
	ref := "refs/skills/resolve-test"
	UpdateRef(ref, commit) //nolint:errcheck

	resolved, err := ResolveRef(ref)
	if err != nil {
		t.Fatal(err)
	}
	if resolved != commit {
		t.Errorf("ResolveRef = %q, want %q", resolved, commit)
	}
}

func TestDeleteRef(t *testing.T) {
	dir := setupRepo(t)
	writeFile(t, filepath.Join(dir, "skill", "SKILL.md"), "hello")
	tree, _ := WriteTreeFromDir(filepath.Join(dir, "skill"))
	commit, _ := CommitTree(tree, "init")
	ref := "refs/skills/delete-me"
	UpdateRef(ref, commit) //nolint:errcheck

	if err := DeleteRef(ref); err != nil {
		t.Fatal(err)
	}
	if RefExists(ref) {
		t.Error("ref should not exist after DeleteRef")
	}
}

// ── ReadTreeToDir ─────────────────────────────────────────────────────────────

func TestReadTreeToDir(t *testing.T) {
	dir := setupRepo(t)
	srcDir := filepath.Join(dir, "src")
	writeFile(t, filepath.Join(srcDir, "SKILL.md"), "hello world")
	writeFile(t, filepath.Join(srcDir, "references", "guide.md"), "guide")

	tree, err := WriteTreeFromDir(srcDir)
	if err != nil {
		t.Fatal(err)
	}

	destDir := filepath.Join(dir, "dest")
	if err := os.MkdirAll(destDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := ReadTreeToDir(tree, destDir); err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(filepath.Join(destDir, "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "hello world" {
		t.Errorf("SKILL.md content = %q, want hello world", content)
	}

	guide, err := os.ReadFile(filepath.Join(destDir, "references", "guide.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(guide) != "guide" {
		t.Errorf("guide.md content = %q, want guide", guide)
	}
}

// ── DiffTree ──────────────────────────────────────────────────────────────────

func TestDiffTree_Changes(t *testing.T) {
	dir := setupRepo(t)
	skillDir := filepath.Join(dir, "skill")
	writeFile(t, filepath.Join(skillDir, "SKILL.md"), "version one")

	tree1, _ := WriteTreeFromDir(skillDir)
	commit1, _ := CommitTree(tree1, "v1")

	writeFile(t, filepath.Join(skillDir, "SKILL.md"), "version two")
	tree2, _ := WriteTreeFromDir(skillDir)
	commit2, _ := CommitTree(tree2, "v2", commit1)

	out, err := DiffTree(commit1, commit2)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "SKILL.md") {
		t.Errorf("diff output should mention SKILL.md, got:\n%s", out)
	}
}

func TestDiffTree_NoChanges(t *testing.T) {
	dir := setupRepo(t)
	skillDir := filepath.Join(dir, "skill")
	writeFile(t, filepath.Join(skillDir, "SKILL.md"), "unchanged")

	tree, _ := WriteTreeFromDir(skillDir)
	commit, _ := CommitTree(tree, "same")

	out, err := DiffTree(commit, commit)
	if err != nil {
		t.Fatal(err)
	}
	if out != "" {
		t.Errorf("diffing a commit against itself should be empty, got %q", out)
	}
}

// ── ForEachRef ────────────────────────────────────────────────────────────────

func TestForEachRef(t *testing.T) {
	dir := setupRepo(t)
	writeFile(t, filepath.Join(dir, "skill", "SKILL.md"), "hello")
	tree, _ := WriteTreeFromDir(filepath.Join(dir, "skill"))
	commit, _ := CommitTree(tree, "init")
	UpdateRef("refs/skills/my-skill", commit) //nolint:errcheck

	lines, err := ForEachRef("refs/skills/", "%(refname)")
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 1 || lines[0] != "refs/skills/my-skill" {
		t.Errorf("ForEachRef = %v, want [refs/skills/my-skill]", lines)
	}
}

func TestForEachRef_Empty(t *testing.T) {
	setupRepo(t)
	lines, err := ForEachRef("refs/skills/", "%(refname)")
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 0 {
		t.Errorf("expected no refs, got %v", lines)
	}
}

// ── CatFile ───────────────────────────────────────────────────────────────────

func TestCatFile_Type(t *testing.T) {
	dir := setupRepo(t)
	writeFile(t, filepath.Join(dir, "skill", "SKILL.md"), "hello")
	tree, _ := WriteTreeFromDir(filepath.Join(dir, "skill"))

	typ, err := CatFile("-t", tree)
	if err != nil {
		t.Fatal(err)
	}
	if typ != "tree" {
		t.Errorf("CatFile -t tree = %q, want tree", typ)
	}
}

// ── MkTree ────────────────────────────────────────────────────────────────────

// ── Log ───────────────────────────────────────────────────────────────────────

func TestLog(t *testing.T) {
	dir := setupRepo(t)
	writeFile(t, filepath.Join(dir, "skill", "SKILL.md"), "content")
	tree, _ := WriteTreeFromDir(filepath.Join(dir, "skill"))
	commit, _ := CommitTree(tree, "Test commit message")
	ref := "refs/skills/log-test"
	UpdateRef(ref, commit) //nolint:errcheck

	out, err := Log(ref, "%s", 1)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Test commit message") {
		t.Errorf("Log output should contain commit subject, got %q", out)
	}
}

func TestLog_MaxCount(t *testing.T) {
	dir := setupRepo(t)
	skillDir := filepath.Join(dir, "skill")
	ref := "refs/skills/log-count"

	writeFile(t, filepath.Join(skillDir, "SKILL.md"), "v1")
	tree1, _ := WriteTreeFromDir(skillDir)
	c1, _ := CommitTree(tree1, "commit one")
	UpdateRef(ref, c1) //nolint:errcheck

	writeFile(t, filepath.Join(skillDir, "SKILL.md"), "v2")
	tree2, _ := WriteTreeFromDir(skillDir)
	c2, _ := CommitTree(tree2, "commit two", c1)
	UpdateRef(ref, c2) //nolint:errcheck

	writeFile(t, filepath.Join(skillDir, "SKILL.md"), "v3")
	tree3, _ := WriteTreeFromDir(skillDir)
	c3, _ := CommitTree(tree3, "commit three", c2)
	UpdateRef(ref, c3) //nolint:errcheck

	out, err := Log(ref, "%s", 2)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "commit one") {
		t.Errorf("maxCount=2 should not include third-oldest commit, got %q", out)
	}
	if !strings.Contains(out, "commit three") || !strings.Contains(out, "commit two") {
		t.Errorf("expected commits two and three in output, got %q", out)
	}
}

// ── WriteTreeFromDir (executable mode) ───────────────────────────────────────

func TestWriteTreeFromDir_ExecutableMode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not support Unix executable bits")
	}
	if os.Getuid() == 0 {
		t.Skip("chmod executable bit may not behave as expected as root")
	}
	dir := setupRepo(t)
	skillDir := filepath.Join(dir, "exec-skill")
	script := filepath.Join(skillDir, "run.sh")
	writeFile(t, script, "#!/bin/sh\necho hello\n")
	if err := os.Chmod(script, 0755); err != nil {
		t.Fatal(err)
	}

	treeSHA, err := WriteTreeFromDir(skillDir)
	if err != nil {
		t.Fatal(err)
	}
	// List the tree entries; the executable file should show mode 100755.
	out, err := Run("ls-tree", treeSHA)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "100755") {
		t.Errorf("executable file should appear as 100755 in tree, got:\n%s", out)
	}
}

// ── ReadTreeToDir error path ──────────────────────────────────────────────────

func TestReadTreeToDir_InvalidTree(t *testing.T) {
	dir := setupRepo(t)
	dest := filepath.Join(dir, "dest")
	os.MkdirAll(dest, 0755) //nolint:errcheck

	err := ReadTreeToDir("0000000000000000000000000000000000000000", dest)
	if err == nil {
		t.Fatal("expected error for invalid tree SHA")
	}
}

// ── WriteTreeFromDir error path ───────────────────────────────────────────────

func TestWriteTreeFromDir_NonExistentDir(t *testing.T) {
	setupRepo(t)
	_, err := WriteTreeFromDir("/nonexistent/path/that/does/not/exist")
	if err == nil {
		t.Fatal("expected error for non-existent directory")
	}
}

func TestMkTree(t *testing.T) {
	dir := setupRepo(t)
	f := filepath.Join(dir, "blob.txt")
	writeFile(t, f, "content")
	blobSHA, err := HashBlob(f)
	if err != nil {
		t.Fatal(err)
	}

	entry := "100644 blob " + blobSHA + "\tblob.txt\n"
	treeSHA, err := MkTree(entry)
	if err != nil {
		t.Fatal(err)
	}
	if len(treeSHA) != 40 {
		t.Errorf("MkTree SHA = %q (len %d), want 40 chars", treeSHA, len(treeSHA))
	}
}

// ── CommitTreeWithMessageFile ─────────────────────────────────────────────────

func TestCommitTreeWithMessageFilePreservesTrailers(t *testing.T) {
	dir := setupRepo(t) // sets cwd to dir for duration of test

	if err := os.WriteFile(filepath.Join(dir, "f.txt"), []byte("hi"), 0644); err != nil {
		t.Fatal(err)
	}
	blob, err := HashBlob(filepath.Join(dir, "f.txt"))
	if err != nil {
		t.Fatal(err)
	}
	tree, err := MkTree("100644 blob " + blob + "\tf.txt\n")
	if err != nil {
		t.Fatal(err)
	}

	msg := "feat: add f\n\nbody\n\nAsset-Kind: skill\n"
	commit, err := CommitTreeWithMessageFile(tree, msg)
	if err != nil {
		t.Fatalf("CommitTreeWithMessageFile: %v", err)
	}

	out, err := Run("cat-file", "-p", commit)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Asset-Kind: skill") {
		t.Errorf("trailer not preserved:\n%s", out)
	}
}
