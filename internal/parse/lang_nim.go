package parse

import "strings"

// Nim. What the index holds, see fixtures/langs/nim/expected.tsv.
//
// The inferred query takes a routine only when its name is a bare identifier.
// A name followed by the export marker (`proc add*`) is an exported_symbol
// node and a backquoted operator (`proc `+`*`) is an accent_quoted node, so
// the public API of every library was missing; iterators, templates, macros,
// converters, ref and distinct types, aliases and let/var were never read. The
// inferred query is replaced.
//
//   - proc, func, iterator, template, macro and converter are functions, at
//     any depth (a proc inside a proc is a named routine); method is a method.
//     Forward declarations (a proc header with no body) count as well;
//   - in a type section, object, enum, tuple, distinct, `ref object` and
//     variant types are classes, a concept is an interface, and everything
//     else (`T = seq[int]`, `proc (x: int)`, `ptr int`, `ref Foo`) is a
//     type_alias;
//   - const NAME is a constant, let and var NAME are variables. These and type
//     sections are read at the top level of the file and in the branches of a
//     top-level `when`; inside a routine they are locals and are skipped;
//   - fields, enum members and parameters are not definitions.
//
// An exported name's `*` is not part of the name; a backquoted operator is
// named by its symbol.
func init() {
	registerDefinitions("nim", nimDefinitions(), true)
}

// nimName matches the name position of a routine, type or variable.
const nimName = `[
    (identifier) @name
    (exported_symbol . (identifier) @name)
    (accent_quoted . (identifier) @name .)
    (exported_symbol (accent_quoted . (identifier) @name .))
  ]`

func nimDefinitions() string {
	var b strings.Builder

	for _, r := range []struct{ node, kind string }{
		{"proc_declaration", "function"},
		{"func_declaration", "function"},
		{"method_declaration", "method"},
		{"iterator_declaration", "function"},
		{"template_declaration", "function"},
		{"macro_declaration", "function"},
		{"converter_declaration", "function"},
	} {
		b.WriteString("(" + r.node + " . " + nimName + ") @definition." + r.kind + "\n")
	}

	// Type, constant, let and var: the top level and the branches of `when`.
	where := []string{
		"(source_file %s)",
		"(source_file (when (statement_list %s)))",
		"(source_file (when (elif_branch (statement_list %s))))",
		"(source_file (when (else_branch (statement_list %s))))",
	}
	typeName := "(type_symbol_declaration . " + nimName + ")"
	types := []struct{ rhs, kind string }{
		{"(object_declaration)", "class"},
		{"(enum_declaration)", "class"},
		{"(distinct_type)", "class"},
		{"(ref_type (object_declaration))", "class"},
		{"(type_expression (tuple_type))", "class"},
		{"(concept_declaration)", "interface"},
		{"(pointer_type)", "type_alias"},
		{"(ref_type (type_expression))", "type_alias"},
		{"(type_expression [(identifier) (bracket_expression) (dot_expression) (proc_type) (tuple_construction) (infix_expression) (call) (prefix_expression)])", "type_alias"},
	}
	for _, w := range where {
		for _, t := range types {
			decl := "(type_section (type_declaration " + typeName + " " + t.rhs + ") @definition." + t.kind + ")"
			b.WriteString(strings.Replace(w, "%s", decl, 1) + "\n")
		}
		for _, v := range []struct{ section, kind string }{
			{"const_section", "constant"},
			{"let_section", "variable"},
			{"var_section", "variable"},
		} {
			decl := "(" + v.section + " (variable_declaration (symbol_declaration_list (symbol_declaration " + nimName + "))) @definition." + v.kind + ")"
			b.WriteString(strings.Replace(w, "%s", decl, 1) + "\n")
		}
	}
	return b.String()
}
