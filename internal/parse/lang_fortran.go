package parse

// Fortran: program units (program, module, submodule, block data),
// procedures (function, subroutine, wherever they sit: in a module, after
// `contains`, in an interface block, or at file level), derived types, generic
// interfaces, type-bound procedure bindings, and the variables and named
// constants a module declares. Variables of a procedure or of a main program
// are locals and are not extracted.
//
// The name of every unit is the `name` of its opening statement.
func init() {
	registerDefinitions("fortran", `
(program (program_statement (name) @name)) @definition.module
(module (module_statement (name) @name)) @definition.module
(submodule (submodule_statement . (module_name) (name) @name)) @definition.module
(block_data (block_data_statement (name) @name)) @definition.module

(function (function_statement (name) @name)) @definition.function
(subroutine (subroutine_statement (name) @name)) @definition.function

(derived_type_definition (derived_type_statement (type_name) @name)) @definition.class
(interface (interface_statement (name) @name)) @definition.interface

(derived_type_procedures (procedure_statement (binding (binding_name (identifier) @name)) @definition.method))

(module (variable_declaration [(identifier) @name (init_declarator . (identifier) @name) (sized_declarator . (identifier) @name)]) @definition.variable)
(module (variable_declaration (type_qualifier) @_q [(identifier) @name (init_declarator . (identifier) @name) (sized_declarator . (identifier) @name)]) @definition.constant
  (#match? @_q "(?i)^parameter$"))
`, true)
}
