# Swift: known gaps

## A declaration with attributes starts on the attribute line

`@inlinable` / `@available(...)` / `@discardableResult` before a `func`, `var`,
`init` or `subscript` is part of the grammar's `modifiers` node, which is the
first child of the declaration, so the symbol's start line is the attribute's
line, not the line of `func`. The name and kind are right. Types are not
affected (`@propertyWrapper struct X` starts at `struct`). The range of a
symbol must be the whole declaration (callers inside the body are attributed
by it), so the capture cannot be narrowed to the keyword.

```
(function_declaration (modifiers (attribute ...) (visibility_modifier)) (simple_identifier) ...)
```

## The grammar loses a whole region on newer syntax

When the grammar cannot read a construct it emits `ERROR` nodes and often
swallows the declarations around it; nothing can be captured from inside an
`ERROR`. The parse status of such a file is `partial`. Seen in real code:

- typed throws, `throws(Failure)` after a parameter list;
- `nonisolated(nonsending)` as a modifier;
- an attribute applied to a parenthesised function type in a parameter, as
  in `_ factory: @escaping @Sendable (String) -> any LogHandler`, when it is
  followed by more such parameters;
- an `#if` / `#elseif` chain that declares the same property with a multi-line
  initialiser in each branch (only one branch is read).

```
(source_file (comment)... (ERROR (modifiers (visibility_modifier)) (simple_identifier) (inheritance_specifier ...) (property_declaration ...)))
```

Declarations after such a region in the same file are usually lost too. The
fix belongs in the grammar, not in a query.

## Not extracted by design

Accessors (`get`, `set`, `willSet`, `didSet`), the members of an enum case's
associated tuple, `macro` declarations (the grammar parses `macro f() = ...` as
an error), `precedencegroup`, and `import`.
