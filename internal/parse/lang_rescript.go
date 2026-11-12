package parse

import "strings"

// ReScript: lets (functions and values), types, modules, module types,
// externals and exceptions, at the top of the file and in module bodies.
//
// A let is a function when its value is a `function` node, or when it is a
// signature (`let f: a => b`, in a module type or an interface file) whose
// type is a function type. Every other let is a variable. The two sets are
// written as disjoint patterns on purpose: when a declaration is tagged both
// ways the kind ranked more specific wins, and `function` ranks lowest, so a
// catch-all variable pattern would turn every function into a variable.
//
// Local lets inside a function body are not definitions, so every pattern is
// anchored to the source file or to the block that is a module's body.
func init() {
	scopes := []string{
		"(source_file %s)",
		"(module_binding (block %s))",
	}
	// Node types a non-function let value can have. A value of any other
	// type is missed rather than guessed at.
	values := "[(number) (string) (template_string) (list) (array) (record) (tuple) (unit) (call_expression) (value_identifier) (member_expression) (binary_expression) (variant) (true) (false) (if_expression) (switch_expression) (pipe_expression) (parenthesized_expression) (ternary_expression) (unary_expression) (block) (polyvar) (character)]"
	plainTypes := "[(type_identifier) (generic_type) (tuple_type) (unit_type) (module_identifier_path) (polyvar_type)]"

	decls := []string{
		// let
		"(let_declaration (let_binding . (value_identifier) @name (function)) @definition.function)",
		"(let_declaration (let_binding . (value_identifier) @name (type_annotation (function_type))) @definition.function)",
		"(let_declaration (let_binding . (value_identifier) @name " + values + ") @definition.variable)",
		"(let_declaration (let_binding . (value_identifier) @name (type_annotation " + plainTypes + ") .) @definition.variable)",
		// type
		"(type_declaration (type_binding . (type_identifier) @name [(variant_type) (record_type)]) @definition.class)",
		"(type_declaration (type_binding . (type_identifier) @name .) @definition.class)",
		"(type_declaration (type_binding . (type_identifier) @name (type_parameters) .) @definition.class)",
		"(type_declaration (type_binding . (type_identifier) @name [(type_identifier) (generic_type) (tuple_type) (function_type) (unit_type) (polyvar_type)]) @definition.type_alias)",
		"(type_declaration (type_binding . (type_identifier) @name (type_parameters) [(type_identifier) (generic_type) (tuple_type) (function_type) (unit_type) (polyvar_type)]) @definition.type_alias)",
		// module, module type, functor, alias
		"(module_declaration (module_binding . (module_identifier) @name) @definition.module)",
		// external and exception
		"(external_declaration . (value_identifier) @name) @definition.function",
		"(exception_declaration . (variant_identifier) @name) @definition.class",
	}
	var b strings.Builder
	for _, scope := range scopes {
		for _, d := range decls {
			b.WriteString(strings.Replace(scope, "%s", d, 1))
			b.WriteByte('\n')
		}
	}
	registerDefinitions("rescript", b.String(), true)
}
