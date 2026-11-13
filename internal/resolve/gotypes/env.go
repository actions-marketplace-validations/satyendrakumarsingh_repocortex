package gotypes

import (
	"os"
	"path/filepath"
)

// findGoroot locates a Go installation's root WITHOUT executing anything.
//
// In order: the GOROOT environment variable; otherwise the first `go` on PATH,
// with symlinks resolved, whose directory's parent is the root (bin/go ->
// root). A candidate is accepted only when src/fmt exists under it, so a
// stale variable or a wrapper script in a bin directory is skipped rather than
// believed. Returns "" when none is found, which the caller reports as one
// line naming what to set.
func findGoroot(env func(string) string) string {
	ok := func(root string) bool {
		if root == "" {
			return false
		}
		fi, err := os.Stat(filepath.Join(root, "src", "fmt"))
		return err == nil && fi.IsDir()
	}
	if r := env("GOROOT"); ok(r) {
		return filepath.Clean(r)
	}
	for _, dir := range filepath.SplitList(env("PATH")) {
		if dir == "" {
			continue
		}
		cand := filepath.Join(dir, "go")
		fi, err := os.Stat(cand)
		if err != nil || fi.IsDir() {
			continue
		}
		real, err := filepath.EvalSymlinks(cand)
		if err != nil {
			continue
		}
		root := filepath.Dir(filepath.Dir(real))
		if ok(root) {
			return root
		}
	}
	return ""
}

// findModCache locates the module cache: GOMODCACHE, else $GOPATH/pkg/mod
// (the first GOPATH entry that has one), else ~/go/pkg/mod. Returns "" when
// none exists as a directory.
func findModCache(env func(string) string) string {
	isDir := func(p string) bool {
		fi, err := os.Stat(p)
		return err == nil && fi.IsDir()
	}
	if m := env("GOMODCACHE"); m != "" {
		if isDir(m) {
			return filepath.Clean(m)
		}
		return ""
	}
	for _, gp := range filepath.SplitList(env("GOPATH")) {
		if gp == "" {
			continue
		}
		if p := filepath.Join(gp, "pkg", "mod"); isDir(p) {
			return p
		}
	}
	if home, err := os.UserHomeDir(); err == nil {
		if p := filepath.Join(home, "go", "pkg", "mod"); isDir(p) {
			return p
		}
	}
	return ""
}
