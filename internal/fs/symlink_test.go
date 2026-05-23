package fs

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func validLinkMethod(m string) bool {
	switch m {
	case "symlink", "junction", "copy":
		return true
	}
	return false
}

func TestEnsureLink_NewLink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	if err := os.MkdirAll(target, 0755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link")
	res, err := EnsureLink(target, link, true)
	if err != nil {
		t.Fatal(err)
	}
	if !validLinkMethod(res.Method) {
		t.Errorf("Method = %q (expected symlink/junction/copy)", res.Method)
	}
	if _, err := os.Stat(link); err != nil {
		t.Errorf("link not present: %v", err)
	}
}

func TestEnsureLink_Idempotent(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	if err := os.MkdirAll(target, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(target, "x"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link")
	if _, err := EnsureLink(target, link, true); err != nil {
		t.Fatal(err)
	}
	res, err := EnsureLink(target, link, true)
	if err != nil {
		t.Fatal(err)
	}
	if !res.AlreadyCorrect {
		t.Errorf("second call should be no-op, AlreadyCorrect=false (Method=%q)", res.Method)
	}
}

func TestEnsureLink_RelativeTarget(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink details differ on Windows")
	}
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	if err := os.MkdirAll(target, 0755); err != nil {
		t.Fatal(err)
	}
	subdir := filepath.Join(dir, "sub")
	if err := os.MkdirAll(subdir, 0755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(subdir, "link")
	res, err := EnsureLink(target, link, true)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && res.Method == "symlink" {
		got, err := os.Readlink(link)
		if err != nil {
			t.Fatal(err)
		}
		if filepath.IsAbs(got) {
			t.Errorf("expected relative symlink target, got absolute: %q", got)
		}
		if !strings.HasPrefix(got, "..") {
			t.Errorf("expected relative target starting with '..', got %q", got)
		}
	}
}

func TestEnsureLink_FileTarget(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "file.md")
	if err := os.WriteFile(target, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "linked.md")
	res, err := EnsureLink(target, link, false)
	if err != nil {
		t.Fatal(err)
	}
	if res.Method == "" {
		t.Error("Method empty")
	}
	data, err := os.ReadFile(link)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello" {
		t.Errorf("content = %q", data)
	}
}

func TestEnsureLink_ForceCopy(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	if err := os.MkdirAll(target, 0755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link")
	res, err := EnsureLinkWithMode(target, link, true, ModeCopy)
	if err != nil {
		t.Fatal(err)
	}
	if res.Method != "copy" {
		t.Errorf("Method = %q, want copy", res.Method)
	}
}

func TestEnsureLink_RejectsEscape(t *testing.T) {
	dir := t.TempDir()
	link := filepath.Join(dir, "link")
	_, err := EnsureLink("..\\..\\..\\windows", link, true)
	if err == nil || !errors.Is(err, ErrTargetEscape) {
		t.Errorf("expected ErrTargetEscape, got %v", err)
	}
}
