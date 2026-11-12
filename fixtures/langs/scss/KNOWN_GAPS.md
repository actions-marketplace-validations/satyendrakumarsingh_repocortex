# SCSS known gaps

## The statement after a one-line map literal is lost

A map written on one line breaks the grammar, and the next statement is
swallowed into the map's declaration:

```
$s: (small: 480px);      ->  (declaration (property_name) (ERROR (plain_value) (ERROR) (ERROR)) (integer_value (unit)))
$d: 4px;                     (this declaration is inside the one above and has no node of its own)
```

`$s` is extracted; the `$d`, `@mixin` or `:root { --x }` that follows it is
not. The same map written over several lines parses cleanly. Nothing in the
tree names the lost statement, so no query can recover it.

## `@use ... as`, `!default` on some values, `@forward`

These produce `ERROR` nodes. They hold no definition themselves; a file full
of them is reported as partial and a later definition can be lost with them.
`$name: "text" !default;` is extracted.

## `@font-face` family

The SCSS grammar gives a quoted string no `string_content` child, so the
family name cannot be taken without its quotes. `@font-face` is not extracted
(CSS extracts it).

## Not captured, by choice

Same as CSS: classes and ids in descendant, compound, listed, pseudo or
nested selectors, rule sets nested in a block, `$variables` inside a mixin,
function, rule or `@if`/`@each` block, and `@include`/`@extend` uses.
