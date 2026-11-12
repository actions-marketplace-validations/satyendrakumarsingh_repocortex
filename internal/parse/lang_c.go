package parse

// C: functions, struct and typedef names, and calls.
//
// The query the library infers is replaced for three reasons, each measured on
// real code:
//
//   - `(struct_specifier (type_identifier) @name)` also matches a struct that
//     is only NAMED, not defined: `struct Foo;` and `typedef struct Foo Foo_t;`
//     recorded `Foo` as a definition with no body.
//   - `(type_definition (type_identifier) @name)` matches any type name in the
//     typedef, so `typedef Iterator* (*BlockFunction)(void*);` declared
//     `Iterator`, the return type, and never `BlockFunction`. A typedef names
//     what stands last among its children, or the name inside the declarator
//     of a function-pointer typedef.
//   - A function returning a pointer, `char *name(int a) { ... }`, has a
//     pointer_declarator between the definition and its function_declarator and
//     matched nothing.
func init() {
	registerDefinitions("c", `
(function_definition (function_declarator (identifier) @name)) @definition.function
(function_definition (pointer_declarator (function_declarator (identifier) @name))) @definition.function
(function_definition (pointer_declarator (pointer_declarator (function_declarator (identifier) @name)))) @definition.function

(type_definition (type_identifier) @name .) @definition.type
(type_definition (function_declarator (parenthesized_declarator (pointer_declarator (type_identifier) @name)))) @definition.type
(type_definition (pointer_declarator (function_declarator (parenthesized_declarator (pointer_declarator (type_identifier) @name))))) @definition.type
(struct_specifier (type_identifier) @name (field_declaration_list)) @definition.type

(call_expression (identifier) @name) @reference.call
(call_expression (field_expression (field_identifier) @name)) @reference.call
`, true)
}
