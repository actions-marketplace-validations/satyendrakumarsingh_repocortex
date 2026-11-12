# Julia: forms that are not extracted

- **Functors and parenthesised constructors**: `(f::Callable)(x, y) = ...`,
  `function (f::Callable)(x) ... end` and
  `function (LittleSet{T, Tuple{Vararg{T}}} where {T})(data) ... end`. They
  define a method on a type, not a function with a name of its own; the
  closest name is the type, but it sits inside a parenthesised expression of
  several shapes. Fragment:

  ```
  (function_definition (signature (call_expression (parenthesized_expression (typed_expression (identifier) (identifier))) (argument_list (identifier)))))
  ```

  Tagging the type would also tag every `(x::T)(...)` call-like left side, so
  it is left out.
- **Short-form methods inside a function body**: `helper(x) = x + n` is a local
  closure, and in the tree it is the same `assignment` as a top-level method;
  only its position tells them apart, so it is not extracted. A long-form
  `function` is extracted at any depth.
- **Short-form methods in other nesting**: only the top level of a file, module
  or struct, and the branches of an `if` or `@static if` there, are read. A
  method inside `let`, `begin` or a `for` at the top level is not.
- **Enum members**: `@enum Color red green blue` gives the class `Color` only.
- **Non-const globals**: `counter = 0`, `global total = 10`. A script variable
  is not a declaration of anything stable.
- **Definitions built by `@eval`**: `@eval f(x) = ...` and `@eval function ...`
  are read as the short or long form when the macro wraps one directly, but
  names generated in a loop (`for op in (:a, :b) @eval $op(x) = ... end`) are
  interpolated and not extracted.
- **Extensionless Julia scripts** with a `#!` line are not detected as Julia
  (language detection, outside this fixture).
