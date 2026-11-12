package parse

import (
	ts "github.com/odvcencio/gotreesitter"

	"github.com/satyendrakumarsingh/repocortex/pkg/model"
)

// Clause merging.
//
// In Elixir and Erlang a function is written as a sequence of clauses, one per
// pattern: `def f(0)` then `def f(n)`. A definition query can only match one
// clause head at a time, so a function of five clauses arrived as five
// symbols of one name and every lookup by name returned duplicates.
//
// A language opts in with registerClauseMerge. The parser then folds
// CONSECUTIVE definitions into one symbol when they agree on name, kind, owner
// and arity. The result starts where the first clause starts and ends where the
// last one ends, so a call written in any clause is still attributed to the
// function.
//
// Two things keep the fold from inventing a function:
//
//   - consecutive means no other symbol of the same owner lies between the
//     two. `f`, `g`, `f` is two definitions of f (the compiler warns about it
//     in both languages), and they stay two symbols;
//   - arity is part of the identity where the language has one. `def f(a)` and
//     `def f(a, b)` are different functions. The rule counts the head's
//     arguments as written, so `def f(a, b \\ 1)` is f/2 and `def f(a)` after
//     it is a separate f/1, which is how the language numbers them when the
//     default is expanded.

// clauseArity returns the arity of the clause head whose function name is the
// given node, as a string comparable between clauses, or "" when the head does
// not say.
type clauseArity func(name *ts.Node, lang *ts.Language) string

var clauseMerges = map[string]clauseArity{}

// registerClauseMerge makes a language's consecutive same-named definitions one
// symbol. Call it from init in the language's lang_<language>.go. arity may be
// nil for a language with no arity in its function heads, in which case
// clauses merge by name alone.
func registerClauseMerge(language string, arity clauseArity) {
	if _, dup := clauseMerges[language]; dup {
		duplicateRegistrations = append(duplicateRegistrations, language+" (clause merge)")
	}
	if arity == nil {
		arity = func(*ts.Node, *ts.Language) string { return "" }
	}
	clauseMerges[language] = arity
}

// symbolAt identifies a symbol before merging: where its range starts, and its
// name. The pair is what dedupeSymbols already treats as one declaration.
type symbolAt struct {
	line, col int
	name      string
}

// clauseArities reads each definition's arity from the tree, for a language
// that merges clauses; nil for every other language.
func (p *genericParser) clauseArities(tree *ts.Tree, tags []ts.Tag) map[symbolAt]string {
	arity, ok := clauseMerges[p.entry.Name]
	if !ok || tree == nil || p.lang == nil {
		return nil
	}
	root := tree.RootNode()
	if root == nil {
		return nil
	}
	out := make(map[symbolAt]string, len(tags))
	for _, tag := range tags {
		if len(tag.Kind) < len("definition") || tag.Kind[:len("definition")] != "definition" {
			continue
		}
		name := root.NamedDescendantForByteRange(tag.NameRange.StartByte, tag.NameRange.EndByte)
		if name == nil {
			continue
		}
		r := toRange(tag.Range)
		out[symbolAt{r.Start.Line, r.Start.Col, tag.Name}] = arity(name, p.lang)
	}
	return out
}

// isClauseKind is the kinds a function can be written as. A repeated module
// attribute or type is a redefinition, not a further clause.
func isClauseKind(k model.EntityKind) bool {
	return k == model.KindFunction || k == model.KindMethod
}

// mergeClauses folds consecutive clauses of one function into one symbol. The
// input is sorted by position, as dedupeSymbols leaves it.
func mergeClauses(in []model.Symbol, arities map[symbolAt]string) []model.Symbol {
	if len(in) < 2 || arities == nil {
		return in
	}
	type key struct {
		name  string
		kind  model.EntityKind
		arity string
	}
	keyOf := func(s model.Symbol) key {
		return key{s.Name, s.Kind, arities[symbolAt{s.Range.Start.Line, s.Range.Start.Col, s.Name}]}
	}
	before := func(a, b model.Position) bool {
		return a.Line < b.Line || (a.Line == b.Line && a.Col < b.Col)
	}

	out := make([]model.Symbol, 0, len(in))
	// open holds the indexes of the output symbols whose range still encloses
	// the current position; the last is the owner. lastChild is the most recent
	// output symbol of each owner (-1 for none, keyed by owner index; the file
	// itself is owner -1).
	var open []int
	lastChild := map[int]int{}
	for _, s := range in {
		for len(open) > 0 && !before(s.Range.Start, out[open[len(open)-1]].Range.End) {
			open = open[:len(open)-1]
		}
		owner := -1
		if len(open) > 0 {
			owner = open[len(open)-1]
		}
		if i, ok := lastChild[owner]; ok && isClauseKind(s.Kind) && keyOf(out[i]) == keyOf(s) &&
			!before(s.Range.Start, out[i].Range.End) {
			if before(out[i].Range.End, s.Range.End) {
				out[i].Range.End = s.Range.End
			}
			// The merged symbol covers s, so what s contains is inside it.
			open = append(open, i)
			continue
		}
		out = append(out, s)
		lastChild[owner] = len(out) - 1
		open = append(open, len(out)-1)
	}
	return out
}
