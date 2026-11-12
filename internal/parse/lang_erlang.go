package parse

import ts "github.com/odvcencio/gotreesitter"

// Erlang's definitions come from the grammar's own inferred query, which is
// not replaced here. What this file adds is the clause fold: the grammar makes
// each `f(0) -> ...;` clause its own fun_decl, so a function of three clauses
// arrived as three symbols. Consecutive clauses of one name and arity are now
// one symbol; `f/1` and `f/2` stay apart. See clausemerge.go.
func init() {
	registerDefinitions("erlang", erlangDefinitions, true)
	registerClauseMerge("erlang", erlangClauseArity)
}

// erlangClauseArity counts the arguments of the clause the name starts:
// the named children of its expr_args.
func erlangClauseArity(name *ts.Node, lang *ts.Language) string {
	clause := name.Parent()
	if clause == nil || clause.Type(lang) != "function_clause" {
		return ""
	}
	return namedChildCount(clause, "expr_args", lang)
}

// What the index holds, see fixtures/langs/erlang/expected.tsv:
//
//   - a function is a function, its clauses folded per name and arity (above);
//   - -module(name). is a package;
//   - -type and -opaque are a type alias, -record(name, {...}) is a class;
//   - -define(NAME, value). is a constant, and -define(NAME(Args), body). is a
//     function: a macro that takes arguments is called, one that does not is a
//     value.
//
// Not definitions: -spec (it describes a function defined elsewhere),
// -callback, -export, -import, -behaviour, record fields, the -ifdef family.
const erlangDefinitions = `
(fun_decl (function_clause (atom) @name)) @definition.function
(module_attribute . (atom) @name) @definition.package
(type_alias (type_name . (atom) @name)) @definition.type_alias
(opaque (type_name . (atom) @name)) @definition.type_alias
(record_decl . (atom) @name) @definition.class
(pp_define (macro_lhs . [(var) (atom)] @name .)) @definition.constant
(pp_define (macro_lhs . [(var) (atom)] @name . (var_args))) @definition.function
`
