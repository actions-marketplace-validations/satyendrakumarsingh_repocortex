package parse

import "strings"

// Bash (and POSIX sh): functions, aliases, and the variables a script sets at
// its top level.
//
// A function is recorded in all four spellings (`name() {}`, `function name
// {}`, `function name() {}` and a subshell body `name() ( ... )`), wherever it
// is written, nested ones and those inside `if` included.
//
// A variable is recorded when it is assigned at the top level of the file:
// `NAME=value`, `export NAME=value`, `declare [-flags] NAME[=value]`,
// `typeset` and `readonly NAME=value`. `readonly` and `declare -r` make a
// `constant`; everything else is a `variable`. Locals (`local`, and anything
// assigned inside a function, loop or conditional) are not recorded, and a
// bare `export NAME` re-exports a name that is declared elsewhere and is not
// recorded. A name assigned twice at the top level is recorded at each
// assignment. An append (`NAME+=value`, `NAME+=(...)`) changes a variable
// that is declared elsewhere and is not an assignment of its own, so it is not
// recorded.
//
// An `alias name=value` anywhere in the file is a `function`: it defines a
// command. The grammar gives the name together with its `=` (`ll=`), which
// bashName removes.
func init() {
	registerDefinitions("bash", `
(function_definition name: (word) @name) @definition.function

(program (variable_assignment name: (variable_name) @name) @definition.variable
  (#not-match? @definition.variable "^[^=]*[+]="))
(program (variable_assignments (variable_assignment name: (variable_name) @name) @definition.variable
  (#not-match? @definition.variable "^[^=]*[+]=")))

(program (declaration_command (variable_assignment name: (variable_name) @name)) @definition.constant
  (#match? @definition.constant "^(readonly\\b|(declare|typeset)\\s+(-\\S+\\s+)*-[A-Za-z]*r)"))
(program (declaration_command (variable_assignment name: (variable_name) @name)) @definition.variable
  (#not-match? @definition.variable "^(readonly\\b|local\\b|(declare|typeset)\\s+(-\\S+\\s+)*-[A-Za-z]*r)"))
(program (declaration_command (variable_name) @name) @definition.variable
  (#match? @definition.variable "^(declare|typeset)\\b"))

(command (command_name (word) @_c) . (concatenation . (word) @name)) @definition.function
  (#eq? @_c "alias")
(command (command_name (word) @_c) . (word) @name) @definition.function
  (#eq? @_c "alias")
  (#match? @name "^[^=-]+=")
`, true)
	registerNameFix("bash", bashName)
}

// bashName cuts an alias name at its `=`. Other captured names (function and
// variable names) cannot contain one.
func bashName(name string) string {
	if i := strings.IndexByte(name, '='); i > 0 {
		return name[:i]
	}
	return name
}
