// Package fs handles cross-platform installation of asset trees and runtime
// fan-out: relative symlinks on Unix/macOS, real directory junctions on
// Windows, copy fallback if neither succeeds. Spec L9.3.
package fs

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Mode controls how EnsureLinkWithMode creates the link.
type Mode int

const (
	ModeAuto Mode = iota // OS link → copy fallback
	ModeCopy             // always copy
)

// LinkResult describes what EnsureLink/EnsureLinkWithMode did.
type LinkResult struct {
	Method         string // "symlink" | "junction" | "copy"
	AlreadyCorrect bool   // true when no action was needed
}

// ErrTargetEscape is returned when target starts with ".." after Clean.
var ErrTargetEscape = errors.New("link target escapes link parent")

// EnsureLink installs `target` at `link`. If `dirOnly` is true, target is
// treated as a directory; otherwise as a file. Idempotent: if `link` already
// resolves to `target` (either by symlink/junction or by identical realpath),
// returns AlreadyCorrect=true with no further action.
func EnsureLink(target, link string, dirOnly bool) (*LinkResult, error) {
	return EnsureLinkWithMode(target, link, dirOnly, ModeAuto)
}

// EnsureLinkWithMode is like EnsureLink but lets the caller choose the mode.
func EnsureLinkWithMode(target, link string, dirOnly bool, mode Mode) (*LinkResult, error) {
	if err := validateLinkTarget(target); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(link), 0755); err != nil {
		return nil, fmt.Errorf("mkdir parent of %s: %w", link, err)
	}
	if ok, err := isLinkAlreadyCorrect(target, link); err != nil {
		return nil, err
	} else if ok {
		return &LinkResult{Method: detectMethod(link), AlreadyCorrect: true}, nil
	}
	if err := removeExisting(link); err != nil {
		return nil, fmt.Errorf("remove existing %s: %w", link, err)
	}

	if mode == ModeAuto {
		if method, ok := tryPlatformLink(target, link, dirOnly); ok {
			return &LinkResult{Method: method}, nil
		}
	}
	if dirOnly {
		if err := CopyTree(target, link, nil); err != nil {
			return nil, err
		}
	} else {
		if err := CopyFile(target, link); err != nil {
			return nil, err
		}
	}
	return &LinkResult{Method: "copy"}, nil
}

func validateLinkTarget(target string) error {
	if target == "" {
		return fmt.Errorf("empty target")
	}
	clean := filepath.Clean(target)
	if strings.HasPrefix(clean, "..") {
		return ErrTargetEscape
	}
	return nil
}

func relTarget(target, link string) (string, error) {
	targetAbs, err := filepath.Abs(target)
	if err != nil {
		return "", err
	}
	linkParent, err := filepath.Abs(filepath.Dir(link))
	if err != nil {
		return "", err
	}
	return filepath.Rel(linkParent, targetAbs)
}

func absTarget(target string) (string, error) {
	return filepath.Abs(target)
}

func isLinkAlreadyCorrect(target, link string) (bool, error) {
	info, err := os.Lstat(link)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	// Reparse points (symlinks on any OS; junctions on Windows) both set
	// ModeSymlink in modern Go runtimes. Compare resolved realpath.
	if info.Mode()&os.ModeSymlink != 0 {
		realLink, err := filepath.EvalSymlinks(link)
		if err != nil {
			return false, nil
		}
		realTarget, err := filepath.EvalSymlinks(target)
		if err != nil {
			return false, nil
		}
		return realLink == realTarget, nil
	}
	if realLink, err := filepath.EvalSymlinks(link); err == nil {
		if realTarget, err2 := filepath.EvalSymlinks(target); err2 == nil {
			return realLink == realTarget, nil
		}
	}
	return false, nil
}

func detectMethod(link string) string {
	info, err := os.Lstat(link)
	if err != nil {
		return ""
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return platformLinkLabel()
	}
	return "copy"
}

func removeExisting(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if info.IsDir() && info.Mode()&os.ModeSymlink == 0 {
		return os.RemoveAll(path)
	}
	return os.Remove(path)
}

// TODO(task-8): remove these stubs when fs/copy.go lands.

// CopyTree recursively copies src to dst. The filter func (if non-nil) is
// called for each entry; returning false skips it.
func CopyTree(src, dst string, _ func(string) bool) error {
	return fmt.Errorf("CopyTree not implemented yet")
}

// CopyFile copies a single file from src to dst.
func CopyFile(src, dst string) error {
	return fmt.Errorf("CopyFile not implemented yet")
}
