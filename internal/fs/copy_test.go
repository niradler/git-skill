package fs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCopyFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(src, []byte("hi"), 0644); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(dir, "b.txt")
	if err := CopyFile(src, dst); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(dst)
	if string(got) != "hi" {
		t.Errorf("content = %q", got)
	}
}

func TestCopyTree(t *testing.T) {
	src := t.TempDir()
	if err := os.MkdirAll(filepath.Join(src, "sub"), 0755); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(src, "root.txt"), []byte("r"), 0644)
	os.WriteFile(filepath.Join(src, "sub", "child.txt"), []byte("c"), 0644)

	dst := filepath.Join(t.TempDir(), "out")
	if err := CopyTree(src, dst, nil); err != nil {
		t.Fatal(err)
	}
	r, _ := os.ReadFile(filepath.Join(dst, "root.txt"))
	if string(r) != "r" {
		t.Error("root.txt missing or wrong")
	}
	c, _ := os.ReadFile(filepath.Join(dst, "sub", "child.txt"))
	if string(c) != "c" {
		t.Error("sub/child.txt missing or wrong")
	}
}

func TestCopyTree_Filter(t *testing.T) {
	src := t.TempDir()
	os.MkdirAll(filepath.Join(src, "eval"), 0755)
	os.WriteFile(filepath.Join(src, "SKILL.md"), []byte("ok"), 0644)
	os.WriteFile(filepath.Join(src, "eval", "x.txt"), []byte("skip"), 0644)

	dst := filepath.Join(t.TempDir(), "out")
	filter := func(rel string) bool { return strings.HasPrefix(rel, "eval") }
	if err := CopyTree(src, dst, filter); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dst, "SKILL.md")); err != nil {
		t.Error("SKILL.md should have been copied")
	}
	if _, err := os.Stat(filepath.Join(dst, "eval", "x.txt")); err == nil {
		t.Error("eval/x.txt should have been filtered out")
	}
}
