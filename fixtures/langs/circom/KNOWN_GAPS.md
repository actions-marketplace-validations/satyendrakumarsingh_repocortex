# Circom: known gaps

## `template Name(...)` on one line has a space in its name

The grammar lexes the identifier after `template` together with the whitespace
in front of it. With the name on the same line as the keyword the symbol would
be named `" Num2Bits"` (leading space, or several spaces if the source has
several). The query language cannot trim a capture, so the template pattern
accepts only a name that starts with a letter (`#match?`). That keeps the
forms whose name is clean (`template parallel X`, `template custom X`, and the
name on the next line) and leaves out the common form. In `fixtures/langs/circom/sample.circom` the
templates `Num2Bits`, `Switcher`, `Wrapper` and `Tagged` are this case.

```
template   B() {}   ->   (template_definition "template"
                            (identifier  start=8 end=12  text="   B") ...)
```

A parallel `parallel`/`custom` template is clean because the modifier is a
token of its own between the keyword and the name. Trimming the `@name` text in
`tagTree` (`internal/parse/generic.go`) would allow the guard to be removed and
the four templates above to be listed.

## `bus` declarations

`bus Point() { ... }` (Circom 2.2) is a syntax error in this grammar and the
declaration is not extracted.

```
(ERROR (ERROR) (ERROR) (ERROR))
```

## Not extracted by choice

Signals (inputs, outputs and intermediates), variables and `component x = T()`
instances: they are locals of a template.
