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
