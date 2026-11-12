package parse

import (
	"testing"

	"github.com/odvcencio/gotreesitter/grammars"
)

// ExtractsDefinitions must agree with what building the parser actually does:
// a language it calls extractable has a parser, and one it does not has none.
// If these drift, `repocortex languages` describes a different binary.
func TestExtractsDefinitionsMatchesTheParser(t *testing.T) {
	for _, e := range grammars.AllLanguages() {
		if isSFC(e.Name) {
			if !ExtractsDefinitions(e.Name) {
				t.Errorf("%s is read through its script block and should count as extractable", e.Name)
			}
			continue
		}
		built := newGenericParser(e).initErr == nil
		if got := ExtractsDefinitions(e.Name); got != built {
			t.Errorf("%s: ExtractsDefinitions = %v, but a parser can be built = %v", e.Name, got, built)
		}
	}
}

func TestGrammarOnlyLanguagesAreNotExtractable(t *testing.T) {
	only := GrammarOnlyLanguages()
	if len(only) == 0 {
		t.Skip("every bundled grammar has a definition query")
	}
	for _, l := range only {
		if ExtractsDefinitions(l) {
			t.Errorf("%s is listed as grammar-only and as extractable", l)
		}
	}
	// A language people index every day must never be in this list unnoticed.
	for _, must := range []string{"go", "java", "python", "typescript", "javascript", "c", "cpp", "rust"} {
		if !ExtractsDefinitions(must) {
			t.Errorf("%s has no definition query", must)
		}
	}
}
