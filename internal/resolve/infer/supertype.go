package infer

import (
	"path"
	"strings"

	"github.com/satyendrakumarsingh/repocortex/internal/registry"
)

// # A qualified supertype is not its unqualified namesake
//
// `class InputStream extends java.io.OutputStream` names the JDK's
// OutputStream. os-lib also declares a nested `SubProcess.OutputStream`, and the
// supertype was bound to it at `high` because the name matches and it is in the
// same file. Both of the file's stream classes extended the wrong one.
//
// What the parser must emit for this rule to act (it does not yet): the
// qualifier a supertype is written with, as the reference's receiver. Today
// `extends java.io.OutputStream` is recorded as a reference named OutputStream
// with an empty receiver, which cannot be told from `extends OutputStream`, and
// nothing in the resolver can recover it. The qualifier is recorded the way a
// call's is: the path as written, `java.io` (`::` is read the same as `.`), with
// the reference's kind unchanged (reference.extends, reference.implements).
//
// With it, a supertype written with a qualifier is bound only to a type the
// qualifier fits, and the qualifier is read as:
//
//   - a root the language owns (java, javax, scala, kotlin): the standard
//     library, external;
//   - a package, shown by a directory of that name in the repository, which
//     the type's file lies under, or the type an inner type is nested in
//     (`Outer.Inner`);
//   - anything else: not a package of this repository, so external. A name
//     the repository declares unqualified, in the same file or anywhere, is
//     not what a qualified clause reaches.

// qualifiedSupertype narrows the candidates of a supertype reference by the
// qualifier it was written with. kept is what the qualifier fits; when it is
// empty the verdict says whether the supertype leaves the repository.
func (r *Resolver) qualifiedSupertype(spec LanguageSpec, ref registry.Reference, cands []registry.SymbolIndexEntry) (kept []registry.SymbolIndexEntry, v verdict) {
	segs := receiverSegments(ref.Receiver)
	if len(segs) == 0 {
		return cands, verdictBound
	}
	if spec.hasExternalRoot(segs) {
		return nil, verdictExternal
	}
	for _, c := range cands {
		if supertypeQualifierFits(segs, c) {
			kept = append(kept, c)
		}
	}
	if len(kept) > 0 {
		return kept, verdictBound
	}
	if r.layout.Known() && !r.layout.HasDirNamed(segs[0]) && !hasNestingOwner(segs, cands) {
		return nil, verdictExternal
	}
	return nil, verdictUnresolved
}

// supertypeQualifierFits reports whether a type is where a qualifier says: the
// qualifier is the tail of the type's own qualified name (nesting), or the
// directories above the type's file end with it (a package).
func supertypeQualifierFits(segs []string, c registry.SymbolIndexEntry) bool {
	qual, _ := strings.CutPrefix(c.FQN, c.FilePath+":")
	typeSegs := strings.Split(qual, ".")
	if len(typeSegs) > 1 && len(segs) <= len(typeSegs)-1 {
		nested := true
		for i := 1; i <= len(segs); i++ {
			if segs[len(segs)-i] != typeSegs[len(typeSegs)-1-i] {
				nested = false
				break
			}
		}
		if nested {
			return true
		}
	}
	var dir []string
	for _, p := range strings.Split(path.Dir(c.FilePath), "/") {
		if p != "" && p != "." {
			dir = append(dir, p)
		}
	}
	if len(dir) < len(segs) {
		return false
	}
	tail := dir[len(dir)-len(segs):]
	for i, seg := range segs {
		if !strings.EqualFold(tail[i], seg) {
			return false
		}
	}
	return true
}

// hasNestingOwner reports whether the qualifier's last segment names a type
// that some candidate is nested in: the qualifier is then a type of this
// repository even if no candidate fits it.
func hasNestingOwner(segs []string, cands []registry.SymbolIndexEntry) bool {
	last := segs[len(segs)-1]
	for _, c := range cands {
		if strings.Contains(strings.TrimPrefix(c.FQN, c.FilePath+":"), last+".") {
			return true
		}
	}
	return false
}
