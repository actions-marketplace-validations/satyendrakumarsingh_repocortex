package parse

import ts "github.com/odvcencio/gotreesitter"

// Erlang's definitions come from the grammar's own inferred query, which is
// not replaced here. What this file adds is the clause fold: the grammar makes
// each `f(0) -> ...;` clause its own fun_decl, so a function of three clauses
// arrived as three symbols. Consecutive clauses of one name and arity are now
// one symbol; `f/1` and `f/2` stay apart. See clausemerge.go.
func init() {
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
