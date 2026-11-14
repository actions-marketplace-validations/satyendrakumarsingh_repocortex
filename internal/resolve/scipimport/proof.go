package scipimport

import (
	"context"
	"sort"
	"strings"

	"github.com/satyendrakumarsingh/repocortex/internal/registry"
	"github.com/satyendrakumarsingh/repocortex/internal/resolve/scip"
)

// Proof collects, across every compiler index imported for a run, what the
// answer layer needs to stop counting an unbound reference as a possible use:
// which files the compilers read, and which positions they resolved to
// something this repository does not define.
//
// It is filled by an Importer that was given one (Importer.Collect) and written
// once, after the last import, by Apply. It is not filled for an index the user
// supplied: an index built from some other commit would claim coverage of
// files it never read.
//
// WHAT COUNTS AS RULED OUT. A reference row is spelled like the definition an
// answer is about, was bound to nothing, and sits where a compiler resolved an
// occurrence spelled the same way to a symbol of another package, one this
// repository does not define. The test is made per (file, line, name):
//
//   - every occurrence of that name on the line must resolve outside the
//     repository (none to a symbol the index defines, matched or not, and none
//     to a package that owns a definition in any imported index, since a
//     sibling project's symbol arrives under its package name and has no
//     counterpart here);
//   - there must be at least as many such occurrences as the parser recorded
//     references of that name on the line, so a second reference the compiler
//     did not resolve cannot hide behind the first.
//
// A position the compiler did not resolve at all (a receiver of unknown type, a
// dynamic attribute) is never ruled out: no occurrence, no proof. That is what
// keeps the rule sound for Python and for TypeScript's `any`, where the Go
// rule "the checker read the file and bound nothing" would be false.
type Proof struct {
	covered map[string]string
	sites   map[siteAt]*siteTally
	own     map[string]struct{}
}

type siteAt struct {
	path string
	line int
	name string
}

type siteTally struct {
	outside int
	inRepo  int
	pkgs    map[string]struct{}
}

// NewProof returns an empty Proof.
func NewProof() *Proof {
	return &Proof{
		covered: map[string]string{},
		sites:   map[siteAt]*siteTally{},
		own:     map[string]struct{}{},
	}
}

// cover records that tool read path and which packages it defines.
func (p *Proof) cover(path, tool string, packages []string) {
	p.covered[path] = tool
	for _, pkg := range packages {
		p.own[normalizePackage(pkg)] = struct{}{}
	}
}

// AddOwn declares a package the repository itself provides, as "manager name"
// the way a SCIP symbol spells it. The indexes name the repository's own
// definitions after the project, but an indexer that finds an INSTALLED copy of
// the same library (pallets/click's own tests run against the click in site-
// packages) resolves imports to that copy, under the library's real name: a
// reference resolved there is a use of repository code resolved to a duplicate
// outside it, and must not be read as "something else".
func (p *Proof) AddOwn(manager, name string) {
	if manager != "" && name != "" {
		p.own[normalizePackage(manager+" "+name)] = struct{}{}
	}
}

// normalizePackage makes the spellings of one package compare equal: case, and
// the hyphen or underscore a distribution name and an import name differ by.
func normalizePackage(pkg string) string {
	return strings.ReplaceAll(strings.ToLower(pkg), "-", "_")
}

// site records one occurrence; inRepo says the symbol is defined by an index.
func (p *Proof) site(path string, s scip.Site, inRepo bool) {
	if s.Local || s.Name == "" {
		// A local symbol is a variable or a parameter: nothing about it says
		// what name it is spelled as, so it can neither rule a reference out
		// nor block a ruling on the same line.
		return
	}
	k := siteAt{path: path, line: s.Line, name: s.Name}
	t := p.sites[k]
	if t == nil {
		t = &siteTally{}
		p.sites[k] = t
	}
	if inRepo || s.Package == "" {
		// A symbol whose package cannot be read is treated as the
		// repository's own: nothing can be proven about it.
		t.inRepo++
		return
	}
	t.outside++
	if t.pkgs == nil {
		t.pkgs = map[string]struct{}{}
	}
	t.pkgs[s.Package] = struct{}{}
}

// ProofStats is what Apply wrote.
type ProofStats struct {
	// Files is how many indexed files a compiler read.
	Files int
	// RuledOut is how many references were marked.
	RuledOut int
	// Unbound is how many of them the resolver had bound to a definition of the
	// repository: the compiler resolved that position outside it, so the
	// binding was removed.
	Unbound int
}

// Apply writes the proof: the covered files and the ruled-out references. It
// first clears what an earlier resolve recorded, so the run describes the
// indexes imported now and no others.
func (p *Proof) Apply(ctx context.Context, reg *registry.Registry, runID int64) (ProofStats, int, error) {
	var st ProofStats
	if err := reg.ClearCompilerProof(ctx, runID); err != nil {
		return st, 0, err
	}
	byTool := map[string][]string{}
	for path, tool := range p.covered {
		byTool[tool] = append(byTool[tool], path)
	}
	tools := make([]string, 0, len(byTool))
	for t := range byTool {
		tools = append(tools, t)
	}
	sort.Strings(tools)
	for _, t := range tools {
		n, err := reg.RecordCompilerCoverage(ctx, runID, t, byTool[t])
		if err != nil {
			return st, 0, err
		}
		st.Files += n
	}
	if st.Files == 0 {
		return st, 0, nil
	}

	var (
		cur      siteAt
		have     bool
		ids      []int64
		markable []int64
	)
	var marks []int64
	ruled := map[siteAt]struct{}{}
	flush := func() {
		if !have {
			return
		}
		if t := p.sites[cur]; t != nil && p.ruledOut(t, len(ids)) {
			ruled[cur] = struct{}{}
			marks = append(marks, markable...)
		}
		ids, markable, have = ids[:0], markable[:0], false
	}
	err := reg.CoveredReferences(ctx, runID, func(r registry.RefSite) error {
		k := siteAt{path: r.Path, line: r.Line, name: r.Name}
		if have && k != cur {
			flush()
		}
		cur, have = k, true
		ids = append(ids, r.ID)
		if !r.ByCompiler {
			markable = append(markable, r.ID)
		}
		return nil
	})
	if err != nil {
		return st, 0, err
	}
	flush()
	n, wasBound, err := reg.MarkRuledOut(ctx, marks)
	st.RuledOut, st.Unbound = n, wasBound
	if err != nil {
		return st, 0, err
	}
	withdrawn, err := p.withdraw(ctx, reg, runID, ruled)
	return st, withdrawn, err
}

// withdraw deletes the heuristic call edges the compilers proved false: an edge
// whose call site is in a covered file, whose target is declared in a covered
// file, and where the compiler resolved every call of that name on that line
// outside the repository. Same rule, same conditions as the Go type check's
// withdrawal (see Importer.withdrawContradicted), made with the same test that
// rules a reference out.
func (p *Proof) withdraw(ctx context.Context, reg *registry.Registry, runID int64, ruled map[siteAt]struct{}) (int, error) {
	if len(ruled) == 0 {
		return 0, nil
	}
	langs, err := reg.CoveredLanguages(ctx, runID)
	if err != nil {
		return 0, err
	}
	var drop []int64
	for _, lang := range langs {
		calls, err := reg.HeuristicCalls(ctx, runID, lang)
		if err != nil {
			return 0, err
		}
		for _, h := range calls {
			if _, ok := p.covered[h.File]; !ok {
				continue
			}
			if _, ok := p.covered[h.DstPath]; !ok {
				continue
			}
			if _, ok := ruled[siteAt{path: h.File, line: h.Line, name: h.DstName}]; ok {
				drop = append(drop, h.ID)
			}
		}
	}
	return reg.DeleteEdges(ctx, drop)
}

// ruledOut applies the rule on the type's comment to one (file, line, name)
// with refs references recorded on it.
func (p *Proof) ruledOut(t *siteTally, refs int) bool {
	if t.inRepo > 0 || t.outside < refs {
		return false
	}
	for pkg := range t.pkgs {
		if _, own := p.own[normalizePackage(pkg)]; own {
			return false
		}
	}
	return true
}
