package mcp

import (
	"strings"
	"testing"

	"github.com/satyendrakumarsingh/repocortex/pkg/model"
)

// ---------------------------------------------------------------- D: the narrowest evidence decides

// lowCoverageJavaTargetBoundWithValueUse adds a method reference, so the target
// is named where no call is recorded: its own accounting is NOT complete.
func lowCoverageJavaTargetBoundWithValueUse(b *indexBuilder) {
	lowCoverageJavaTargetBound(b)
	b.fileText("api/Callbacks.java", "java", "class Callbacks {\n  Object cb = Service::update;\n}\n", typ("Callbacks", model.KindClass, 1, 3))
}

func TestALanguageWideRateDoesNotMakeAFullyAccountedSymbolUnreliable(t *testing.T) {
	fs := serverFor(t, lowCoverageJavaTargetBound)
	calls := callsForEveryTool("update", "svc/Service.java")

	// Every tool that holds the symbol's own accounting.
	for _, tool := range []string{"find_callers", "find_references", "analyze_impact", "find_tests", "plan_change", "find_owners", "plan_rename"} {
		text, res := answer(t, fs.sess, tool, calls[tool])
		if res.IsError {
			t.Fatalf("%s failed:\n%s", tool, text)
		}
		first := firstLine(text)
		if strings.HasPrefix(first, leadUnreliable) || strings.Contains(text, "COVERAGE TOO LOW") || strings.Contains(text, "Do not present what follows") {
			t.Errorf("%s: a symbol whose every reference is bound is unreliable because of the language's rate: %q", tool, first)
		}
	}
	callers := mustAnswer(t, fs, "find_callers", map[string]any{"name": "update"})
	if !strings.Contains(callers, "verdict: strong") {
		t.Errorf("two high callers on a fully accounted symbol lost their verdict:\n%s", callers)
	}
	if !strings.Contains(callers, "Language-wide, 20% of the") || !strings.Contains(callers, "That rate does not apply to this answer") {
		t.Errorf("the language-wide rate is not stated in a note:\n%s", callers)
	}

	// The tools that have no per-name accounting keep today's grade.
	callees := mustAnswer(t, fs, "find_callees", map[string]any{"name": "update"})
	if first := firstLine(callees); !strings.HasPrefix(first, leadUnreliable) {
		t.Errorf("find_callees has no accounting for the name and must keep the language's grade: %q", first)
	}
}

func TestTheLanguageWideRateStandsWhenTheSymbolsOwnAccountingIsNotComplete(t *testing.T) {
	// A reference spelled like the symbol that was bound to nothing.
	fs := serverFor(t, lowCoverageJava)
	for _, tool := range []string{"find_callers", "analyze_impact", "find_tests", "plan_change"} {
		args := map[string]any{"name": "update"}
		if tool != "find_callers" {
			args = map[string]any{"symbol": "update"}
		}
		if first := firstLine(mustAnswer(t, fs, tool, args)); !strings.HasPrefix(first, leadUnreliable) {
			t.Errorf("%s: an ambiguous reference spelled like the symbol must keep it UNRELIABLE: %q", tool, first)
		}
	}
	// A name used where no call is recorded: a doubt, so the rate stands.
	fs = serverFor(t, lowCoverageJavaTargetBoundWithValueUse)
	if first := firstLine(mustAnswer(t, fs, "find_callers", map[string]any{"name": "update"})); !strings.HasPrefix(first, leadUnreliable) {
		t.Errorf("a possible value use must keep the language's grade: %q", first)
	}
}
