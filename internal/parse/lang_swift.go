package parse

// Swift: types, functions, initialisers, subscripts, properties, type aliases,
// enum cases and operator declarations.
//
// The inferred query is replaced. It names enum cases `enum_member`, which
// becomes a function, and gives every function the kind `function` even
// inside a type, so methods were indistinguishable from free functions. It
// also finds no initialiser, deinitialiser, subscript, property, typealias,
// associatedtype or extension.
//
// Kinds: struct, enum, actor, class and extension are `class`; a protocol is
// `interface`; a function in a type or protocol body is `method`, anywhere
// else `function` (a free function, or one nested in another function).
// init, deinit and subscript are `method` named by their keyword (several
// initialisers in one type share the name `init`; they differ by line).
// Properties declared in a type body, computed or stored, static or not, are
// `field`. At file level `let` is `constant` and `var` is `variable`; locals
// are not recorded. Enum cases are `constant`. A typealias and an
// associatedtype are `type_alias`. An operator declaration
// (`infix operator **`) is a `function` named by the operator; an operator
// implementation (`static func ==`) is a method named by the operator.
//
// An extension is recorded as a `class` under the name of the type it
// extends, at the extension's own line, so that `Array` has an entry for each
// place it is extended. Its members are methods and fields as in any type.
//
// A declaration with attributes (`@MainActor`, `@discardableResult`) starts at
// the attribute, because the grammar makes the attributes part of the
// declaration; that is the line recorded.
func init() {
	registerDefinitions("swift", `
(protocol_declaration (type_identifier) @name) @definition.interface
(class_declaration "class" (type_identifier) @name) @definition.class
(class_declaration "struct" (type_identifier) @name) @definition.class
(class_declaration "actor" (type_identifier) @name) @definition.class
(class_declaration "enum" (type_identifier) @name) @definition.class
(class_declaration "extension" (user_type (type_identifier) @name .)) @definition.class

(typealias_declaration (type_identifier) @name) @definition.type_alias
(associatedtype_declaration (type_identifier) @name) @definition.type_alias

(enum_entry (simple_identifier) @name) @definition.constant

; functions: free and nested ones are functions, those in a type are methods
(source_file (function_declaration (simple_identifier) @name) @definition.function)
(function_body (statements (function_declaration (simple_identifier) @name) @definition.function))
(class_body (function_declaration (simple_identifier) @name) @definition.method)
(enum_class_body (function_declaration (simple_identifier) @name) @definition.method)
(protocol_function_declaration (simple_identifier) @name) @definition.method

; operator implementations: a custom operator is a node, the built-in ones are tokens
(class_body (function_declaration "func" [(custom_operator) "==" "!=" "<" ">" "<=" ">=" "+" "-" "*" "/" "%" "&" "|" "^" "!" "~" "&&" "||" "<<" ">>" "+=" "-=" "*=" "/=" "??" "..." "..<" "~=" "==="] @name) @definition.method)
(enum_class_body (function_declaration "func" [(custom_operator) "==" "!=" "<" ">" "<=" ">=" "+" "-" "*" "/" "%" "&" "|" "^" "!" "~" "&&" "||" "<<" ">>" "+=" "-=" "*=" "/=" "??" "..." "..<" "~=" "==="] @name) @definition.method)
(source_file (function_declaration "func" [(custom_operator) "==" "!=" "<" ">" "<=" ">=" "+" "-" "*" "/" "%" "&" "|" "^" "!" "~" "&&" "||" "<<" ">>" "+=" "-=" "*=" "/=" "??" "..." "..<" "~=" "==="] @name) @definition.function)

(init_declaration "init" @name) @definition.method
(deinit_declaration "deinit" @name) @definition.method
(subscript_declaration "subscript" @name) @definition.method

(operator_declaration (custom_operator) @name) @definition.function
(macro_declaration (simple_identifier) @name) @definition.function

; properties
(source_file (property_declaration (value_binding_pattern) @_b (pattern (simple_identifier) @name)) @definition.constant
  (#match? @_b "^let"))
(source_file (property_declaration (value_binding_pattern) @_b (pattern (simple_identifier) @name)) @definition.variable
  (#match? @_b "^var"))
(source_file (property_declaration (value_binding_pattern) @_b (pattern (pattern (simple_identifier) @name))) @definition.constant
  (#match? @_b "^let"))
(source_file (property_declaration (value_binding_pattern) @_b (pattern (pattern (simple_identifier) @name))) @definition.variable
  (#match? @_b "^var"))
(class_body (property_declaration (pattern (simple_identifier) @name)) @definition.field)
(enum_class_body (property_declaration (pattern (simple_identifier) @name)) @definition.field)
(protocol_property_declaration (pattern (simple_identifier) @name)) @definition.field
`, true)
}
