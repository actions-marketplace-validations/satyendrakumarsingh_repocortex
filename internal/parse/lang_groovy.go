package parse

import "strings"

// Groovy (and Gradle build scripts, Jenkinsfiles, Spock specifications).
//
// The grammar reads only part of the language. A class body is a `closure`; a
// `trait` and an `enum` are not nodes of their own but the sequence
// identifier, identifier, closure; `implements` is an ERROR node, which moves
// the class name into it when the class has no `extends`; a constructor is
// parsed as a call with a closure argument. The patterns below follow what the
// tree really is, and each was checked against files for what it also matches.
//
// The inferred query tagged every identifier child of a class (so the
// superclass was a second class, and `implements Serializable` produced the
// class `erializable`) and found neither constructors, generic methods,
// Spock's string-named methods, enums nor traits.
//
// Kinds: class and enum are `class`; interface and trait are `interface`;
// methods and constructors in a class body are `method`; a `def` or typed
// function at the top level of a script is `function`; a Gradle task
// (`task name {}`, `task name(type: T) {}`, `task('name') {}`,
// `tasks.register('name') {}`, `tasks.create('name')`, `project.task('name')`)
// is `function` named by the task. A field whose value is a closure
// (`def handler = { x -> ... }`) is a `method` in a class and a `function` at
// the top level of a script. A Spock feature method is named by its quoted
// title, without the quotes.
//
// Plain fields and script variables are not recorded: a misread constructor
// leaves its parameters behind as declarations, so a field pattern would also
// record parameters.
func init() {
	registerDefinitions("groovy", `
; class, interface, @interface: the first identifier after the keyword. A class that has
; "implements" and no "extends" is read with the keyword inside an ERROR node and its name
; mangled; that form is the ERROR followed by an identifier (the mangled name).
(class_definition "class" . (identifier) @name) @definition.class
(class_definition "interface" . (identifier) @name) @definition.interface
(class_definition "@interface" . (identifier) @name) @definition.interface
(class_definition (ERROR . (identifier) @name) . (identifier)) @definition.class

; trait and enum are the sequence identifier, identifier, closure; with "implements" the name
; is read as a declaration
(source_file (identifier) @_k . (identifier) @name @definition.interface . (closure) (#eq? @_k "trait"))
(source_file (identifier) @_k . (identifier) @name @definition.class . (closure) (#eq? @_k "enum"))
(closure (identifier) @_k . (identifier) @name @definition.interface . (closure) (#eq? @_k "trait"))
(closure (identifier) @_k . (identifier) @name @definition.class . (closure) (#eq? @_k "enum"))
(source_file (identifier) @_k . (declaration . (identifier) @name @definition.interface . (identifier)) (#eq? @_k "trait"))
(source_file (identifier) @_k . (declaration . (identifier) @name @definition.class . (identifier)) (#eq? @_k "enum"))

; functions at the top of a script, methods in a body
(source_file (function_definition (identifier) @name . (parameter_list)) @definition.function)
(class_definition (closure (function_definition (identifier) @name . (parameter_list)) @definition.method))
(class_definition (closure (function_definition (quoted_identifier) @name . (parameter_list)) @definition.method))
(class_definition (closure (function_declaration (identifier) @name . (parameter_list)) @definition.method))
(source_file (closure (function_definition (identifier) @name . (parameter_list)) @definition.method))
(source_file (closure (function_declaration (identifier) @name . (parameter_list)) @definition.method))

; constructors, generic methods and bodiless methods with a class-typed parameter are read as a
; call, which has no other meaning directly in a class body
(class_definition (closure (function_call (identifier) @name) @definition.method))

; a method with modifiers and no return type (private static name(a) {) is a declaration of
; the modifiers and the name, then the parameters as a parenthesised expression, then the body
(class_definition (closure (declaration (identifier) @name .) @definition.method . (parenthesized_expression) . (closure)))
(source_file (declaration (identifier) @name .) @definition.function . (parenthesized_expression) . (closure))

; closures assigned to a field or script variable
(class_definition (closure (declaration name: (identifier) @name (closure)) @definition.method))
(source_file (declaration name: (identifier) @name (closure)) @definition.function)

; Gradle tasks
(source_file (identifier) @_t . (juxt_function_call (identifier) @name) @definition.function (#eq? @_t "task"))
(source_file (identifier) @_t . (function_call (identifier) @name) @definition.function (#eq? @_t "task"))
(function_call (identifier) @_t (argument_list . (string (string_content) @name))) @definition.function
  (#eq? @_t "task")
(function_call (dotted_identifier (identifier) @_o (identifier) @_m) (argument_list . (string (string_content) @name))) @definition.function
  (#eq? @_o "tasks")
  (#any-of? @_m "register" "create" "maybeCreate")
(function_call (dotted_identifier (identifier) @_o (identifier) @_m) (argument_list . (string (string_content) @name))) @definition.function
  (#eq? @_m "task")
`, true)
	registerNameFix("groovy", groovyName)
}

// groovyName removes the quotes of a Spock feature title (`"adds two numbers"`),
// which is a quoted_identifier node that includes them.
func groovyName(name string) string {
	if len(name) >= 2 && (name[0] == '"' || name[0] == '\'') && name[len(name)-1] == name[0] {
		return strings.TrimSpace(name[1 : len(name)-1])
	}
	return name
}
