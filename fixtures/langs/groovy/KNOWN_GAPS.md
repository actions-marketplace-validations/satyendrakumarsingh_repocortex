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

## Enums and traits

`enum` and `trait` are not nodes of their own: the grammar produces the
sequence identifier, identifier, closure (or a declaration when `implements`
follows). The name is captured from that sequence. Not captured:

- enum constants (`ACTIVE('A'), INACTIVE('I')` is read as a parameter list
  inside an `ERROR`);
- the constructor of an enum, and methods of an enum or trait that has an
  `implements` clause (the body is read as `ERROR` plus a nested closure).

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
