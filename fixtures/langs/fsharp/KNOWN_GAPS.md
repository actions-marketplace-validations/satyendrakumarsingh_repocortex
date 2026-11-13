# F#: known gaps

F# is whitespace-sensitive and the grammar reads it with an approximation of
the offside rule. On real code it often gives up part-way through a file; what
follows is what that costs and what the query cannot recover.

## Files the grammar reads wrongly

A type with an `internal` or `private` constructor and `as self`
(`type JsonPickleFormatProvider internal (indent, omitHeader) as self =`), a
`new` constructor with attributes on its arguments, or a long multi-line
expression in a class `let` can make the grammar flatten what follows. The
members and the type header are then siblings at file level, with no
`type_definition` around them:

```
(ERROR (namespace ...) (line_comment) (ERROR) (type_name (identifier))
  (primary_constr_args ...) (type_extension_elements (member_defn ...)) ...)
```

or, without an `ERROR` node at all:

```
(file (long_identifier ..) (import_decl ..) (type_definition ..)
  (type_name (identifier)) (primary_constr_args) (type_extension_elements ..))
```

A `type_name` that is a direct child of the file or of an `ERROR` could be a
union, a record, a class or an interface, so it is not extracted: its kind
would be a guess. Members that survive inside `member_defn` are extracted;
members that end up as a bare `property_or_ident` followed by patterns (for
example `member __.CreateWriter (stream, encoding) =` after such a failure) are
not. In Linguist's `JsonSerializer.fs` this loses all but one definition.

A module that follows another one in a file the grammar mangled can lose its
header entirely (`module Bson =` after `module Json =` in `Combinators.fs`),
and a `let` in it can come out as a bare `value_declaration_left` with no
`function_or_value_defn`, which is not extracted.

## A member constraint call stops the parse (`budget_exceeded`)

A statically-resolved member call, the form F# inline builders use to call
`GetAwaiter` on a type chosen at the call site, sends the grammar's parser into
a loop that consumes no token:

```fsharp
let f task =
    let a = (^T: (member GetAwaiter: unit -> ^A) (task))
    a
```

The parse ends there and the file is recorded as `budget_exceeded` ("iteration
limit") with everything after it missing. It is not the iteration budget
running out. The limit is 30 iterations per byte (10,000 at least); this
five-line file stops after 156 of 10,000, and FsToolkit.ErrorHandling's
`TaskOptionCE.fs` (690 lines, limit 825,480) after 3,218 at byte 18,683, the
`let mutable awaiter = (^TaskLike: (member GetAwaiter: ...) (task))` line.
Raising the limit to 50,000,000 stops at the same iteration and byte (measured),
so a per-grammar factor recovers nothing and none was added. The library's
no-token-progress guard (128 consecutive dispatches without consuming a token)
ends the loop and reports it with the iteration-limit reason. The tree up to the
stop is also incomplete, so the namespace and type headers above the stop are
lost with the rest (they come back once the parse reaches the end).

Not extracted, therefore: any definition after the first such call in a file.
Where the call is blanked with same-length text before parsing (a mask in the
spirit of `cmacro.go`, which this change was not allowed to add), the three
FsToolkit files parse to the end and yield 61, 61 and 64 symbols, which is
the number the independent check expected (24, 24 and 23 without it). The
construct is deliberately not in the fixture: a fixture file that stops the
parser would hide everything after it. `TestFSharpMemberConstraintCallStallsTheParser`
pins the behaviour and fails when a library upgrade fixes it.

## Indented namespaces

When the body of `namespace A.B` is indented, the grammar drops the namespace
node and leaves its name as the first `long_identifier` in the file. That is
read as the namespace/module header (kind `package`). It is the only way a
bare `long_identifier` appears there; if a script starts with a lone dotted
name for another reason it would be recorded too.

## Interfaces written as abstract-only classes

`type IShape = abstract Area: unit -> float` has the same tree as a class whose
members are all abstract, so it is a `class`. Only the explicit form
(`interface ... end`) is an `interface`.

## Single-case unions

`type Id = int` is parsed as a union with one case. A field-less union with a
single case and no bar is therefore read as an abbreviation (`type_alias`). A
genuine `type T = A` single-case union is recorded as an alias too.

## Not extracted

`new(...)` additional constructors (no name), class-private `let` bindings,
union cases, record fields, enum members, `let` inside functions, and patterns
that bind several names (`let (a, b) = ...`). Operators and active patterns are
recorded with their punctuation, as written: `(+++)` and `(|Even|Odd|)`.

## Detection

`.fs` files are routed to the Forth grammar, not F#; only `.fsx` and `.fsi`
reach F#. This tree does not change detection.
