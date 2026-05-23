package gitops

import (
	"fmt"

	"github.com/niradler/git-skill/internal/git"
)

// FetchPartial pulls a single refspec with --filter=blob:none --depth=1.
// Used by discovery: cheap to list refs and metadata without trees.
func FetchPartial(remote, refspec string) error {
	_, err := git.Run("fetch", "--filter=blob:none", "--depth=1", remote, refspec)
	return err
}

// FetchPinnedCommit pulls the named ref from remote (full objects). After
// success, `commit` is in the object DB and can be checked out into a
// tempdir via git.ReadTreeToDir.
func FetchPinnedCommit(remote, ref, commit string, fallbackRefs ...string) error {
	if _, err := git.Run("fetch", "--no-tags", remote, ref); err != nil {
		return err
	}
	// Verify the expected commit landed.
	if _, err := git.Run("cat-file", "-e", commit); err != nil {
		verifyErr := err
		for _, fallbackRef := range fallbackRefs {
			if fallbackRef == "" {
				continue
			}
			if _, fetchErr := git.Run("fetch", "--no-tags", remote, fallbackRef); fetchErr != nil {
				return fmt.Errorf("fetch fallback %s: %w", fallbackRef, fetchErr)
			}
			if _, catErr := git.Run("cat-file", "-e", commit); catErr == nil {
				return nil
			} else {
				verifyErr = catErr
			}
		}
		return verifyErr
	}
	return nil
}
