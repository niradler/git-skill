//go:build !windows

package fs

import "os"

// tryPlatformLink creates a relative symlink. Returns ("symlink", true) on
// success, ("", false) on failure so the caller can fall back to copy.
func tryPlatformLink(target, link string, _ bool) (string, bool) {
	rel, err := relTarget(target, link)
	if err != nil {
		return "", false
	}
	if err := os.Symlink(rel, link); err != nil {
		return "", false
	}
	return "symlink", true
}

func platformLinkLabel() string { return "symlink" }
