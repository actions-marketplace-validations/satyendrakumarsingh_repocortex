package parse

import "strings"

// OCaml. What the index holds, see fixtures/langs/ocaml/expected.tsv.
//
// The inferred query names every `let` binding a function wherever it is, so a
// `let x = ... in` inside a function body, and every binding in a local
// `let module`, was indexed as a definition (about one symbol in six was one of
// those). It also knows no operator definitions, no `external`, no `val`
// specification in an .mli, and gives every value the kind function. It is
// replaced.
//
// A definition is an item of the file or of a module or signature written in
// the file, to a depth of four modules:
//
//   - let and let rec (and every `and` binding) is a function when it has
//     parameters or its body is `fun` or `function`, and a variable otherwise.
//     A parenthesised operator (`let ( +++ ) a b`) is named by the operator.
//     `let () = ...`, `let _ = ...` and tuple patterns declare no name;
//   - val in an .mli or a signature is a function when its type is a function
//     type and a variable otherwise;
//   - external is a function;
//   - type is a class when it is a variant, a record, abstract or extensible,
//     and a type_alias when it is an alias of another type expression;
//   - exception is a class;
//   - module is a package, module type an interface;
//   - class is a class, class type an interface, a method of either is a
//     method and an instance variable (`val`) is a field.
//
// Not definitions: a let inside an expression (`let x = ... in`), a module
// declared by `let module`, the constructors of a variant, the fields of a
// record, and open/include.
func init() {
	registerDefinitions("ocaml", ocamlDefinitions(), true)
}

// ocamlItems are the patterns for one declaration inside a container: the
// compilation unit, a structure or a signature. They are written against
// whatever the container is and wrapped by ocamlContainers.
func ocamlItems() []string {
	opName := `[(value_name) @name (parenthesized_operator (_) @name)]`
	// A let binding with no parameter, whose body is not `fun` or `function`.
	// What may sit between the name and the body is given by between.
	variable := func(name, between string) string {
		return `(value_definition (let_binding pattern: ` + name + ` ` + between + ` body: (_) @_body
  (#not-match? @_body "^(fun|function)\\b")) @definition.variable)`
	}
	return []string{
		// let / let rec / and: a function when it has parameters
		`(value_definition (let_binding pattern: ` + opName + ` (parameter)) @definition.function)`,
		// or a function-valued body
		`(value_definition (let_binding pattern: ` + opName + ` body: [(fun_expression) (function_expression)]) @definition.function)`,
		// a variable otherwise: nothing between the name and the body but a
		// type annotation and comments
		variable(opName, `.`),
		variable(opName, `. (comment) .`),
		variable(opName, `. type: (_) .`),
		variable(opName, `. type: (_) . (comment) .`),

		// val in an interface or signature
		`(value_specification ` + opName + ` type: [(function_type) (polymorphic_type type: (function_type))]) @definition.function`,
		`(value_specification ` + opName + ` type: (_) @_type
  (#not-match? @_type "->")) @definition.variable`,
		`(value_specification ` + opName + ` type: [(constructed_type) (tuple_type)]) @definition.variable`,

		`(external ` + opName + `) @definition.function`,

		// types
		`(type_definition (type_binding name: (type_constructor) @name body: [(variant_declaration) (record_declaration)]) @definition.class)`,
		`(type_definition (type_binding name: (type_constructor) @name .) @definition.class)`,
		`(type_definition (type_binding name: (type_constructor) @name body: [
  (type_constructor_path) (constructed_type) (function_type) (tuple_type)
  (parenthesized_type) (aliased_type) (polymorphic_variant_type) (object_type)
  (hash_type) (package_type) (type_variable) (local_open_type) (any_type)]) @definition.type_alias)`,
		`(exception_definition (constructor_declaration . (constructor_name) @name)) @definition.class`,

		// modules
		`(module_definition (module_binding . (module_name) @name) @definition.module)`,
		`(module_type_definition . (module_type_name) @name) @definition.interface`,

		// classes and their members
		`(class_definition (class_binding . (class_name) @name) @definition.class)`,
		`(class_type_definition (class_type_binding . (class_type_name) @name) @definition.interface)`,
		`(class_definition (class_binding body: (object_expression (method_definition (method_name) @name) @definition.method)))`,
		`(class_definition (class_binding body: (object_expression (instance_variable_definition (instance_variable_name) @name) @definition.field)))`,
		`(class_definition (class_binding class_type: (class_body_type (method_specification (method_name) @name) @definition.method)))`,
		`(class_definition (class_binding class_type: (class_function_type codomain: (class_body_type (method_specification (method_name) @name) @definition.method))))`,
		`(class_type_definition (class_type_binding body: (class_body_type (method_specification (method_name) @name) @definition.method)))`,
	}
}

// ocamlContainers are the places an item can sit and still be a definition of
// the file: the file itself, and the structure or signature of a module that
// is itself declared at the file level or inside such a module. `let module`
// is an expression and so is not among them. HOLE stands for what the
// container holds.
var ocamlContainers = []string{
	`(module_definition (module_binding body: (structure HOLE)))`,
	`(module_definition (module_binding module_type: (signature HOLE)))`,
	`(module_type_definition body: (signature HOLE))`,
}

func ocamlDefinitions() string {
	// One pattern per item and container, not one alternation of items: when
	// several siblings of a pattern's root match (the bindings joined by `and`,
	// the methods of one class) an alternation reports only the first.
	items := ocamlItems()

	// Containers nested up to four deep: file > module > module > module > item.
	wrappers := []string{"HOLE"}
	level := []string{"HOLE"}
	for depth := 0; depth < 3; depth++ {
		var next []string
		for _, inner := range level {
			for _, c := range ocamlContainers {
				next = append(next, strings.Replace(inner, "HOLE", c, 1))
			}
		}
		wrappers = append(wrappers, next...)
		level = next
	}

	var out []string
	for _, w := range wrappers {
		for _, item := range items {
			out = append(out, "(compilation_unit "+strings.Replace(w, "HOLE", item, 1)+")")
		}
	}
	return strings.Join(out, "\n")
}
