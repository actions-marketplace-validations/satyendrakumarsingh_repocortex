package parse

import "strings"

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
// A function is recorded once per clause head: `def f(0)` and `def f(n)` are
// two symbols, as are the bodiless head `def f(a, b \\ 1)` and the clauses
// that follow it. The grammar cannot tell the first clause from the others,
// and the name alone (without the arity) is what a query can capture, so
// collapsing runs of clauses is not possible here. A clause with a guard
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
; @type t :: ..., @typep t(a) :: ..., @opaque t :: ...
(unary_operator (call (identifier) @_a (arguments . (binary_operator . (call (identifier) @name))) (#any-of? @_a "type" "typep" "opaque"))) @definition.type_alias
(unary_operator (call (identifier) @_a (arguments . (binary_operator . (identifier) @name)) (#any-of? @_a "type" "typep" "opaque"))) @definition.type_alias

; @callback f(a) :: ..., @macrocallback f(a) :: ...
(unary_operator (call (identifier) @_a (arguments . (binary_operator . (call (identifier) @name))) (#any-of? @_a "callback" "macrocallback"))) @definition.method

; @name value
(do_block (unary_operator (call (identifier) @name (arguments (_)))) @definition.constant
  (#match? @definition.constant "^@")
  (#not-any-of? @name "doc" "moduledoc" "typedoc" "spec" "type" "typep" "opaque" "callback" "macrocallback" "impl" "behaviour" "derive" "enforce_keys" "deprecated" "since" "dialyzer" "compile" "on_definition" "before_compile" "after_compile" "on_load" "external_resource" "file" "vsn" "optional_callbacks" "nifs"))
`)
	registerDefinitions("elixir", q.String(), true)
}
