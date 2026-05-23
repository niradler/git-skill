// Package gitops layers asset-aware operations on top of internal/git plumbing.
//
// Commits carry an Asset-Kind: <skill|agent> trailer. This is the
// authoritative kind for a commit at write time. Readers consult it via
// ReadKindTrailer. See spec L3.
package gitops

import (
	"strings"

	"github.com/niradler/git-skill/internal/git"
	"github.com/niradler/git-skill/internal/kind"
)

const TrailerKey = "Asset-Kind"

// WriteCommitWithKind builds a commit whose message ends with the
// Asset-Kind trailer. parent may be "" for an initial commit.
func WriteCommitWithKind(tree, subject string, k kind.Kind, parent string) (string, error) {
	msg := subject + "\n\n" + TrailerKey + ": " + k.String() + "\n"
	if parent == "" {
		return git.CommitTreeWithMessageFile(tree, msg)
	}
	return git.CommitTreeWithMessageFile(tree, msg, parent)
}

// ReadKindTrailer pulls the Asset-Kind trailer from a commit. ok=false when
// the trailer is absent — callers fall through to the next tier of the
// discriminator (frontmatter, then marker filename).
func ReadKindTrailer(commitish string) (kind.Kind, bool, error) {
	body, err := git.Run("log", "-1", "--format=%B", commitish)
	if err != nil {
		return 0, false, err
	}
	for _, line := range strings.Split(body, "\n") {
		if strings.HasPrefix(line, TrailerKey+": ") {
			val := strings.TrimSpace(strings.TrimPrefix(line, TrailerKey+": "))
			k, err := kind.Parse(val)
			if err != nil {
				return 0, true, err
			}
			return k, true, nil
		}
	}
	return 0, false, nil
}
