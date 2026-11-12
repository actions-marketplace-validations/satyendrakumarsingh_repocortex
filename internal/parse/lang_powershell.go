package parse

import "regexp"

// PowerShell. What the index holds, see fixtures/langs/powershell/expected.tsv:
//
//   - function, filter and workflow are a function, at any depth, under the
//     name written after the keyword with its scope modifier removed:
//     `function script:Get-Item` is Get-Item, as it is called;
//   - class and enum are a class;
//   - a method or constructor of a class is a method.
//
// Class properties, enum members, parameters and variables are not
// definitions. A class's base class (`class Dog : Animal`) is a use of Animal,
// not a second definition of it, so only the first name of the class counts.
func init() {
	registerDefinitions("powershell", powershellDefinitions, true)
	registerNameFix("powershell", powershellName)
}

// powershellScope is the scope modifier that may precede a function's name.
var powershellScope = regexp.MustCompile(`(?i)^(global|script|local|private):`)

// powershellName drops the scope modifier from a function name. The same
// function is called without it, so the name with it matches no call.
func powershellName(name string) string {
	return powershellScope.ReplaceAllString(name, "")
}

const powershellDefinitions = `
(function_statement (function_name) @name) @definition.function
(class_statement . (simple_name) @name) @definition.class
(class_method_definition (simple_name) @name) @definition.method
(enum_statement . (simple_name) @name) @definition.class
`
