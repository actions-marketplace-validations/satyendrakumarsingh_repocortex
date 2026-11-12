package parse

import "strings"

// Dart: classes (abstract, sealed, base, final, interface and mixin classes),
// mixins, enums and their constants, extensions, extension types, typedefs,
// functions, methods, constructors, getters, setters, operators, fields and
// top-level variables.
//
// The inferred query is replaced. It records a constructor under every
// identifier of its signature, so `Shape.named()` gave a symbol `Shape` and a
// symbol `named`; it files a mixin and every ordinary method as a function;
// and it finds no setter, operator, typedef, extension, extension type,
// factory constructor, field or top-level variable.
//
// The Dart grammar does not wrap a declaration: at file level a function is a
// `function_signature` followed by a sibling `function_body`, and a variable
// is a type node followed by an `initialized_identifier_list`. A symbol's range
// is therefore the signature only, and for a file-level variable starts at the
// name, not at `const` or the type.
//
// Kinds: classes, enums, extensions and extension types are `class`; a mixin
// is `interface`; a typedef is `type_alias`; enum constants and `const` or
// `final` initialised variables (`static_final_declaration_list`) are
// `constant`; a `var` or typed variable is `variable` at file level and
// `field` in a type; a function at file level or nested in a function is
// `function`, in a type `method`; constructors, getters, setters and operators
// in a type are `method` (a file-level getter or setter is a `function`).
//
// A constructor is named as it is written: `Shape` for `Shape(...)`,
// `Shape.named` for `Shape.named(...)` and `Shape._` for `Shape._()`. Its two
// identifiers are not one node, so the query captures the whole signature as
// the name and the name fix below cuts it back to the words before the
// parameter list (dropping `const` and `factory`). An operator is
// named by its symbol (`==`, `+`, `[]`). An unnamed extension (`extension on
// int`) has no name and is not recorded; its members are.
func init() {
	registerDefinitions("dart", `
(class_definition (identifier) @name) @definition.class
(class_definition (mixin_application_class . (identifier) @name)) @definition.class
(mixin_declaration (identifier) @name) @definition.interface
(enum_declaration (identifier) @name) @definition.class
(enum_constant (identifier) @name) @definition.constant
(extension_declaration (identifier) @name) @definition.class
(extension_type_declaration (identifier) @name) @definition.class

; typedefs: "typedef Name<T> = ...;" and the old "typedef R Name(args);"
(type_alias . (type_identifier) @name) @definition.type_alias
  (#match? @definition.type_alias "^typedef\\s+[A-Za-z0-9_$]+\\s*(<[^=]*>)?\\s*=")
(type_alias (type_identifier) @name . (formal_parameter_list)) @definition.type_alias
(type_alias (type_identifier) @name . (type_parameters) . (formal_parameter_list)) @definition.type_alias

; functions
(program (function_signature (identifier) @name) @definition.function)
(program (getter_signature (identifier) @name) @definition.function)
(program (setter_signature (identifier) @name) @definition.function)
(local_function_declaration (lambda_expression (function_signature (identifier) @name) @definition.function))

; members
(method_signature (function_signature (identifier) @name) @definition.method)
(declaration (function_signature (identifier) @name) @definition.method)
(method_signature (getter_signature (identifier) @name) @definition.method)
(declaration (getter_signature (identifier) @name) @definition.method)
(method_signature (setter_signature (identifier) @name) @definition.method)
(declaration (setter_signature (identifier) @name) @definition.method)
(operator_signature (binary_operator) @name) @definition.method
(operator_signature ["[]" "[]=" "~" "-"] @name) @definition.method

; constructors: the whole signature is captured as the name, see dartName
(constructor_signature (identifier) . (formal_parameter_list)) @definition.constructor @name
(constant_constructor_signature (identifier) . (formal_parameter_list)) @definition.constructor @name
(factory_constructor_signature (identifier) . (formal_parameter_list)) @definition.constructor @name
(redirecting_factory_constructor_signature (identifier) . (formal_parameter_list)) @definition.constructor @name

; variables and fields
(program (static_final_declaration_list (static_final_declaration . (identifier) @name)) @definition.constant)
(program (initialized_identifier_list (initialized_identifier . (identifier) @name)) @definition.variable)
(declaration (static_final_declaration_list (static_final_declaration . (identifier) @name))) @definition.constant
(declaration (initialized_identifier_list (initialized_identifier . (identifier) @name))) @definition.field
`, true)
	registerNameFix("dart", dartName)
}

// dartName reduces a captured constructor signature (`const factory
// Shape.circle(double r) = Circle`) to the name it declares (`Shape.circle`).
// Every other captured name is a single identifier or operator and has no
// parenthesis, so it is returned as it is.
func dartName(name string) string {
	head, _, found := strings.Cut(name, "(")
	if !found {
		return name
	}
	words := strings.Fields(head)
	if len(words) == 0 {
		return name
	}
	return words[len(words)-1]
}
