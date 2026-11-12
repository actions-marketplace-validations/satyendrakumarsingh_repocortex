package parse

import (
	"sort"
	"strings"

	"github.com/odvcencio/gotreesitter/grammars"
)

// ExtractsDefinitions reports whether this build can produce a definition for
// a language: its grammar has a definition query, or it is a single-file
// component read through its script block.
//
// A bundled grammar is not the same thing. Most of the grammars the backend
// carries have no definition query, and a file in one of those languages is
// skipped. TierFor reports TierParsed for them as well, because a tier says
// how far resolution goes once definitions exist, not whether extraction
// starts. Anything that describes language support to a person has to ask
// this question too; a count of grammars answers a different one.
func ExtractsDefinitions(language string) bool {
	if isSFC(language) {
		return true
	}
	entry := grammars.DetectLanguageByName(language)
	if entry == nil {
		return false
	}
	return definitionQueryText(*entry) != ""
}

// definitionQueryText is the query a parser for this grammar would be built
// from, before the inheritance patterns are appended. Empty means the parser
// cannot be built and every file of the language is skipped.
func definitionQueryText(entry grammars.LangEntry) string {
	query := grammars.ResolveTagsQuery(entry)
	if sup := Supplement(entry.Name); sup != "" {
		query = query + "\n" + sup
	}
	return strings.TrimSpace(query)
}

// GrammarOnlyLanguages returns the languages whose grammar is bundled but for
// which nothing is extracted, sorted.
func GrammarOnlyLanguages() []string {
	var out []string
	for _, l := range PureGoBackend().Languages() {
		if !ExtractsDefinitions(l) {
			out = append(out, l)
		}
	}
	sort.Strings(out)
	return out
}
