package parse

import (
	"testing"

	"github.com/odvcencio/gotreesitter/grammars"
)

// TestFSharpMemberConstraintCallStallsTheParser pins a measurement, not a
// wish. A statically-resolved member call, `(^T: (member M: unit -> ^A) (x))`,
// puts gotreesitter's F# parser into a loop that consumes no token. The
// library ends it with its no-progress guard, which it reports as an
// iteration-limit stop, so the file is recorded as budget_exceeded.
//
// It is NOT the work limit running out: the limit is 30 iterations per byte
// (10,000 at least) and this five-line file stops after 156 of 10,000, a
// 690-line file after 3,218 of 825,480. The stopping point does not move when
// the limit is raised to fifty million (measured), so a larger per-grammar
// factor recovers nothing. See fixtures/langs/fsharp/KNOWN_GAPS.md.
//
// If a library upgrade fixes the loop this test fails; then remove the gap
// from KNOWN_GAPS.md and add the construct to the fixture.
func TestFSharpMemberConstraintCallStallsTheParser(t *testing.T) {
	entry := grammars.DetectLanguageByName("fsharp")
	if entry == nil {
		t.Skip("no F# grammar")
	}
	p := newGenericParser(*entry, defaultParserConfig)
	if p.lang == nil {
		t.Skip("F# grammar did not load")
	}
	src := []byte("module M\n\nlet f task =\n    let a = (^T: (member GetAwaiter: unit -> ^A) (task))\n    a\n\nlet g x = x\n")
	_, _, stop, err := p.parseTree(t.Context(), src)
	if err != nil {
		t.Fatal(err)
	}
	if !stop.early {
		t.Fatal("the parser now reads a member constraint call to the end; remove the gap from fixtures/langs/fsharp/KNOWN_GAPS.md")
	}
	if stop.iterations*10 >= stop.iterLimit {
		t.Fatalf("stopped after %d of %d iterations: that is the limit running out, not a stall", stop.iterations, stop.iterLimit)
	}
}
