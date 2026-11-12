package parse

import (
	"sort"
	"strings"

	"github.com/odvcencio/gotreesitter/grammars"
)

// A language's definition query can come from three places, tried in order.
//
//  1. The query gotreesitter infers for the grammar. It is built from a fixed
//     list of common node names (function_definition, class_declaration and
//     the like), so it finds plain functions and classes in a grammar that
//     happens to use those names and nothing a language spells its own way.
//     For many grammars it is empty, or it matches nothing.
//  2. A supplement in the supplements map, added to the inferred query.
//  3. A query registered here from a lang_<name>.go file, written by hand
//     against the grammar's real node types. It is added to the above, or
//     replaces the inferred query when the inferred one is wrong rather than
//     merely incomplete (it over-captures, or names nodes the grammar lacks).
//
// The third exists so that each language is one file plus one fixture. A
// language verified by hand should be changeable without touching a map that
// every other language shares.
type registeredQuery struct {
	query           string
	replaceInferred bool
}

var (
	registeredDefinitions = map[string]registeredQuery{}

	// duplicateRegistrations is checked by a test instead of panicking at
	// init: two files claiming one language is a mistake in the source tree,
	// and a panic at start-up would take every other language down with it.
	duplicateRegistrations []string
)

// registerDefinitions sets the hand-written definition query for a language.
// Call it from an init function in lang_<language>.go.
//
// With replaceInferred the inferred query is discarded and only this one (and
// any supplement) is used. Without it, this query is appended.
//
// The query uses tree-sitter tag captures: @name for the identifier and
// @definition.KIND for the declaring node, where KIND is one kindFromTag
// understands (function, method, constructor, class, struct, enum, interface,
// trait, type_alias, module, namespace, constant, variable, field).
// References are @reference.KIND with the same @name.
func registerDefinitions(language, query string, replaceInferred bool) {
	if _, dup := registeredDefinitions[language]; dup {
		duplicateRegistrations = append(duplicateRegistrations, language)
		return
	}
	registeredDefinitions[language] = registeredQuery{query: query, replaceInferred: replaceInferred}
}

// HandWrittenLanguages lists the languages whose definition query was written
// by hand and registered, sorted. Each must have a conformance fixture.
func HandWrittenLanguages() []string {
	out := make([]string, 0, len(registeredDefinitions))
	for l := range registeredDefinitions {
		out = append(out, l)
	}
	sort.Strings(out)
	return out
}

// definitionQueryText is the query a parser for this grammar is built from,
// before the inheritance patterns are appended. Empty means no parser can be
// built and every file of the language is skipped.
func definitionQueryText(entry grammars.LangEntry) string {
	reg, registered := registeredDefinitions[entry.Name]

	var parts []string
	if !registered || !reg.replaceInferred {
		if inferred := grammars.ResolveTagsQuery(entry); strings.TrimSpace(inferred) != "" {
			parts = append(parts, inferred)
		}
	}
	if sup := Supplement(entry.Name); sup != "" {
		parts = append(parts, sup)
	}
	if registered && strings.TrimSpace(reg.query) != "" {
		parts = append(parts, reg.query)
	}
	return strings.TrimSpace(strings.Join(parts, "\n"))
}
