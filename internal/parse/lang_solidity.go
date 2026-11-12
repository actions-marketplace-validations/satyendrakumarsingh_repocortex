package parse

// Solidity. What the index holds, see fixtures/langs/solidity/expected.tsv:
//
//   - contract (also abstract) and library are a class, and so are a struct
//     and an enum, at file level or inside a contract; interface is an
//     interface;
//   - function is a function, free or in a contract, interface or library;
//     event and custom error declarations are a function (they are called by
//     emit and revert);
//   - modifier, constructor, receive and fallback are a method: they exist
//     only inside a contract and are not called by name;
//   - `type Price is uint256;` is a type alias;
//   - a constant or immutable, in a contract or at file level, is a constant;
//     any other state variable is a field.
//
// Struct members, enum values, parameters, local variables and `using`
// directives are not definitions.
func init() {
	registerDefinitions("solidity", solidityDefinitions, true)
}

const solidityDefinitions = `
(contract_declaration name: (identifier) @name) @definition.class
(library_declaration name: (identifier) @name) @definition.class
(interface_declaration name: (identifier) @name) @definition.interface
(struct_declaration name: (identifier) @name) @definition.class
(enum_declaration name: (identifier) @name) @definition.class
(function_definition name: (identifier) @name) @definition.function
(event_definition name: (identifier) @name) @definition.function
(error_declaration name: (identifier) @name) @definition.function
(modifier_definition name: (identifier) @name) @definition.method
(constructor_definition "constructor" @name) @definition.method
(fallback_receive_definition ["receive" "fallback"] @name) @definition.method
(user_defined_type_definition name: (identifier) @name) @definition.type_alias
(constant_variable_declaration name: (identifier) @name) @definition.constant

((state_variable_declaration name: (identifier) @name) @definition.constant
  (#match? @definition.constant "^[^=;]*\\b(constant|immutable)\\b"))

((state_variable_declaration name: (identifier) @name) @definition.field
  (#not-match? @definition.field "^[^=;]*\\b(constant|immutable)\\b"))
`
