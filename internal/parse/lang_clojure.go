package parse

import "strings"

// Clojure. What the index holds, see fixtures/langs/clojure/expected.tsv.
//
// The inferred query for this grammar matches a form anywhere in the file, so
// a `(def x ...)` inside a function body, inside `(comment ...)` or inside a
// `let` was indexed as a definition, and it knows only def, defn, defn- and
// defmacro. It is replaced.
//
// A definition is a defining form at the top level of the file, or directly
// inside a top-level reader conditional (`#?(:clj (defn f ...))`):
//
//   - ns is a package;
//   - defn, defn- and defmacro are functions;
//   - def and defonce are variables;
//   - defmulti is a function, and each defmethod is a method named by the
//     multimethod it extends (the dispatch value is not part of the name, so
//     the several methods of one multimethod share a name and differ by line);
//   - defprotocol and definterface are interfaces, and each method signature
//     inside them is a method;
//   - defrecord and deftype are classes.
//
// Metadata on the name (`^:private`, `^String`, `^{:doc "..."}`) is skipped by
// the grammar. Not definitions: forms nested in another form (a `def` inside a
// `defn`, `let`, `do`, `when` or `comment`), protocol implementations inside
// defrecord, deftype, extend-protocol, extend-type or reify, deftest, and the
// bindings of let.
func init() {
	registerDefinitions("clojure", clojureDefinitions(), true)
}

// clojureForm is one defining form: the heads that introduce it and the tag
// kind it is recorded as.
type clojureForm struct {
	heads string
	kind  string
}

var clojureForms = []clojureForm{
	{`"defn" "defn-" "defmacro" "defmulti"`, "function"},
	{`"def" "defonce"`, "variable"},
	{`"defmethod"`, "method"},
	{`"defprotocol" "definterface"`, "interface"},
	{`"defrecord" "deftype"`, "class"},
	{`"ns"`, "package"},
}

func clojureDefinitions() string {
	// Each pattern is written once against LIST and placed under both a
	// top-level form and a top-level reader conditional, and again for a form
	// that opens with metadata of its own (`^:clj (ns x)`), where the grammar
	// puts the metadata inside the list before the head.
	var patterns []string
	for _, lead := range []string{". ", ". (meta_lit) . "} {
		for _, f := range clojureForms {
			patterns = append(patterns,
				`(list_lit `+lead+`(sym_lit (sym_name) @_head) . (sym_lit (sym_name) @name) (#any-of? @_head `+f.heads+`)) @definition.`+f.kind)
		}
		// A method signature inside a protocol or interface: `(perimeter [this])`.
		patterns = append(patterns,
			`(list_lit `+lead+`(sym_lit (sym_name) @_head)
  (list_lit . (sym_lit (sym_name) @name) . (vec_lit)) @definition.method
  (#any-of? @_head "defprotocol" "definterface"))`)
	}

	var out []string
	for _, p := range patterns {
		out = append(out, "(source "+p+")")
		out = append(out, "(source (read_cond_lit "+p+"))")
	}
	return strings.Join(out, "\n")
}
