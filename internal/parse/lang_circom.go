package parse

// Circom. The inferred query already captures function_definition and calls,
// so this adds to it rather than replacing it.
//
// What is a definition: template (class), function (function, from the
// inferred query), and the `component main` entry point (function, named
// main). Signals, variables and component instances inside a template are
// locals of that template and are not definitions.
//
// The grammar lexes the identifier after `template` with the whitespace in
// front of it when both are on one line, so `template Num2Bits(n)` yields the
// name " Num2Bits". The query cannot trim that, and a symbol with a wrong name
// is worse than a gap, so the template pattern accepts only a name that starts
// with a letter. That leaves out the usual one-line form; see KNOWN_GAPS.md.
func init() {
	registerDefinitions("circom", `
(template_definition (identifier) @name (#match? @name "^[A-Za-z_$]")) @definition.class
(main_component_definition "main" @name) @definition.function
`, false)
}
