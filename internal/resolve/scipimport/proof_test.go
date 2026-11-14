package scipimport

import (
	"context"
	"database/sql"
	"sync"
	"testing"

	scippb "github.com/scip-code/scip/bindings/go/scip"

	"github.com/satyendrakumarsingh/repocortex/internal/registry"
	"github.com/satyendrakumarsingh/repocortex/internal/resolve/scip"
	"github.com/satyendrakumarsingh/repocortex/internal/store"
	"github.com/satyendrakumarsingh/repocortex/pkg/model"
)

const (
	tsSvc  = "src/svc.ts"
	tsUse  = "src/use.ts"
	tsSkip = "src/skipped.ts"

	tsGetSym   = "scip-typescript npm proj 1.0.0 `src/svc.ts`/Service#get()."
	tsOtherGet = "scip-typescript npm lodash 4.17.0 `index.d.ts`/get()."
	tsOwnGet   = "scip-typescript npm sibling 1.0.0 `src/x.ts`/get()."
	tsUseSym   = "scip-typescript npm proj 1.0.0 `src/use.ts`/run()."
)

// testDBs maps a test registry to its database, for tests that must read or
// write a table the registry has no method for.
var testDBs sync.Map

func dbOf(reg *registry.Registry) *store.DB {
	db, _ := testDBs.Load(reg)
	return db.(*store.DB)
}

// proofRepo is a TypeScript repository of three files: svc.ts defines the
// method get, use.ts calls get several ways, and skipped.ts calls get too but
// is outside the indexer's program.
func proofRepo(t *testing.T) (*registry.Registry, int64, map[string]int64) {
	t.Helper()
	ctx := context.Background()
	db, err := store.OpenMemory(ctx)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	reg := registry.New(db)
	testDBs.Store(reg, db)
	run, err := reg.BeginRun(ctx, "auto", "cafe")
	if err != nil {
		t.Fatal(err)
	}
	w, err := reg.NewWriter(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	ids := map[string]int64{}
	file := func(path string, syms ...model.Symbol) int64 {
		fid, err := w.AddFile(ctx, &model.File{Path: path, Language: "typescript", ContentHash: "h" + path})
		if err != nil {
			t.Fatal(err)
		}
		for i := range syms {
			syms[i].FileID = fid
			syms[i].FQN = path + ":" + syms[i].Name
			syms[i].Confidence = model.ConfidenceHigh
			syms[i].Source = model.SourceSourceCode
			id, err := w.AddSymbol(ctx, &syms[i])
			if err != nil {
				t.Fatal(err)
			}
			ids[syms[i].Name] = id
		}
		return fid
	}
	at := func(n int) model.Range {
		return model.Range{Start: model.Position{Line: n}, End: model.Position{Line: n + 2}}
	}
	file(tsSvc, model.Symbol{Kind: model.KindMethod, Name: "get", Range: at(6)})
	use := file(tsUse, model.Symbol{Kind: model.KindFunction, Name: "run", Range: at(4)})
	skip := file(tsSkip, model.Symbol{Kind: model.KindFunction, Name: "skippedFn", Range: at(2)})
	for _, r := range []struct {
		fid  int64
		line int
	}{{use, 10}, {use, 11}, {use, 12}, {use, 13}, {skip, 5}} {
		if err := w.AddReference(ctx, r.fid, registry.Reference{
			Name: "get", Kind: "reference.call", Line: r.line, Col: 8, ArgCount: model.ArityUnknown,
		}); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Flush(ctx); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if err := reg.CommitRun(ctx, run.ID); err != nil {
		t.Fatal(err)
	}
	// None of the references is bound: the heuristic resolver left them.
	if _, err := db.ExecContext(ctx, `UPDATE symbol_references SET resolution = 'unresolved'`); err != nil {
		t.Fatal(err)
	}
	return reg, run.ID, ids
}

func tsDocs() []*scippb.Document {
	enc := scippb.PositionEncoding_UTF16CodeUnitOffsetFromLineStart
	return []*scippb.Document{
		{
			RelativePath: tsSvc, Language: "typescript", PositionEncoding: enc,
			Symbols:     []*scippb.SymbolInformation{{Symbol: tsGetSym, Kind: scippb.SymbolInformation_Method, DisplayName: "get"}},
			Occurrences: []*scippb.Occurrence{def(tsGetSym, rng(5, 2, 5, 5), rng(5, 0, 7, 1))},
		},
		{
			RelativePath: tsUse, Language: "typescript", PositionEncoding: enc,
			Symbols: []*scippb.SymbolInformation{{Symbol: tsUseSym, Kind: scippb.SymbolInformation_Function, DisplayName: "run"}},
			Occurrences: []*scippb.Occurrence{
				def(tsUseSym, rng(3, 9, 3, 12), rng(3, 0, 20, 1)),
				// line 10: resolved to lodash, outside the repository.
				ref(tsOtherGet, rng(9, 8, 9, 11)),
				// line 11: no occurrence at all: the receiver had no type.
				// line 12: resolved to the repository's own method.
				ref(tsGetSym, rng(11, 8, 11, 11)),
				// line 13: resolved to a package a sibling project provides.
				ref(tsOwnGet, rng(12, 8, 12, 11)),
			},
		},
	}
}

func importProof(t *testing.T, reg *registry.Registry, runID int64, own bool) ProofStats {
	t.Helper()
	ctx := context.Background()
	proof := NewProof()
	if own {
		proof.AddOwn("npm", "sibling")
	}
	path := scipFile(t, tsDocs()...)
	imp := NewImporter(reg, runID, scip.Options{RepoRoot: "/build/repo"}).Collect(proof)
	if _, err := imp.Import(ctx, path); err != nil {
		t.Fatalf("import: %v", err)
	}
	st, _, err := proof.Apply(ctx, reg, runID)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	return st
}

func TestProofRulesOutOnlyWhatTheCompilerResolvedElsewhere(t *testing.T) {
	reg, runID, ids := proofRepo(t)
	st := importProof(t, reg, runID, true)

	if st.Files != 2 {
		t.Errorf("covered %d files, want 2 (svc.ts and use.ts; skipped.ts is not a document)", st.Files)
	}
	if st.RuledOut != 1 {
		t.Fatalf("ruled out %d references, want exactly the lodash one at line 10", st.RuledOut)
	}

	b, err := reg.NameBinding(context.Background(), runID, ids["get"], "get")
	if err != nil {
		t.Fatal(err)
	}
	if b.ProvenOther != 1 {
		t.Errorf("ProvenOther = %d, want 1 (line 10)", b.ProvenOther)
	}
	// Line 12 is bound to get by the compiler. Left unbound and counted: line 11
	// (the compiler resolved nothing there), line 13 (a sibling project's
	// symbol), and skipped.ts line 5 (a file the indexer never read).
	if got := b.Unresolved + b.Ambiguous + b.Pending; got != 3 {
		t.Errorf("%d references still unbound, want 3: %+v", got, b)
	}
}

func TestProofDoesNotTrustAPackageTheRepositoryProvidesAnotherWay(t *testing.T) {
	reg, runID, ids := proofRepo(t)
	// Without declaring the sibling package as the repository's own, the
	// symbol at line 13 reads as a library's: that is the case AddOwn exists
	// for, so here it is ruled out, and with AddOwn (the test above) it is not.
	st := importProof(t, reg, runID, false)
	if st.RuledOut != 2 {
		t.Fatalf("ruled out %d without AddOwn, want 2 (lines 10 and 13)", st.RuledOut)
	}
	b, err := reg.NameBinding(context.Background(), runID, ids["get"], "get")
	if err != nil {
		t.Fatal(err)
	}
	if b.ProvenOther != 2 {
		t.Errorf("ProvenOther = %d, want 2", b.ProvenOther)
	}
}

func TestProofIgnoresADefinitionTheIndexerNeverRead(t *testing.T) {
	reg, runID, ids := proofRepo(t)
	importProof(t, reg, runID, true)

	// skippedFn lives in a file the indexer did not cover; a definition there
	// named like a reference that WAS ruled out elsewhere must not be settled.
	b, err := reg.NameBinding(context.Background(), runID, ids["skippedFn"], "get")
	if err != nil {
		t.Fatal(err)
	}
	if b.ProvenOther != 0 {
		t.Errorf("ProvenOther = %d for a definition in an unread file, want 0", b.ProvenOther)
	}
}

// TestProofUnbindsAReferenceTheResolverBoundToAStranger: the resolver bound the
// reference at line 10 to the repository's `get` by name, and the compiler
// resolved that position to lodash. The binding goes, the reference is marked
// as ruled out, and the resolver's own verdict is kept for the accuracy report.
func TestProofUnbindsAReferenceTheResolverBoundToAStranger(t *testing.T) {
	ctx := context.Background()
	reg, runID, ids := proofRepo(t)
	if _, err := dbOf(reg).ExecContext(ctx, `
		UPDATE symbol_references SET resolution = 'resolved', resolved_symbol_id = ?
		 WHERE index_run_id = ? AND start_line = 10`, ids["get"], runID); err != nil {
		t.Fatal(err)
	}
	st := importProof(t, reg, runID, true)
	if st.RuledOut != 1 || st.Unbound != 1 {
		t.Fatalf("stats %+v, want the one bound reference ruled out and unbound", st)
	}
	var resolution, source, prior string
	var bound sql.NullInt64
	if err := dbOf(reg).QueryRowContext(ctx, `
		SELECT resolution, resolution_source, prior_resolution, resolved_symbol_id
		  FROM symbol_references WHERE index_run_id = ? AND start_line = 10`, runID).
		Scan(&resolution, &source, &prior, &bound); err != nil {
		t.Fatal(err)
	}
	if resolution != "unresolved" || source != registry.ResolutionSourceRuledOut || prior != "resolved" || bound.Valid {
		t.Errorf("line 10 is %q by %q (prior %q, bound %v), want unresolved, ruled out, prior resolved, unbound",
			resolution, source, prior, bound)
	}
	b, err := reg.NameBinding(ctx, runID, ids["get"], "get")
	if err != nil {
		t.Fatal(err)
	}
	if b.ProvenOther != 1 {
		t.Errorf("ProvenOther = %d, want the unbound reference counted as a use of something else", b.ProvenOther)
	}
}

func TestApplyIsRepeatable(t *testing.T) {
	reg, runID, _ := proofRepo(t)
	first := importProof(t, reg, runID, true)
	second := importProof(t, reg, runID, true)
	if first != second {
		t.Errorf("a second import changed the result: %+v then %+v", first, second)
	}
}
