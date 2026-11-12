package web

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	gogit "github.com/go-git/go-git/v5"
)

// TestOwnershipSaysWhenTheHistoryEndsAtACloneBoundary.
//
// A CI checkout holds the last few commits. The walk used to fail there; when
// it does not fail it returns fewer commits than were asked for, which is the
// shape this screen read as "the repository's whole history". It must say the
// clone is shallow, and the remedy it offers must be the one that helps:
// raising ?commits= reads nothing more from a clone that has nothing more.
func TestOwnershipSaysWhenTheHistoryEndsAtACloneBoundary(t *testing.T) {
	t.Parallel()

	root := scriptedHistory(t)
	repo, err := gogit.PlainOpen(root)
	if err != nil {
		t.Fatal(err)
	}
	iter, err := repo.Log(&gogit.LogOptions{})
	if err != nil {
		t.Fatal(err)
	}
	var hashes []string // newest first
	for {
		c, nerr := iter.Next()
		if nerr != nil {
			break
		}
		hashes = append(hashes, c.Hash.String())
	}
	if len(hashes) != 4 {
		t.Fatalf("scripted history has %d commits, want 4", len(hashes))
	}
	// As `clone --depth 3` leaves it: the third commit is the boundary and its
	// parent is not in the object store.
	gone := hashes[3]
	if err := os.Remove(filepath.Join(root, ".git", "objects", gone[:2], gone[2:])); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".git", "shallow"), []byte(hashes[2]+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	srv := scriptedServer(t, root)
	var got OwnershipResponse
	pillarGet(t, srv.handleOwnership, "/api/who?symbol=Alpha&commits=50", &got)

	if got.Status != BlastOK || !got.History.Read {
		t.Fatalf("status = %q, history read %v: %s", got.Status, got.History.Read, got.History.Note)
	}
	if got.Window.CommitsRead != 3 || !got.Window.Shallow || got.Window.Exhausted {
		t.Fatalf("window = %+v, want 3 commits read, shallow, not exhausted", got.Window)
	}
	var stated *Omission
	for i, om := range got.Omissions.Omitted {
		if om.Reason == ReasonTraversalBound {
			stated = &got.Omissions.Omitted[i]
		}
	}
	if stated == nil {
		t.Fatalf("a history cut by the clone stated no omission: %+v", got.Omissions.Omitted)
	}
	if !strings.Contains(stated.Detail, "Shallow clone: 3 commit(s) available") {
		t.Errorf("omission = %q, want the clone depth stated", stated.Detail)
	}
	if !strings.Contains(stated.Remedy, "fetch --unshallow") {
		t.Errorf("remedy = %q, want the fetch that brings in the history", stated.Remedy)
	}
}
