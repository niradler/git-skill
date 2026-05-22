// Package assetignore parses .assetignore files (gitignore-style) and answers
// whether a given path within an asset should be excluded from a prod install.
// See spec L9.4.
//
// Discovery (per L9.4): a single file at the source repo root applies to every
// asset in the repo. A file at the asset directory root overrides the root file
// for that asset (no merge; closest file wins).
package assetignore

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type pattern struct {
	raw     string
	negate  bool
	dirOnly bool
	rooted  bool
	parts   []string
}

// Matcher holds a parsed set of .assetignore patterns.
type Matcher struct {
	patterns []pattern
}

// Empty returns a Matcher that never matches anything.
func Empty() *Matcher { return &Matcher{} }

// ParseString parses .assetignore content from a string.
func ParseString(s string) (*Matcher, error) {
	m := &Matcher{}
	for _, raw := range strings.Split(s, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		p := pattern{raw: line}
		if strings.HasPrefix(line, "!") {
			p.negate = true
			line = line[1:]
		}
		if strings.HasPrefix(line, "/") {
			p.rooted = true
			line = line[1:]
		}
		if strings.HasSuffix(line, "/") {
			p.dirOnly = true
			line = strings.TrimSuffix(line, "/")
		}
		p.parts = strings.Split(line, "/")
		m.patterns = append(m.patterns, p)
	}
	return m, nil
}

// ParseFile parses an .assetignore file from disk. Returns Empty() if not found.
func ParseFile(path string) (*Matcher, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Empty(), nil
		}
		return nil, err
	}
	defer f.Close()

	var buf strings.Builder
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		buf.WriteString(sc.Text())
		buf.WriteByte('\n')
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return ParseString(buf.String())
}

// LoadFromTree parses the .assetignore in assetDir if present.
func LoadFromTree(assetDir string) (*Matcher, error) {
	return ParseFile(filepath.Join(assetDir, ".assetignore"))
}

// Discover returns the effective Matcher for assetDir inside repoRoot.
// If assetDir contains a .assetignore, it fully replaces the root one (no merge).
// Otherwise the root .assetignore is used.
func Discover(repoRoot, assetDir string) (*Matcher, error) {
	perAsset := filepath.Join(assetDir, ".assetignore")
	if _, err := os.Stat(perAsset); err == nil {
		return ParseFile(perAsset)
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("stat %s: %w", perAsset, err)
	}
	root := filepath.Join(repoRoot, ".assetignore")
	return ParseFile(root)
}

// Match reports whether relPath should be ignored.
// A trailing slash in relPath signals the entry is a directory.
func (m *Matcher) Match(relPath string) bool {
	isDir := strings.HasSuffix(relPath, "/")
	return m.MatchPath(strings.TrimSuffix(relPath, "/"), isDir)
}

// MatchPath reports whether the relative path (no trailing slash) should be ignored.
// isDir must be true when the path refers to a directory entry.
func (m *Matcher) MatchPath(relPath string, isDir bool) bool {
	relPath = filepath.ToSlash(relPath)
	relPath = strings.TrimSuffix(relPath, "/")
	matched := false
	for _, p := range m.patterns {
		if p.matchesPath(relPath, isDir) {
			matched = !p.negate
		}
	}
	return matched
}

// matchesPath checks whether a single pattern matches the given path.
// For dirOnly patterns we match:
//   - the directory entry itself (isDir=true, path == pattern)
//   - any path that starts with the pattern directory prefix (path has pattern segs as prefix)
func (p pattern) matchesPath(path string, isDir bool) bool {
	segs := strings.Split(path, "/")

	if p.rooted {
		return p.matchSegsRooted(segs, isDir)
	}
	// Unrooted: try anchoring at every starting segment.
	for i := 0; i < len(segs); i++ {
		if p.matchSegsRooted(segs[i:], isDir && i == 0) {
			return true
		}
	}
	return false
}

// matchSegsRooted attempts to match p.parts against segs starting from the root.
func (p pattern) matchSegsRooted(segs []string, isDir bool) bool {
	if p.dirOnly {
		// A dirOnly pattern matches:
		//   1. The directory entry itself: segs == p.parts and isDir
		//   2. Any path whose leading segments match p.parts (the path is inside that dir)
		if len(segs) == len(p.parts) {
			return segmentsMatch(p.parts, segs) && (isDir || true)
			// gitignore: trailing / means "this dir OR anything inside it"
		}
		if len(segs) > len(p.parts) {
			return segmentsMatch(p.parts, segs[:len(p.parts)])
		}
		return false
	}
	return len(segs) == len(p.parts) && segmentsMatch(p.parts, segs)
}

// segmentsMatch checks whether pat (a slice of glob patterns) matches seg exactly
// (same length, each element glob-matched).
// Supports ** in pat: ** consumes zero or more path segments.
func segmentsMatch(pat, seg []string) bool {
	if len(pat) == 0 {
		return len(seg) == 0
	}
	if pat[0] == "**" {
		// ** consumes 0..len(seg) segments
		for i := 0; i <= len(seg); i++ {
			if segmentsMatch(pat[1:], seg[i:]) {
				return true
			}
		}
		return false
	}
	if len(seg) == 0 {
		return false
	}
	if !globMatch(pat[0], seg[0]) {
		return false
	}
	return segmentsMatch(pat[1:], seg[1:])
}

// globMatch matches a single path segment against a glob pattern (supports * wildcard).
// * matches any sequence of characters within a single segment (does not cross /).
func globMatch(pat, name string) bool {
	if pat == "*" {
		return true
	}
	// Simple linear scan with backtracking for *.
	pi, ni := 0, 0
	starPi, starNi := -1, -1
	for ni < len(name) {
		if pi < len(pat) && (pat[pi] == name[ni] || pat[pi] == '?') {
			pi++
			ni++
		} else if pi < len(pat) && pat[pi] == '*' {
			starPi = pi
			starNi = ni
			pi++
			// * matches empty string: advance pattern, not name
		} else if starPi != -1 {
			// backtrack: * consumes one more char
			starNi++
			ni = starNi
			pi = starPi + 1
		} else {
			return false
		}
	}
	// Consume trailing stars
	for pi < len(pat) && pat[pi] == '*' {
		pi++
	}
	return pi == len(pat)
}
