# Elixir: known gaps

## One symbol per clause head, not per function and arity

`def handle_cast({:push, item}, state)` and `def handle_cast(:clear, _state)`
are two symbols, as is a bodiless head such as `def floor(number, precision
\\ 0)`. A query can capture a name but not the arity, and cannot tell a first
clause from a later one, so the clauses of a function cannot be collapsed
here. A module with many clauses of one function has that many symbols of the
same name, at different lines.

## A function whose name is generated

`def unquote(:"name_#{name}")(), do: ...` declares functions with computed
names; the tree has a call as the head, no identifier:

```
(call (identifier) (arguments (call (call (identifier) (arguments ...)) (arguments))) ...)
```

Nothing is recorded for it. (`defp power_of_10(unquote(x))`, where only the
arguments are generated, is recorded.)

## Declarations with no name of their own

`defstruct` and `defexception` are not recorded: the struct is the module that
holds it, which is already a `package`. A `defimpl` is recorded under the name
of the protocol it implements (as a `class`), because the implementation
module's name (`Protocol.Type`) is not a node.

## Not recorded by design

`@spec`, `@doc`, `@moduledoc`, `@impl`, `@behaviour` and the other attributes
the compiler reads; `use`, `import`, `alias`, `require`; ExUnit `test`,
`describe` and `setup` blocks; operator definitions (`def a <+> b`); anonymous
functions.
