package parse

import "strings"

// Make. What the index holds, see fixtures/langs/make/expected.tsv:
//
//   - a rule's literal target names are a function (`build`, `test`, `lib.a`),
//     one per name in a multi-target rule;
//   - `define NAME ... endef` is a function (a callable macro);
//   - a variable assigned with `=`, `:=`, `::=`, `?=` or `!=`, also after
//     `export` or `override` and inside a conditional, is a variable.
//
// A `+=` appends to a variable that is defined elsewhere, so it is not
// captured: a variable is recorded where it is first assigned, not at every
// append. Make's own settings (SHELL, MAKEFLAGS, MAKEFILES, VPATH) and every
// name that begins with a dot (.DEFAULT_GOAL, and the BSD directives .if,
// .else, .endif and .include, which the grammar reads as assignments) are
// configuration of make, not names of the project, and are not captured. A
// target-specific assignment (`app: CFLAGS = -O0`) is scoped to that target
// and is not captured either.
//
// Not captured: prerequisites; pattern rules (a target containing %), suffix
// rules and special targets such as .PHONY (a target starting with a dot); a
// name that begins with - @ or + or contains a backslash (a recipe line of a
// non-GNU makefile indented with spaces, which the grammar reads as a rule); a
// target spelled with a variable reference ($(BUILD)/out); include and vpath
// directives; conditionals.
func init() {
	registerDefinitions("make", makeDefinitions, false)
	registerNameFix("make", makeName)
}

// makeName drops an assignment operator the grammar lexes into the variable
// name when there is no space before it: `CC?=gcc` names the word "CC?".
func makeName(name string) string {
	return strings.TrimRight(name, "?:+!")
}

const makeDefinitions = `
(rule
  (targets (word) @name)
  (#not-match? @name "^\\.")
  (#not-match? @name "%")
  (#not-match? @name "^[-@+]|\\\\")) @definition.function

(define_directive (word) @name) @definition.function

((variable_assignment . (word) @name) @definition.variable
  (#not-match? @name "^(SHELL|MAKEFLAGS|MAKEFILES|VPATH|\\..*)$")
  (#not-match? @definition.variable "^[^=:]*\\+="))

((shell_assignment . (word) @name) @definition.variable
  (#not-match? @name "^(SHELL|MAKEFLAGS|MAKEFILES|VPATH|\\..*)$"))
`
