package capability_test

import (
	"context"
	"strings"
	"testing"

	"github.com/satyendrakumarsingh/repocortex/internal/capability"
	"github.com/satyendrakumarsingh/repocortex/internal/capability/captest"
	"github.com/satyendrakumarsingh/repocortex/internal/registry"
)

func targetOf(t *testing.T, deps capability.Deps, runID int64, name string) capability.Target {
	t.Helper()
	tgt, err := capability.ResolveTarget(context.Background(), deps, runID, name)
	if err != nil {
		t.Fatal(err)
	}
	return tgt
}

// TestTargetBindingIsAboutTheSymbolNotTheRun: a run can bind most of its
// references and none of the ones naming the symbol asked about.
func TestTargetBindingIsAboutTheSymbolNotTheRun(t *testing.T) {
	refs := captest.Refs{Resolved: 90, Ambiguous: 2} // the run: 98% bound
	for i := 0; i < 6; i++ {
		refs.Named = append(refs.Named, captest.NamedRef{
			Name: "decode", Line: 10 + i, Resolution: registry.ResolutionAmbiguous})
	}
	refs.Named = append(refs.Named, captest.NamedRef{
		Name: "decode", Path: "lib/a.go", Line: 3, Resolution: registry.ResolutionResolved, Target: 1})

	deps, runID := captest.Build(t,
		[]captest.Sym{captest.S(1, "decode", "lib/a.go"), captest.S(2, "encode", "lib/a.go")}, nil, refs)

	b, err := deps.TargetBinding(context.Background(), runID, targetOf(t, deps, runID, "decode"))
	if err != nil {
		t.Fatal(err)
	}
	if !b.Known || b.Language != "go" || b.Name != "decode" {
		t.Fatalf("binding = %+v", b)
	}
	if b.References != 7 || b.Ambiguous != 6 || b.BoundHere != 1 || b.WithoutEdge != 1 {
		t.Errorf("counts = %+v; want 7 references, 6 ambiguous, 1 bound here with no relationship", b)
	}
	if b.Unlisted() != 7 {
		t.Errorf("Unlisted = %d, want 7", b.Unlisted())
	}
	reason, undetermined := b.Undetermined()
	if !undetermined || !strings.Contains(reason, "7 of the 7 reference(s)") ||
		!strings.Contains(reason, "6 ambiguous") || !strings.Contains(reason, "1 bound to it from outside") {
		t.Errorf("Undetermined = %q, %v", reason, undetermined)
	}
	if !b.LanguageExtracts || b.LanguageStats.Total == 0 {
		t.Errorf("language stats = %+v; Go has references in this run", b.LanguageStats)
	}

	// Its neighbour, with nothing spelled like it, is determined.
	e, err := deps.TargetBinding(context.Background(), runID, targetOf(t, deps, runID, "encode"))
	if err != nil {
		t.Fatal(err)
	}
	if reason, undetermined := e.Undetermined(); undetermined {
		t.Errorf("a symbol with no reference to its name is undetermined: %q", reason)
	}
	if e.UnlistedSentence() != "" {
		t.Errorf("UnlistedSentence = %q, want empty", e.UnlistedSentence())
	}
}

// TestALanguageWithNoReferencesIsUndetermined is PHP, Kotlin and Scala: the
// parser's tier table calls them resolved, and an index of them holds no
// reference at all.
func TestALanguageWithNoReferencesIsUndetermined(t *testing.T) {
	deps, runID := captest.Build(t,
		[]captest.Sym{captest.S(1, "emit", "okio/Sink.kt"), captest.S(2, "flush", "okio/Buffer.kt"), captest.S(3, "main", "cmd/main.go")},
		nil, captest.Refs{Resolved: 5}) // the Go file has references; the Kotlin files have none

	b, err := deps.TargetBinding(context.Background(), runID, targetOf(t, deps, runID, "emit"))
	if err != nil {
		t.Fatal(err)
	}
	if b.LanguageExtracts {
		t.Fatalf("Kotlin is reported as yielding references: %+v", b)
	}
	reason, undetermined := b.Undetermined()
	want := "RepoCortex does not extract calls from Kotlin: this index holds 2 Kotlin definition(s) and no reference from any of them"
	if !undetermined || !strings.Contains(reason, want) {
		t.Errorf("Undetermined = %q, want it to say %q", reason, want)
	}

	// The Go symbol in the same run is judged on the Go files.
	g, err := deps.TargetBinding(context.Background(), runID, targetOf(t, deps, runID, "main"))
	if err != nil {
		t.Fatal(err)
	}
	if !g.LanguageExtracts || g.LanguageStats.Resolved != 5 {
		t.Errorf("Go binding = %+v", g)
	}
}

// TestPendingReferencesAreUndeterminedAndSayToResolve: the one case
// 'repocortex resolve' fixes.
func TestPendingReferencesAreUndeterminedAndSayToResolve(t *testing.T) {
	deps, runID := captest.Build(t, []captest.Sym{captest.S(1, "run", "a.go")}, nil, captest.Refs{Pending: 9})
	b, err := deps.TargetBinding(context.Background(), runID, targetOf(t, deps, runID, "run"))
	if err != nil {
		t.Fatal(err)
	}
	reason, undetermined := b.Undetermined()
	if !undetermined || !strings.Contains(reason, "repocortex resolve") {
		t.Errorf("Undetermined = %q, %v", reason, undetermined)
	}
}

// TestLanguageFamiliesAreJudgedTogetherAndJVMLanguagesAreNot: a TypeScript
// function is called from .tsx files; a Kotlin one is not described by Java.
func TestLanguageFamiliesAreJudgedTogetherAndJVMLanguagesAreNot(t *testing.T) {
	langs := map[string]registry.LanguageBinding{
		"typescript": {Files: 2, Definitions: 10, ResolutionStats: registry.ResolutionStats{Total: 10, Resolved: 4}},
		"tsx":        {Files: 3, Definitions: 5, ResolutionStats: registry.ResolutionStats{Total: 30, Resolved: 26, Ambiguous: 4}},
		"java":       {Files: 9, Definitions: 90, ResolutionStats: registry.ResolutionStats{Total: 100, Resolved: 50, Ambiguous: 50}},
		"kotlin":     {Files: 7, Definitions: 70},
	}
	ts := capability.LanguageBindingFor(langs, "TypeScript")
	if ts.Files != 5 || ts.Definitions != 15 || ts.Total != 40 || ts.Resolved != 30 || ts.Ambiguous != 4 {
		t.Errorf("typescript family = %+v", ts)
	}
	if tsx := capability.LanguageBindingFor(langs, "tsx"); tsx.Total != ts.Total {
		t.Errorf("tsx and typescript are judged differently: %d and %d", tsx.Total, ts.Total)
	}
	k := capability.LanguageBindingFor(langs, "kotlin")
	if k.Total != 0 || k.Definitions != 70 || k.ExtractsReferences() {
		t.Errorf("kotlin = %+v; it must not inherit Java's references", k)
	}
	if missing := capability.LanguageBindingFor(langs, "zig"); missing.Files != 0 || missing.Language != "zig" {
		t.Errorf("an absent language = %+v", missing)
	}
}

func TestDisplayLanguage(t *testing.T) {
	for in, want := range map[string]string{
		"go": "Go", "c_sharp": "C#", "tsx": "TypeScript", "KOTLIN": "Kotlin", "zig": "zig", "": "",
	} {
		if got := capability.DisplayLanguage(in); got != want {
			t.Errorf("DisplayLanguage(%q) = %q, want %q", in, got, want)
		}
	}
}
