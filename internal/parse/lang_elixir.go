package parse

import (
	"strconv"
	"strings"

	ts "github.com/odvcencio/gotreesitter"
)

// Elixir: modules, protocols and their implementations, functions and macros,
// types, callbacks and module attributes.
//
// Everything in Elixir is a `call` whose target is an identifier, so each
// pattern names the macro: `defmodule`, `def`, `defp`, `defmacro`,
// `defmacrop`, `defguard`, `defguardp`, `defdelegate`, `defprotocol`,
// `defimpl`.
//
// Kinds:
//   - defmodule is a `package`; defprotocol an `interface`; defimpl a `class`
//     named by the protocol it implements (the implementation module has no
//     name of its own that a node carries).
//   - def, defp, defmacro, defmacrop, defguard, defguardp and defdelegate are
//     `function`; inside a defprotocol they are `method`.
//   - @type, @typep and @opaque are `type_alias`; @callback and @macrocallback
//     are `method`; every other module attribute that is given a value, except
//     the ones the compiler reads (@doc, @spec, @impl, @behaviour ...), is a
//     `constant`.
//
// The query matches one clause head at a time, so a function written as
// several clauses arrives as several symbols; registerClauseMerge folds the
// consecutive clauses of one name and arity back into one (see
// clausemerge.go). `def f(a)` and `def f(a, b)` stay two functions, as does a
// later `def f` that follows another function. A clause with a guard
// (`def f(n) when is_integer(n)`) or without parentheses (`def f, do: 1`) is
// found like any other.
//
// Not recorded: defstruct and defexception (they declare no name; the struct
// is the module), @spec, imports, and the generated functions of `test`
// blocks.
func init() {
	const defs = `"def" "defp" "defmacro" "defmacrop" "defguard" "defguardp" "defdelegate"`
	// The ways a clause head can be written, as the first argument of def.
	forms := []string{
		`(call (identifier) @name)`,                     // def f(a), do: ...
		`(identifier) @name`,                            // def f, do: ...
		`(binary_operator . (call (identifier) @name))`, // def f(a) when guard
		`(binary_operator . (identifier) @name)`,        // def f when guard
	}

	var q strings.Builder
	q.WriteString(`
(call (identifier) @name) @reference.call

(call (identifier) @_head (arguments . (alias) @name) (#eq? @_head "defmodule")) @definition.module
(call (identifier) @_head (arguments . (alias) @name) (#eq? @_head "defprotocol")) @definition.interface
(call (identifier) @_head (arguments . (alias) @name) (#eq? @_head "defimpl")) @definition.class
`)
	for _, f := range forms {
		q.WriteString("(call (identifier) @_d (arguments . " + f + ") (#any-of? @_d " + defs + ")) @definition.function\n")
		q.WriteString("(call (identifier) @_p (do_block (call (identifier) @_d (arguments . " + f + ") (#any-of? @_d " + defs + ")) @definition.method) (#eq? @_p \"defprotocol\"))\n")
	}
	q.WriteString(`
; @type t :: ..., @typep t(a) :: ..., @opaque t :: ..., also with a when clause
(unary_operator (call (identifier) @_a (arguments . (binary_operator . (call (identifier) @name))) (#any-of? @_a "type" "typep" "opaque"))) @definition.type_alias
(unary_operator (call (identifier) @_a (arguments . (binary_operator . (identifier) @name)) (#any-of? @_a "type" "typep" "opaque"))) @definition.type_alias
(unary_operator (call (identifier) @_a (arguments . (binary_operator . (binary_operator . (call (identifier) @name)))) (#any-of? @_a "type" "typep" "opaque"))) @definition.type_alias
(unary_operator (call (identifier) @_a (arguments . (binary_operator . (binary_operator . (identifier) @name))) (#any-of? @_a "type" "typep" "opaque"))) @definition.type_alias

; @callback f(a) :: ..., @macrocallback f(a) :: ..., also with no parentheses (f :: ...)
; and with a when clause after the return type (f(a) :: r when a: term)
(unary_operator (call (identifier) @_a (arguments . (binary_operator . (call (identifier) @name))) (#any-of? @_a "callback" "macrocallback"))) @definition.method
(unary_operator (call (identifier) @_a (arguments . (binary_operator . (identifier) @name)) (#any-of? @_a "callback" "macrocallback"))) @definition.method
(unary_operator (call (identifier) @_a (arguments . (binary_operator . (binary_operator . (call (identifier) @name)))) (#any-of? @_a "callback" "macrocallback"))) @definition.method
(unary_operator (call (identifier) @_a (arguments . (binary_operator . (binary_operator . (identifier) @name))) (#any-of? @_a "callback" "macrocallback"))) @definition.method

; @name value
(do_block (unary_operator (call (identifier) @name (arguments (_)))) @definition.constant
  (#match? @definition.constant "^@")
  (#not-any-of? @name "doc" "moduledoc" "typedoc" "spec" "type" "typep" "opaque" "callback" "macrocallback" "impl" "behaviour" "derive" "enforce_keys" "deprecated" "since" "dialyzer" "compile" "on_definition" "before_compile" "after_compile" "on_load" "external_resource" "file" "vsn" "optional_callbacks" "nifs"))
`)
	registerDefinitions("elixir", q.String(), true)
	registerClauseMerge("elixir", elixirClauseArity)
}

// elixirClauseArity counts the arguments of the clause head a name sits in:
// the named children of the `arguments` of `f(a, b)`. A head with no
// parentheses (`def f, do: 1`) has none.
func elixirClauseArity(name *ts.Node, lang *ts.Language) string {
	head := name.Parent()
	if head == nil || head.Type(lang) != "call" {
		return "0"
	}
	return namedChildCount(head, "arguments", lang)
}

// namedChildCount is how many named, non-comment children the first child of
// the given type has, as a string; "" when the node has no such child.
func namedChildCount(n *ts.Node, childType string, lang *ts.Language) string {
	for i := 0; i < n.ChildCount(); i++ {
		c := n.Child(i)
		if c == nil || c.Type(lang) != childType {
			continue
		}
		total := 0
		for j := 0; j < c.ChildCount(); j++ {
			if a := c.Child(j); a != nil && a.IsNamed() && !a.IsExtra() {
				total++
			}
		}
		return strconv.Itoa(total)
	}
	return ""
}
