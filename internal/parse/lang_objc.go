package parse

// Objective-C: the C declarations (functions, typedefs, file-level
// variables) and the Objective-C ones (@interface, @implementation,
// @protocol, categories, methods, properties, instance variables).
//
// The inferred query is replaced because it tags every identifier of a method
// declaration (the parameter names too) as a method and finds no classes. The
// call references it provided are kept.
//
// A method is named by its first selector part: `scaleBy:animated:` is
// recorded as `scaleBy`, the only part that is a node of its own.
//
// A category (`@interface Shape (Printing)`) and a class extension
// (`@interface Shape ()`) have the same tree as a class with a superclass:
// two identifiers, no field to tell them apart, and the parentheses are not
// nodes a pattern can name. They are told apart by the first words of the
// declaration's own text. A category is recorded under its own name; an
// extension has none and is not recorded.
func init() {
	registerDefinitions("objc", `
(call_expression (identifier) @name) @reference.call
(call_expression (field_identifier) @name) @reference.call

; C functions
(function_definition (function_declarator . (identifier) @name)) @definition.function
(function_definition (pointer_declarator (function_declarator . (identifier) @name))) @definition.function

; @interface Name : Super, @interface Name, and @interface Name (Category)
(class_interface . (identifier) @name) @definition.class
  (#not-match? @definition.class "^@interface\\s+[A-Za-z0-9_]+\\s*\\(")
(class_interface . (identifier) (identifier) @name) @definition.class
  (#match? @definition.class "^@interface\\s+[A-Za-z0-9_]+\\s*\\(")
(class_implementation . (identifier) @name) @definition.class
  (#not-match? @definition.class "^@implementation\\s+[A-Za-z0-9_]+\\s*\\(")
(class_implementation . (identifier) (identifier) @name) @definition.class
  (#match? @definition.class "^@implementation\\s+[A-Za-z0-9_]+\\s*\\(")
(protocol_declaration . (identifier) @name) @definition.interface

; methods, declared or defined; "+" and "-" alike
(method_declaration (method_type) . (identifier) @name) @definition.method
(method_definition (method_type) . (identifier) @name) @definition.method

; @property and instance variables
(property_declaration (struct_declaration (struct_declarator [(identifier) @name (pointer_declarator (identifier) @name)]))) @definition.field
(property_declaration (struct_declaration (struct_declarator (function_declarator (parenthesized_declarator (block_pointer_declarator (identifier) @name)))))) @definition.field
(instance_variable (struct_declaration (struct_declarator [(identifier) @name (pointer_declarator (identifier) @name)]))) @definition.field

; typedefs
(type_definition (struct_specifier) (type_identifier) @name .) @definition.class
(type_definition (enum_specifier) (type_identifier) @name .) @definition.class
(type_definition (union_specifier) (type_identifier) @name .) @definition.class
(type_definition [(type_identifier) (primitive_type) (sized_type_specifier)] (type_identifier) @name .) @definition.type_alias
(type_definition (function_declarator (parenthesized_declarator [(block_pointer_declarator (type_identifier) @name) (pointer_declarator (type_identifier) @name)]))) @definition.type_alias

; file-level variables with an initialiser (constants when const), and without one
; when the declaration starts with a storage class or a built-in type; a plain
; a plain "Foo bar;" is not told apart from a function whose return-type macro the
; grammar does not know
(translation_unit (declaration (init_declarator . [(identifier) @name (pointer_declarator (identifier) @name) (pointer_declarator (pointer_declarator (identifier) @name))])) @definition.variable)
(translation_unit (declaration (type_qualifier) @_q (init_declarator . [(identifier) @name (pointer_declarator (identifier) @name)])) @definition.constant
  (#match? @_q "^const$"))
(translation_unit (declaration (init_declarator . (pointer_declarator (type_qualifier) @_q (identifier) @name))) @definition.constant
  (#match? @_q "^const$"))
(translation_unit (declaration [(storage_class_specifier) (primitive_type) (sized_type_specifier)] [(identifier) @name (pointer_declarator (identifier) @name)] .) @definition.variable
  (#not-match? @definition.variable "^extern\\b"))
`, true)
}
