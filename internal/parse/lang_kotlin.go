package parse

import ts "github.com/odvcencio/gotreesitter"

// Kotlin calls and the types they reach. The inferred tags query for Kotlin
// finds definitions and no call at all, so every "who calls this" answer for
// Kotlin was empty.
//
// A call is a call_expression whose callee is either a bare identifier or a
// navigation_expression ending in a navigation_suffix:
//
//	f(x), Widget(3)        a bare name; Widget(3) constructs when Widget is a
//	                       class, which the resolver decides (CallConstructs)
//	x.m(), x?.m()          a call on x; shape.go reads `?.` already
//	this.m(), super.m()    `self` and `super`
//	A.m()                  a call on the name A: a class's companion object, or
//	                       an object, whose members are indexed under A
//	"s".trim(), f(1).m()   a call on an expression
//	x.ext()                an extension function is called like a method; the
//	                       call is extracted, and bound only where one function
//	                       of that name is the unique candidate
//
// Not extracted: infix calls (`a to b`), operator calls (`a + b`), calls
// through a function-typed value (`callback(x)` is a bare call whose callee is
// a local, which the resolver does not bind), and `object : Base() {}`.
//
// Two definition gaps are closed because calls cannot bind without them: a
// class declared with no body (`class A(val x: Int)`, `data class P(...)`) was
// not indexed at all, and an `object` was indexed as a FUNCTION by the
// catch-all kind, which made `Registry.lookup()` a call on a callable. It is
// now a variable: a singleton value, which is neither called nor constructed,
// so it competes with no class of the same name. (A module was tried and made
// `extends A` ambiguous wherever a class has a companion of its name.)
func init() {
	registerDefinitions("kotlin", `
(call_expression . (simple_identifier) @name) @reference.call
(call_expression (navigation_expression (navigation_suffix (simple_identifier) @name .))) @reference.call
(class_declaration "class" (type_identifier) @name) @definition.class
(object_declaration (type_identifier) @name) @definition.variable
`, false)
}

// kotlinField answers the field lookups the scope walk (shape.go) makes of a
// declaration, for a grammar that defines no fields at all: the Kotlin grammar
// names no child, so a field lookup of "name" is nil for every class and
// function and the walk saw no class scope in Kotlin. Every method then
// qualified as a free function, and `this.m()`, `super.m()` and `A.m()` could
// not find the type they were made on.
//
// Only "name" and "body", and only for the declarations that open a scope. The
// name is the first identifier among the node's own children (an extension
// function's receiver type is a node of its own, so `fun String.ext()` names
// `ext`), and the body is the block of members.
func (w *shWalker) kotlinField(n *ts.Node, name string) *ts.Node {
	var nameType string
	var bodyTypes []string
	switch n.Type(w.lang) {
	case "class_declaration", "object_declaration":
		nameType, bodyTypes = "type_identifier", []string{"class_body", "enum_class_body"}
	case "function_declaration":
		nameType, bodyTypes = "simple_identifier", []string{"function_body"}
	default:
		return nil
	}
	for i := 0; i < n.ChildCount(); i++ {
		c := n.Child(i)
		if c == nil || !c.IsNamed() {
			continue
		}
		ct := c.Type(w.lang)
		if name == "name" && ct == nameType {
			return c
		}
		if name == "body" {
			for _, b := range bodyTypes {
				if ct == b {
					return c
				}
			}
		}
	}
	return nil
}
