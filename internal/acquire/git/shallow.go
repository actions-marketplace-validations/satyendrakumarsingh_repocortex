package git

import (
	"os"
	"path/filepath"
)

// Shallow reports whether the repository was cloned with truncated history
// (`git clone --depth`, which is what most CI checkouts do).
//
// A history walk cannot tell this from the commits it returns: a shallow
// clone's oldest commit looks exactly like a repository's first one. Anything
// that draws a conclusion from how much history exists has to ask, or it
// reports "the whole history is three commits" for a repository with ten
// thousand.
//
// False for a directory that is not a repository, and false when the answer
// cannot be read: "not known to be shallow" is the only safe default, since
// the callers use true to add a warning.
func (r *Repo) Shallow() bool {
	if !r.isRepo {
		return false
	}
	if r.repo != nil {
		boundary, err := r.repo.Storer.Shallow()
		return err == nil && len(boundary) > 0
	}
	// Degraded mode: go-git declined the repository, so read the file git
	// itself records the boundary in. Empty means not shallow.
	if r.gitDir == "" {
		return false
	}
	info, err := os.Stat(filepath.Join(r.gitDir, "shallow"))
	return err == nil && !info.IsDir() && info.Size() > 0
}
