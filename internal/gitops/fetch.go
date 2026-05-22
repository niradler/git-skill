package gitops

import (
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
func FetchPinnedCommit(remote, ref, commit string) error {
	if _, err := git.Run("fetch", "--no-tags", remote, ref); err != nil {
		return err
	}
	// Verify the expected commit landed.
	if _, err := git.Run("cat-file", "-e", commit); err != nil {
		return err
	}
	return nil
}
