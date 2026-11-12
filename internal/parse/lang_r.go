package parse

// R has no declaration syntax: a function is a value assigned to a name, and
// an S4 or reference class is a call to setClass / setRefClass. The grammar's
// `binary_operator` has the operator as an anonymous node, so it is matched by
// field and literal.
//
// Only file-level assignments are definitions; a function assigned inside
// another function's body is a local. Non-function assignments (`x <- 10`)
// are not extracted: in R they are data or script state, not API.
func init() {
	registerDefinitions("r", `
; name <- function(...)  /  name = function(...)  /  name <<- function(...)
(program (binary_operator lhs: (identifier) @name operator: ["<-" "=" "<<-"] rhs: (function_definition)) @definition.function)
(program (binary_operator lhs: (string (string_content) @name) operator: ["<-" "=" "<<-"] rhs: (function_definition)) @definition.function)
; function(...) ... -> name
(program (binary_operator lhs: (function_definition) operator: "->" rhs: (identifier) @name) @definition.function)

; the same inside a top-level if / else block
(program (if_statement consequence: (braced_expression (binary_operator lhs: (identifier) @name operator: ["<-" "=" "<<-"] rhs: (function_definition)) @definition.function)))

; setClass("Name", ...), setRefClass("Name", ...), R6Class("Name", ...), with or without "X <-"
(program (call function: (identifier) @_c arguments: (arguments . (argument value: (string (string_content) @name)))) @definition.class
  (#any-of? @_c "setClass" "setRefClass" "R6Class"))
(program (binary_operator operator: ["<-" "=" "<<-"] rhs: (call function: (identifier) @_c arguments: (arguments . (argument value: (string (string_content) @name))))) @definition.class
  (#any-of? @_c "setClass" "setRefClass" "R6Class"))

; setGeneric("name", ...) is a generic function; setMethod("name", signature, ...) a method
(program (call function: (identifier) @_c arguments: (arguments . (argument value: (string (string_content) @name)))) @definition.function
  (#eq? @_c "setGeneric"))
(program (call function: (identifier) @_c arguments: (arguments . (argument value: (string (string_content) @name)))) @definition.method
  (#eq? @_c "setMethod"))
; methods of a reference class: setRefClass("X", methods = list(name = function(...) ...)),
; and R6: R6Class("X", public = list(name = function(...) ...)), also private and active
(program (binary_operator operator: ["<-" "=" "<<-"] rhs: (call function: (identifier) @_c arguments: (arguments (argument name: (identifier) @_k value: (call function: (identifier) @_l arguments: (arguments (argument name: (identifier) @name value: (function_definition)) @definition.method))))))
  (#any-of? @_c "setRefClass" "R6Class") (#any-of? @_k "methods" "public" "private" "active") (#eq? @_l "list"))
(program (call function: (identifier) @_c arguments: (arguments (argument name: (identifier) @_k value: (call function: (identifier) @_l arguments: (arguments (argument name: (identifier) @name value: (function_definition)) @definition.method)))))
  (#any-of? @_c "setRefClass" "R6Class") (#any-of? @_k "methods" "public" "private" "active") (#eq? @_l "list"))
`, true)
}
