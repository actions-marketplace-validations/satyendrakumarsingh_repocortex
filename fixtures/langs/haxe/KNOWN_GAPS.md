# Haxe: forms that are not extracted, and why

The Haxe grammar is the limiting factor. It has no node for an enum or an
abstract, no name node for a constructor, and when it meets a construct it
cannot read it does not recover locally: the parse of everything after it, up
to the whole file, changes shape (`(ERROR ...)` wrapping, class members left
floating at module level). On the five real files used to measure this
(hscript), most of the loss is of that kind. The query cannot repair a tree
that does not contain the name.

Constructs that break the parse, each seen in real code:

- **Object-literal typedefs and enum constructors with arguments** in the same
  file as other declarations: `typedef Argument = { name : String, ?t : CType };`
  and `enum Expr { EConst( c : Const ); ... }`. A file with enums, `#if`
  blocks and `{> Base, ...}` typedefs (hscript's `Expr.hx`) comes back as one
  ERROR node; its `enum` and `class` declarations are no longer separate
  nodes, only the two block-form typedefs are found. The same enums parse
  correctly when the file is cut short before the first object-literal
  typedef, which is how the fixture keeps them (`types.hx`).
- **A class header with `extends` after other errors**: the class name is not
  in the tree (`class Colorizer extends Parser {` becomes
  `(ERROR (type_name) (variable_declaration ...) ...)`), so the class and its
  fields are not extracted. Methods inside are found, as methods.
- **Members of a class whose closing brace the parser lost** are left as
  `function_declaration` children of `module`. They are recorded as `function`
  rather than `method`, since a function at module level is also legal Haxe
  and the two cannot be told apart. A method of an `abstract` is in the same
  position and is also recorded as `function`.
- **Functions after a `try`/`catch` or `switch` with patterns** in the same
  class may be lost entirely (`executeScriptFile` in hscript's `RunScript.hx`).
- **Constructors**: `function new` has no identifier node; it is read from the
  anonymous `new` token, and only inside a class body.
- **Expression-bodied functions** (`function f():Int return 1;`), `macro`
  functions and `enum abstract` are not read: the parse breaks around them.
  Nothing is recovered from these.
- **`extern interface` with metadata above it** (`@:autoBuild(...)`): the
  interface is lost to an ERROR node (hscript `Live.hx`).

How the extraction recovers what it can: an enum, from the `enum` keyword node
followed by the name; an abstract, from an ERROR node whose text starts with
`abstract`; a class or interface whose own node is missing, from an ERROR node
whose text starts with `class` or `interface`. All three select on the text, so
they match nothing else.
