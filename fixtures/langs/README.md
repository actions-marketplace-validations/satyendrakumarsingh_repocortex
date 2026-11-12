# Language conformance fixtures

One directory per language, named exactly as the grammar is (`go`, `c_sharp`,
`objc`). Each holds source files written to exercise every way that language
declares something, and an `expected.tsv` that lists every definition those
files contain.

`TestLanguageConformance` in `internal/parse` parses each file with that
language's parser and compares what it extracts with the list. The two must be
equal: a definition in the list that is not extracted fails the test, and so
does a symbol that is extracted and not in the list. The fixture is the
specification of what "supported" means for the language.

## `expected.tsv`

Tab-separated, one definition per line:

```
file	line	kind	name
```

- `file` is relative to the language directory.
- `line` is the 1-based line the declaration starts on.
- `kind` is one of `function`, `method`, `class`, `interface`, `type_alias`,
  `package`, `constant`, `variable`, `field`. Structs, enums and unions are
  `class`; traits and protocols are `interface`; modules and namespaces are
  `package`.
- Lines starting with `#` and blank lines are ignored.

## Writing a fixture

Write the source by hand; do not copy files from other projects. Cover each
declaration form the language has, including the awkward ones: a method on a
type, a nested type, a generic, an exported and an unexported name, a
multi-line signature, a declaration after a construct that has confused the
grammar before. Include things that must NOT be extracted (a call, a local
variable, a type used in a parameter list) so over-capture fails too.

List only what a developer would expect a code index to hold. A local variable
is not a definition. Whether a top-level constant or a struct field is one
depends on the language; decide, write the decision as a comment at the top of
`expected.tsv`, and be consistent.

A language may also carry `KNOWN_GAPS.md`: declaration forms that are not yet
extracted, each with the reason. A gap listed there is not in `expected.tsv`.
The file exists so that an incomplete language says so in the tree.
