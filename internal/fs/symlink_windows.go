//go:build windows

package fs

import (
	"syscall"

	"golang.org/x/sys/windows"
)

// SYMBOLIC_LINK_FLAG_ALLOW_UNPRIVILEGED_CREATE allows unprivileged symlinks
// when Developer Mode is enabled (Windows 10 1703+). Value 0x2 per MSDN.
const symbolicLinkFlagAllowUnprivilegedCreate uint32 = 0x2

// tryPlatformLink creates either a directory junction (dirOnly=true) or a
// file symlink (dirOnly=false). Both go through CreateSymbolicLinkW with the
// unprivileged flag set so Developer Mode users don't need admin.
// Junction-style symlinks must use absolute targets per the OS contract.
func tryPlatformLink(target, link string, dirOnly bool) (string, bool) {
	abs, err := absTarget(target)
	if err != nil {
		return "", false
	}
	linkU16, err := syscall.UTF16PtrFromString(link)
	if err != nil {
		return "", false
	}
	targetU16, err := syscall.UTF16PtrFromString(abs)
	if err != nil {
		return "", false
	}
	var flags uint32 = symbolicLinkFlagAllowUnprivilegedCreate
	if dirOnly {
		flags |= windows.SYMBOLIC_LINK_FLAG_DIRECTORY
	}
	if err := windows.CreateSymbolicLink(linkU16, targetU16, flags); err != nil {
		return "", false
	}
	if dirOnly {
		return "junction", true
	}
	return "symlink", true
}

func platformLinkLabel() string {
	// Best-effort: most consumer installs are directory installs, where the
	// label is "junction". CopyTree-fallback callers will see "copy" instead.
	return "junction"
}
