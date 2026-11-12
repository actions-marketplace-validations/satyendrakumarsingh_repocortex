# Authzed: known gaps

## `relation name: type` has a colon in its name

The grammar lexes the identifier of a relation together with the colon that
follows it. In the usual form, `relation admin: user`, the symbol would be named
`"admin:"`. The query language cannot trim a capture, so the relation pattern
accepts only a name made of identifier characters (`#match?`). That keeps the
form with a space before the colon, `relation viewer : user`, and leaves out the
usual one. In `fixtures/langs/authzed/sample.zed` every relation written as
`name: type` is this case.

```
relation admin: user | team#member
  (relation (relation_literal) (identifier [35,41] "admin:") (rel_expression ...))
relation viewer : user
  (relation (relation_literal) (identifier [74,80] "viewer") (rel_expression ": user"))
```

Trimming the `@name` text in `tagTree` (`internal/parse/generic.go`) of a trailing
`:` would allow the guard to be removed and every relation to be listed.

## Caveats with an `ipaddress` parameter

`ipaddress` is a SpiceDB caveat type, but this grammar does not know it. A
caveat that declares one is a syntax error, and the whole file is lost with it:
the caveat and every definition in the same file, before and after it, are not
extracted (see `network.zed`). Every other caveat type (`any`, `bool`, `string`,
`int`, `uint`, `double`, `bytes`, `duration`, `timestamp`, `list<T>`, `map<T>`)
parses.

```
caveat in_network(ip ipaddress) { ... }   ->   (ERROR (caveat_literal) (identifier) ...)
```

## Any name with a number in it

An identifier that contains a numeral (`server1`, `hcs_de_p43_case`, `r1`) is
a syntax error, whether it is declared or only used. Error recovery then drops
the declaration around it and often the declarations that follow (see
`numerals.zed`). In the real schema `oviva.zed` this loses seven definitions and
27 permissions after the first such name.

```
definition server1 { ... }   ->   (ERROR ...)
```

## Not extracted by choice

Subject types, `with` caveat references, and names used inside permission
expressions. Prefixed definition names such as `project/folder` are extracted
whole.
