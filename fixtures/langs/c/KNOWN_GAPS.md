# C: known gaps

## File-level variables, enums and unions

`static counter_t total;`, `enum color { RED, GREEN };` and `union U { ... };`
are not extracted. The definition query predates this fixture and names
functions, structs and typedefs only; the fixture records that rather than
widening it in a change about macros.

## Macros the mask does not recognise

Declarations behind a macro are read after the macro is masked (see
`internal/parse/cmacro.go`). A macro written in mixed case (`Py_LOCAL_INLINE(int)`)
or one that follows the parameter list (`void f(void) NORETURN;`) is not a
candidate, and the declaration is read as the grammar sees it.

## Conditional compilation

An `#if` group that cuts a declaration apart (see `conditional.c`, a parameter
list with a different tail in each branch) is read with one branch: the first
that is not `#if 0`, with two groups that test one macro read in the same
world. `extern "C" {` guards are kept, both halves together. A definition that
exists only in a branch that is not kept is not extracted, and the number of
branches blanked is in the file's stored parse text.
