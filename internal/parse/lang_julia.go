package parse

import "strings"

// Julia. What the index holds, see fixtures/langs/julia/expected.tsv.
//
// The inferred query takes only `function f(args)` with a bare name. It drops
// a qualified name (`function Base.get(...)`), a constructor
// (`Dict{K,V}(...)`), a `where` clause, a return type, the short form
// `f(x) = ...`, and every type that is not a plain struct, so it is replaced.
//
//   - module and baremodule are packages;
//   - struct, mutable struct, abstract type and primitive type are classes,
//     named without their type parameters or supertype; `@enum Color a b`
//     is a class named Color (its members are not extracted);
//   - function f(...), function Base.get(...), function Base.:+(...),
//     function Dict{K,V}(...), each with or without `where` and a return type,
//     and `function f end`, are functions, at any depth;
//   - a macro is a function named without the @;
//   - the short form `f(x) = ...` (same name shapes, `+(a, b) = ...`
//     included, also behind a macro such as @inline) is a function when it
//     stands at the top level of a file, a module or a struct, or in the
//     branches of an `if` (or `@static if`) there. Inside a function body it is a local closure and is not extracted. An inner
//     constructor in a struct is such a method;
//   - const NAME = ... (also `const global`) is a constant; const NAME{T} = ... and const NAME =
//     Union{...} (a right-hand side that is a parametrised type) are
//     type_alias;
//   - plain assignments, global declarations and `(x::T)(args) = ...`
//     functors are not definitions.
func init() {
	registerDefinitions("julia", juliaDefinitions(), true)
}

// juliaCallName matches the call-shaped left side of a method definition and
// captures its name.
const juliaCallName = `(call_expression . [
    (identifier) @name
    (operator) @name
    (field_expression (_) . (identifier) @name .)
    (field_expression (quote_expression (operator) @name))
    (parametrized_type_expression . (identifier) @name)
  ])`

func juliaDefinitions() string {
	// A method's left side is the call, optionally with a return type or a
	// where clause around it, in either order.
	shapes := []string{
		juliaCallName,
		"(typed_expression " + juliaCallName + ")",
		"(where_expression " + juliaCallName + ")",
		"(where_expression (typed_expression " + juliaCallName + "))",
		"(typed_expression (where_expression " + juliaCallName + "))",
	}

	var b strings.Builder
	b.WriteString(`
(module_definition . (identifier) @name) @definition.module

(struct_definition (type_head . (identifier) @name)) @definition.class
(struct_definition (type_head . (parametrized_type_expression . (identifier) @name))) @definition.class
(struct_definition (type_head . (binary_expression . (identifier) @name))) @definition.class
(struct_definition (type_head . (binary_expression . (parametrized_type_expression . (identifier) @name)))) @definition.class
(abstract_definition (type_head . (identifier) @name)) @definition.class
(abstract_definition (type_head . (parametrized_type_expression . (identifier) @name))) @definition.class
(abstract_definition (type_head . (binary_expression . (identifier) @name))) @definition.class
(abstract_definition (type_head . (binary_expression . (parametrized_type_expression . (identifier) @name)))) @definition.class
(primitive_definition (type_head . (identifier) @name)) @definition.class
(primitive_definition (type_head . (parametrized_type_expression . (identifier) @name))) @definition.class
(primitive_definition (type_head . (binary_expression . (identifier) @name))) @definition.class

(macrocall_expression
  (macro_identifier (identifier) @kw)
  (macro_argument_list . [(identifier) @name (typed_expression . (identifier) @name)])
  (#eq? @kw "enum")) @definition.class

(const_statement (assignment . (identifier) @name)) @definition.constant
(const_statement (global_statement (assignment . (identifier) @name))) @definition.constant
(const_statement (assignment . (typed_expression . (identifier) @name))) @definition.constant
(const_statement (assignment . (open_tuple (identifier) @name))) @definition.constant
(const_statement (assignment . (parametrized_type_expression . (identifier) @name))) @definition.type_alias
(const_statement (assignment . (identifier) @name . (operator) . (parametrized_type_expression))) @definition.type_alias

(function_definition (signature . (identifier) @name)) @definition.function
(function_definition (signature . (field_expression (_) . (identifier) @name .))) @definition.function
(macro_definition (signature . (call_expression . (identifier) @name))) @definition.function
(macro_definition (signature . (identifier) @name)) @definition.function
`)

	for _, s := range shapes {
		b.WriteString("(function_definition (signature " + s + ")) @definition.function\n")
	}

	// Short form: at the top of a file, a module or a struct, behind a macro
	// such as @inline, and inside the branches of a top-level `if` (also
	// behind @static), which is how version-dependent methods are written.
	inner := []string{
		"%s",
		"(if_statement (block %s))",
		"(if_statement (elseif_clause (block %s)))",
		"(if_statement (else_clause (block %s)))",
	}
	outer := []string{
		"(source_file %s)",
		"(module_definition (block %s))",
		"(struct_definition (block %s))",
		"(source_file (macrocall_expression (macro_argument_list %s)))",
		"(module_definition (block (macrocall_expression (macro_argument_list %s))))",
	}
	for _, s := range shapes {
		lhs := "(assignment . " + s + " . (operator)) @definition.function"
		for _, in := range inner {
			mid := strings.Replace(in, "%s", lhs, 1)
			for _, out := range outer {
				if in != "%s" && strings.HasPrefix(out, "(struct_definition") {
					continue
				}
				b.WriteString(strings.Replace(out, "%s", mid, 1) + "\n")
			}
		}
	}
	return b.String()
}
