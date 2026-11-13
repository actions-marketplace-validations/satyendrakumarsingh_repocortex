package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/satyendrakumarsingh/repocortex/internal/registry"
)

// An answer about one of several definitions says so before the answer, and
// does not state a count it did not read: the lookup stops at
// targetLookupLimit, so a name that fills it has "at least" that many.
func TestAnAmbiguousNameIsAnnouncedBeforeTheAnswer(t *testing.T) {
	hit := func(path string, line int) registry.SymbolHit {
		return registry.SymbolHit{Name: "dispose", Kind: "method", FilePath: path, StartLine: line}
	}

	var one bytes.Buffer
	announceAmbiguity(&one, []registry.SymbolHit{hit("a.ts", 1)}, "dispose", hit("a.ts", 1))
	if one.Len() != 0 {
		t.Errorf("a unique name was announced as ambiguous: %q", one.String())
	}

	var two bytes.Buffer
	announceAmbiguity(&two, []registry.SymbolHit{hit("a.ts", 1), hit("b.ts", 9)}, "dispose", hit("a.ts", 1))
	if got := two.String(); !strings.HasPrefix(got, "AMBIGUOUS — 2 definitions share the name \"dispose\".") ||
		!strings.Contains(got, "a.ts:1") || !strings.Contains(got, "--path") {
		t.Errorf("announcement = %q", got)
	}

	full := make([]registry.SymbolHit, targetLookupLimit)
	for i := range full {
		full[i] = hit("a.ts", i+1)
	}
	var many bytes.Buffer
	announceAmbiguity(&many, full, "dispose", full[0])
	if got := many.String(); !strings.HasPrefix(got, "AMBIGUOUS — At least 50 definitions") {
		t.Errorf("a full lookup must not be stated as an exact count: %q", got)
	}

	var end bytes.Buffer
	reportAmbiguity(&end, full, "dispose")
	if !strings.Contains(end.String(), "At least 50 definitions share the name") {
		t.Errorf("closing list = %q", end.String()[:120])
	}
}
