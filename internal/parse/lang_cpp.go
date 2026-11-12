package parse

// C++: the C declarations plus classes, enums and out-of-line members.
//
// The query the library infers is replaced for the reasons lang_c.go gives, and
// for three more:
//
//   - `class Cache;` and `struct Foo;` are forward declarations. They name a
//     type and define nothing, and a symbol for one is where an inheritance
//     edge ends instead of at the definition. A class or struct is a
//     definition only when it has a body (the supplement in queries.go says
//     so for both).
//   - A member defined outside its class, `void Table::Open(int) { }`, has a
//     qualified_identifier for a name and matched nothing. It is recorded as a
//     method named by its last segment; shape.go reads the owner from the
//     source (`Table.Open`), and a namespace in front (`ns::Table::Open`) does
//     not change either.
//   - The same for a function returning a pointer or a reference, which has a
//     pointer_declarator or reference_declarator in between.
func init() {
	registerDefinitions("cpp", `
(function_definition (function_declarator (identifier) @name)) @definition.function
(function_definition (function_declarator (field_identifier) @name)) @definition.function
(function_definition [
  (pointer_declarator (function_declarator [(identifier) (field_identifier)] @name))
  (pointer_declarator (pointer_declarator (function_declarator [(identifier) (field_identifier)] @name)))
  (reference_declarator (function_declarator [(identifier) (field_identifier)] @name))
]) @definition.function

(function_definition [
  (function_declarator (qualified_identifier (identifier) @name .))
  (function_declarator (qualified_identifier (qualified_identifier (identifier) @name .) .))
  (function_declarator (qualified_identifier (qualified_identifier (qualified_identifier (identifier) @name .) .) .))
  (pointer_declarator (function_declarator (qualified_identifier (identifier) @name .)))
  (pointer_declarator (function_declarator (qualified_identifier (qualified_identifier (identifier) @name .) .)))
  (pointer_declarator (function_declarator (qualified_identifier (qualified_identifier (qualified_identifier (identifier) @name .) .) .)))
  (reference_declarator (function_declarator (qualified_identifier (identifier) @name .)))
  (reference_declarator (function_declarator (qualified_identifier (qualified_identifier (identifier) @name .) .)))
  (reference_declarator (function_declarator (qualified_identifier (qualified_identifier (qualified_identifier (identifier) @name .) .) .)))
]) @definition.method

(type_definition (type_identifier) @name .) @definition.type
(type_definition (function_declarator (parenthesized_declarator (pointer_declarator (type_identifier) @name)))) @definition.type
(type_definition (pointer_declarator (function_declarator (parenthesized_declarator (pointer_declarator (type_identifier) @name))))) @definition.type

(call_expression (identifier) @name) @reference.call
(call_expression (field_expression (field_identifier) @name)) @reference.call
`, true)
}
