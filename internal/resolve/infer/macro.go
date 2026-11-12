package infer

import (
	"strings"

	"github.com/satyendrakumarsingh/repocortex/internal/registry"
	"github.com/satyendrakumarsingh/repocortex/pkg/model"
)

// # Macro invocations are not function calls
//
// `format!(...)`, `vec![...]` and `println!(...)` invoke a macro. A macro
// lives in its own namespace, and a function of the same name is not what the
// invocation reaches. fd bound std's `format!` to a test function named
// `format` at `high`, twice in thirty.
//
// What the parser must emit for this rule to act (it does not yet):
//
//   - a reference of kind `reference.call.macro` for each invocation, written
//     `name!(...)`, `name![...]` or `name!{...}`, with the name WITHOUT the
//     bang and a path qualifier recorded as the receiver as for any call. Today
//     it emits `reference.call` with receiver kind `bare`, which cannot be told
//     from a call of a function.
//   - a symbol of kind `macro` for each `macro_rules! name { ... }` definition
//     (and `macro name` declarative macros 2.0), qualified like a function at
//     the module level it is written in, so that isModuleLevel holds.
//
// Until then coreSyntax (corefunctions.go) keeps the standard macros from
// binding to anything, and this rule is inert: no symbol has the kind and no
// reference has the suffix.

const (
	// macroRefSuffix ends the kind of a reference to a macro invocation.
	macroRefSuffix = ".macro"
	// macroSymbolKind is the kind of a symbol that defines a macro.
	macroSymbolKind = "macro"
)

// isMacroInvocation reports whether a reference is a macro invocation.
func isMacroInvocation(refKind string) bool { return strings.HasSuffix(refKind, macroRefSuffix) }

// withoutMacros drops macro definitions from the candidates of a call that is
// not a macro invocation: a function call never reaches one.
func withoutMacros(cands []registry.SymbolIndexEntry) []registry.SymbolIndexEntry {
	has := false
	for _, s := range cands {
		if s.Kind == macroSymbolKind {
			has = true
			break
		}
	}
	if !has {
		return cands
	}
	kept := make([]registry.SymbolIndexEntry, 0, len(cands))
	for _, s := range cands {
		if s.Kind != macroSymbolKind {
			kept = append(kept, s)
		}
	}
	return kept
}

// callMacro binds a macro invocation to a macro definition, and to nothing
// else. In the file that defines it the macro is in scope from its definition
// on; elsewhere `#[macro_use]` and `#[macro_export]` make it crate-wide, which
// the index does not record, so the one macro of that name in the repository is
// a weaker claim.
func (r *Resolver) callMacro(c call) decision {
	var macros, here []registry.SymbolIndexEntry
	for _, s := range c.cands {
		if s.Kind != macroSymbolKind || !isModuleLevel(s) {
			continue
		}
		macros = append(macros, s)
		if s.FileID == c.fr.FileID {
			here = append(here, s)
		}
	}
	if first, ok := firstDeclaration(here); ok {
		return bound(candidate{first, model.ConfidenceHigh, "macro defined in the same file"})
	}
	if len(here) > 1 {
		return ambiguousCall
	}
	if len(macros) == 0 {
		if c.spec.isCoreSyntax(c.ref.Name) || c.spec.isCoreFunction(c.ref.Name) {
			return externalCall
		}
		return unresolvedCall
	}
	if first, ok := firstDeclaration(macros); ok && !r.callsTestHelper(c, first) {
		return bound(candidate{first, model.ConfidenceMedium, "the only macro of this name in the repository"})
	}
	if len(macros) > 1 {
		return ambiguousCall
	}
	return unresolvedCall
}
