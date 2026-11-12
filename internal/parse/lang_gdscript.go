package parse

// GDScript: a script is itself a class. What it declares at the top of the
// file (outside any `class`) is recorded as a function, constant, variable;
// what an inner `class` declares is a method, constant, field. A signal is a
// field. Locals inside a function body are `variable_statement` nodes under a
// `body`, so every pattern is anchored to the source file or a class body.
//
// An anonymous `enum { A, B }` has no name and is not extracted, nor are the
// enumerators of a named one.
func init() {
	registerDefinitions("gdscript", `
(source (class_name_statement (name) @name) @definition.class)
(source (class_definition (name) @name) @definition.class)
(class_body (class_definition (name) @name) @definition.class)

(source (function_definition (name) @name) @definition.function)
(class_body (function_definition (name) @name) @definition.method)

(source (variable_statement (name) @name) @definition.variable)
(class_body (variable_statement (name) @name) @definition.field)
(source (export_variable_statement (name) @name) @definition.variable)
(source (onready_variable_statement (name) @name) @definition.variable)
(class_body (export_variable_statement (name) @name) @definition.field)
(class_body (onready_variable_statement (name) @name) @definition.field)

(source (const_statement (name) @name) @definition.constant)
(class_body (const_statement (name) @name) @definition.constant)

(source (signal_statement (name) @name) @definition.field)
(class_body (signal_statement (name) @name) @definition.field)

(source (enum_definition (name) @name) @definition.class)
(class_body (enum_definition (name) @name) @definition.class)
`, true)
}
