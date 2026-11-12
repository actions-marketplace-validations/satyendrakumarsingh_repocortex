# C++: known gaps

## Out-of-line destructors and operators

```cpp
Comparator::~Comparator() {}
bool Env::operator==(const Env&) const { return true; }
```

The name is a `destructor_name` or an `operator_name` node, not an
`identifier`, so the definition is not extracted.

## `using` aliases

`using Callback = int (*)(int);` is an `alias_declaration`; the query has no
pattern for it. `typedef` is extracted.

## Declarations

A method declared in a class and defined elsewhere is recorded once, where it
is defined. A prototype without a body is never a definition here.
