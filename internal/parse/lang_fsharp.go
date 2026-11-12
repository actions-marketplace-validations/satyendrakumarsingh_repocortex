package parse

import "strings"

// F#: modules, namespaces, types, exceptions, module-level lets and members.
//
// The grammar does not separate a function from a value in the tree the way
// the source does:
//
//   - `let f (x: int) = ...` is a function_declaration_left;
//   - `let f x = ...` is a value_declaration_left whose identifier_pattern holds
//     the name AND the argument patterns, so a pattern with more than one child
//     is a function and one with a single child is a value;
//   - `let f = fun x -> ...` is a single-child pattern with a fun_expression.
//
// A function ranks below a variable when a declaration is tagged both ways, so
// the variable pattern excludes lambda bodies by text rather than relying on
// ranking. A return type written after the parameters (`member _.F(x) : T =`)
// wraps them in a typed_pattern, and type parameters (`member _.F<'T>() =`)
// put a type_arguments node between the name and them; both are methods. For members the ranking is used the other way round: the broad
// pattern is a field (property) and the narrower one a method, and the method,
// ranking higher, wins.
//
// Lets are definitions only directly inside a file, namespace or module;
// a let in a function body or in a class (a private field) is not.
func init() {
	scopes := []string{
		"(file %s)",
		"(namespace %s)",
		"(named_module %s)",
		"(module_defn %s)",
	}
	const pat = "(identifier_pattern . (long_identifier_or_op [(identifier) @name (long_identifier . (identifier) @name .)]) %s)"
	// A type with type parameters (`type Box<'T> =`) has its name wrapped in a
	// long_identifier; one without has a bare identifier.
	const tn = "[(identifier) @name (long_identifier . (identifier) @name)]"
	decls := []string{
		// functions
		"(declaration_expression (function_or_value_defn (function_declaration_left . [(identifier) (op_identifier) (active_pattern)] @name) ) @definition.function)",
		"(declaration_expression (function_or_value_defn (value_declaration_left " + strings.Replace(pat, "%s", ". (_)", 1) + ")) @definition.function)",
		"(declaration_expression (function_or_value_defn (value_declaration_left " + strings.Replace(pat, "%s", ".", 1) + ") . (fun_expression)) @definition.function)",
		"(declaration_expression (function_or_value_defn (value_declaration_left " + strings.Replace(pat, "%s", ".", 1) + ") . (function_expression)) @definition.function)",
		// values
		"(declaration_expression (function_or_value_defn (value_declaration_left " + strings.Replace(pat, "%s", ".", 1) + ") . (_) @_body) @definition.variable (#not-match? @_body \"^(fun|function)\\\\b\"))",
		// types
		"(type_definition [(union_type_defn (type_name " + tn + ")) (record_type_defn (type_name " + tn + ")) (anon_type_defn (type_name " + tn + ")) (delegate_type_defn (type_name " + tn + ")) (enum_type_defn (type_name " + tn + "))] @definition.class)",
		"(type_definition (interface_type_defn (type_name " + tn + ")) @definition.interface)",
		"(type_definition (type_abbrev_defn (type_name [(identifier) @name (long_identifier . (identifier) @name)])) @definition.type_alias)",
		// `type Id = int` has no bar and no fields: the grammar reads it as a union with one case
		"(type_definition (union_type_defn (type_name " + tn + ") (union_type_cases . (union_type_case (identifier) .) .)) @definition.type_alias)",
		// modules
		"(module_defn . (identifier) @name) @definition.module",
		"(module_defn (attributes) . (identifier) @name) @definition.module",
		"(exception_definition . [(identifier) @name (long_identifier . (identifier) @name .)]) @definition.class",
	}
	// The grammar sometimes reads a top-level let as a value_declaration
	// rather than a declaration_expression (after a module it lost track of).
	for _, d := range decls {
		if strings.HasPrefix(d, "(declaration_expression") {
			decls = append(decls, strings.Replace(d, "(declaration_expression", "(value_declaration", 1))
		}
	}
	var b strings.Builder
	for _, scope := range scopes {
		for _, d := range decls {
			b.WriteString(strings.Replace(scope, "%s", d, 1))
			b.WriteByte('\n')
		}
	}
	// A module or namespace header names itself.
	b.WriteString("(named_module . (long_identifier) @name) @definition.module\n")
	b.WriteString("(namespace . (long_identifier) @name) @definition.module\n")
	// When the content of a namespace or module is indented, the grammar drops
	// the header node and leaves its dotted name as the first thing in the
	// file. Nothing else puts a bare long_identifier there.
	b.WriteString("(file . (long_identifier) @name) @definition.module\n")
	// Members
	b.WriteString(`
(member_defn (method_or_prop_defn . (property_or_ident (identifier) @name .)) @definition.field)
(member_defn (method_or_prop_defn . (property_or_ident (identifier) @name .) . [(paren_pattern) (argument_patterns) (identifier_pattern) (typed_pattern) (const (unit))]) @definition.method)
(member_defn (method_or_prop_defn . (property_or_ident (identifier) @name .) . (type_arguments) . [(paren_pattern) (argument_patterns) (identifier_pattern) (typed_pattern) (const (unit))]) @definition.method)
(member_defn . (property_or_ident (identifier) @name .)) @definition.field
(member_defn (member_signature . (identifier) @name) @definition.field)
(member_defn (member_signature . (identifier) @name (curried_spec (arguments_spec))) @definition.method)
`)
	registerDefinitions("fsharp", b.String(), true)
}
