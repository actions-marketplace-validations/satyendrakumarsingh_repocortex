package parse

import (
	"strings"

	ts "github.com/odvcencio/gotreesitter"
)

// Arity for Kotlin, Scala, C# and C++.
//
// The same contract as Go, Java and TypeScript (see queries.go): a count is
// published only when it is exact. A parameter list with a default value or a
// variable number of parameters, and a call spreading a collection, publish no
// count at all (model.ArityUnknown), because the Arity type has one number and
// "exactly N" and "at least N" are different facts. A resolver that wants the
// second needs a second field on model.Symbol; until it exists the safe answer
// is the same as for TypeScript's optional parameter: unknown.
//
// What is counted, per language:
//
//	Kotlin  parameters: the `parameter` nodes of a function. A default value
//	        (the grammar writes it as a sibling of the parameter) or `vararg`
//	        makes the count unknown.
//	        arguments: the value arguments of the call's parenthesised list, a
//	        named argument (`f(b = 1)`) as one, a trailing lambda (`f(1) { }`,
//	        `f { }`) as one more, because it fills the last parameter. A
//	        spread (`f(*xs)`) makes the count unknown.
//	Scala   parameters: the first parameter list only, `def f(a: A)(b: B)` is
//	        1, which is the list overload resolution looks at first. A default
//	        value or a repeated parameter (`xs: Int*`) makes the count unknown.
//	        arguments: the first argument list of the call, `f(1)(2)` is 1 for
//	        the call of f. A call written with a block (`f { x => x }`) and an
//	        infix call (`a f b`) carry no count. A `xs: _*` spread is unknown.
//	C#      parameters: the `parameter` nodes. A default value, `params`, and the
//	        `this` of an extension method (the call does not pass it) make the
//	        count unknown.
//	        arguments: the arguments of the call; named, ref, out and in as one
//	        each.
//	C++     parameters: the parameter declarations, `f(void)` as none. A default
//	        value, `...` and a parameter pack make the count unknown.
//	        arguments: the arguments of the call; a pack expansion makes the
//	        count unknown.
//
// The match between a declaration and its count is by line and name, as for
// every other language (see callSite): a declaration whose name is not on the
// line its range starts on (a C++ return type on the line above the name)
// carries no count.
func init() {
	paramCountQueries["kotlin"] = `
(function_declaration (simple_identifier) @name (function_value_parameters) @params)
`
	argCountQueries["kotlin"] = `
(call_expression . (simple_identifier) @name (call_suffix) @args)
(call_expression (navigation_expression (navigation_suffix (simple_identifier) @name .)) (call_suffix) @args)
`
	arityDialects["kotlin"] = arityDialect{
		items:    map[string]bool{"parameter": true, "value_argument": true, "annotated_lambda": true},
		skip:     map[string]bool{"parameter_modifiers": true, "type_arguments": true},
		flatten:  map[string]bool{"value_arguments": true},
		variadic: map[string]bool{"spread_expression": true},
		opaque: func(n *ts.Node, lang *ts.Language, src []byte) bool {
			return n.Type(lang) == "parameter_modifiers" && strings.Contains(n.Text(src), "vararg")
		},
	}

	paramCountQueries["scala"] = `
(function_definition name: (identifier) @name . (parameters) @params)
(function_definition (operator_identifier) @name . (parameters) @params)
(function_definition name: (identifier) @name . (type_parameters) . (parameters) @params)
(function_definition (operator_identifier) @name . (type_parameters) . (parameters) @params)
`
	argCountQueries["scala"] = `
(call_expression function: (identifier) @name arguments: (arguments) @args)
(call_expression function: (field_expression field: (identifier) @name) arguments: (arguments) @args)
(call_expression function: (generic_function function: (identifier) @name) arguments: (arguments) @args)
(call_expression function: (generic_function function: (field_expression field: (identifier) @name)) arguments: (arguments) @args)
`
	arityDialects["scala"] = arityDialect{
		variadic: map[string]bool{"repeated_parameter_type": true, "vararg": true},
		opaque: func(n *ts.Node, lang *ts.Language, _ []byte) bool {
			if n.Type(lang) != "parameter" {
				return false
			}
			for i := 0; i < n.ChildCount(); i++ {
				if n.FieldNameForChild(i, lang) == "default_value" {
					return true
				}
			}
			return false
		},
	}

	paramCountQueries["c_sharp"] = `
(method_declaration name: (identifier) @name parameters: (parameter_list) @params)
(constructor_declaration name: (identifier) @name parameters: (parameter_list) @params)
`
	argCountQueries["c_sharp"] = `
(invocation_expression function: (identifier) @name arguments: (argument_list) @args)
(invocation_expression function: (member_access_expression name: (identifier) @name) arguments: (argument_list) @args)
(invocation_expression function: (generic_name (identifier) @name) arguments: (argument_list) @args)
(invocation_expression function: (member_access_expression name: (generic_name (identifier) @name)) arguments: (argument_list) @args)
(invocation_expression function: (conditional_access_expression (member_binding_expression name: (identifier) @name)) arguments: (argument_list) @args)
`
	arityDialects["c_sharp"] = arityDialect{
		items:  map[string]bool{"parameter": true, "argument": true},
		opaque: csharpOpaqueParameter,
	}

	cppParams := `
(function_declarator declarator: [(identifier) (field_identifier)] @name parameters: (parameter_list) @params)
(function_declarator declarator: (qualified_identifier name: (identifier) @name) parameters: (parameter_list) @params)
(function_declarator declarator: (qualified_identifier name: (qualified_identifier name: (identifier) @name)) parameters: (parameter_list) @params)
(function_declarator declarator: (qualified_identifier name: (qualified_identifier name: (qualified_identifier name: (identifier) @name))) parameters: (parameter_list) @params)
`
	paramCountQueries["cpp"] = cppParams
	argCountQueries["cpp"] = `
(call_expression function: (identifier) @name arguments: (argument_list) @args)
(call_expression function: (field_expression field: (field_identifier) @name) arguments: (argument_list) @args)
(call_expression function: (qualified_identifier name: (identifier) @name) arguments: (argument_list) @args)
(call_expression function: (template_function name: (identifier) @name) arguments: (argument_list) @args)
`
	arityDialects["cpp"] = arityDialect{
		// An argument is any expression, so only the parameter list is closed.
		items:    map[string]bool{"parameter_declaration": true},
		closedIn: map[string]bool{"parameter_list": true},
		variadic: map[string]bool{
			"variadic_parameter_declaration": true,
			"parameter_pack_expansion":       true,
			"...":                            true,
		},
		voidList: func(list *ts.Node, lang *ts.Language, src []byte) bool {
			if list.Type(lang) != "parameter_list" {
				return false
			}
			return strings.Join(strings.Fields(list.Text(src)), "") == "(void)"
		},
	}
}

// csharpOpaqueParameter reports a C# parameter whose declaration cannot be set
// against a call's argument count: one with a default value, which the grammar
// writes as a further child of the parameter, and the `this` parameter of an
// extension method, which the call does not pass.
func csharpOpaqueParameter(n *ts.Node, lang *ts.Language, src []byte) bool {
	if n.Type(lang) != "parameter" {
		return false
	}
	body := 0
	for i := 0; i < n.ChildCount(); i++ {
		c := n.Child(i)
		if c == nil || !c.IsNamed() || c.IsExtra() {
			continue
		}
		switch c.Type(lang) {
		case "modifier":
			if strings.TrimSpace(c.Text(src)) == "this" {
				return true
			}
		case "attribute_list":
		default:
			body++
		}
	}
	// A parameter is a type and a name. Anything more is `= default`.
	return body > 2
}
