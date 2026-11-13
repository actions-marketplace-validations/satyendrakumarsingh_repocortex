# Groovy: known gaps

The Groovy grammar reads only part of the language. It reports a partial parse
for many ordinary files, and what follows a construct it cannot read is
sometimes lost or attached to the wrong parent. Nothing can be captured from
inside an `ERROR` node.

## A class or method with an annotation starts on the annotation line

`@CompileStatic class Inventory`, and `@Override` before a method, are part of
the declaration node, so the symbol starts at the annotation. Name and kind are
right.

```
(class_definition (annotation (identifier)) (identifier) ...)
(function_definition (annotation (identifier)) (identifier) (identifier) (parameter_list) (closure))
```

## A private/public constructor with two or more parameters

`private Inventory(String name, int size) { this(name) }` (Inventory.groovy
line 24) is read as a declaration with an `ERROR` inside, a second declaration
for `int size`, and a stray closure. The constructor name is in the `ERROR`
only for some shapes, so it is not captured. The same holds for a method with
modifiers and no return type whose parameter list has two typed parameters
(`private static writeStackToFile(File f, Helper h) {` in a real file).

```
(declaration (access_modifier) (ERROR (identifier)) (identifier) (identifier))
(ERROR) (declaration (builtintype) (identifier)) (ERROR) (closure (function_call ...))
```

## A one-line doc comment swallows the rest of the file

```groovy
class Flat implements Marker {
    protected final Map<String, Item> items = new LinkedHashMap<String, Item>()
    /** a one-line doc comment */
    int size
```

The grammar's doc-comment rule takes `a one-line doc comment */` as the first
line of the comment, so the comment never closes. Everything after it up to
the next `*/` is comment text and the rest of the file is read as that
comment's content, ending with a missing `*/` at the end of the file:

```
(groovy_doc (groovy_package ...) "class" (ERROR (identifier) (identifier) (identifier))
  (identifier) "{" (declaration ...) "/**" (first_line) (MISSING "*/"))
```

Nothing after the comment is a node, so nothing after it can be extracted. The
class header, which is loose under the `groovy_doc`, is recovered (the pattern
cannot match in a tree that was read properly, as a doc comment cannot hold a
`class` keyword). In grails-core's `DefaultConstrainedProperty.groovy` that is
1 of 50 definitions: the one-line `/** ... */` at line 73 hides the other 49.
Measured with the comment's second `*` replaced by a space before parsing (a
mask like `cmacro.go`'s, which this tree does not have for Groovy), the same
file yields all 50. Multi-line doc comments are not affected.

## A class closed early by the grammar

A closure parameter with a dotted type and a default value as the value of a
map entry whose key is a parenthesised expression closes the class at the end
of that entry:

```groovy
Map handlers = [
    (Op.Equals): { Op.Equals eq, Prop prop, Closure fn = null, boolean onValue = false ->
        ...
    }
]
```

```
(source_file (class_definition) (ERROR "]") (function_definition ...) (declaration (access_modifier)) (function_call ...) ...)
```

The members after it are siblings at the top of the file, after an `ERROR`.
They are recorded as methods when a `class_definition` and an `ERROR` precede
them (`function_definition`, `function_declaration`, a modifiers-only
declaration followed by a call with a closure, and `modifiers name(a, b) {`);
the same shapes in a file without that `ERROR` are script functions and keep
their kind. A script function written after a class that was closed by a stray
`}` is indistinguishable from a member and is recorded as a method.

## Enums and traits

`enum` and `trait` are not nodes of their own: the grammar produces the
sequence identifier, identifier, closure (or a declaration when `implements`
follows). The name is captured from that sequence. Not captured:

- enum constants (`ACTIVE('A'), INACTIVE('I')` is read as a parameter list
  inside an `ERROR`);
- the constructor of an enum when no field comes before it (one that follows
  a field is read as a declaration plus a call and is recorded), and methods of
  an enum that has an `implements` clause when its body is read as `ERROR` plus
  a nested closure. A trait with type parameters or `implements`
  (`trait Repo<D> implements Base<D> {`), with static methods whose return type
  is a bare type variable or that have their own type parameters
  (`static D create() {`, `static <T> T with(...) {`), is recorded.

## `static` after a dot ends the class early

`metaClazz.static.invokeMethod = ...` is not valid for the grammar; the class
closes at that line and every later member is read at the top level of the
file, so it is recorded as a `function` instead of a `method` (seen in
`InterceptingGCL.groovy`), and members that follow an `ERROR` can be missed.

## Not recorded by design

Plain fields and script variables (`def threshold = 10`): a misread
constructor leaves its parameters behind as declarations, so a field pattern
would record parameters. Closure-valued fields and variables are recorded.
Locals, parameters, calls, Gradle configuration blocks, `tasks.named`, and
Jenkins pipeline `stage` steps.
