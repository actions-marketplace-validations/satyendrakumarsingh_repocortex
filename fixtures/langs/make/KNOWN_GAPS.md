# Make: forms not extracted

- **Targets spelled with a variable reference.** `$(NAME): main.go` and
  `$(BIN_DIR)/%.gz:` have a `variable_reference` or `concatenation` node in
  `targets` instead of a `word`. The name is only known after expansion, so no
  symbol is recorded.
- **Pattern rules** (`%.o: %.c`, `check-%:`) and **suffix rules** (`.c.o:`) are
  not definitions: there is no one name to look up. Special targets (`.PHONY`,
  `.SUFFIXES`, `.DEFAULT_GOAL`) are make's own and are skipped by their leading
  dot.
- **Variables defined only by `+=`.** An append is never recorded, so a
  variable that is first created with `+=` (legal, and rare) is missing.
- **Variables created by `$(eval ...)`, `$(foreach ...)` or `define`d
  macros** are not seen; the grammar parses these as function calls.
- **Make's own settings** (`SHELL`, `MAKEFLAGS`, `MAKEFILES`, `VPATH`, and
  names beginning with a dot) are not recorded.
- **Target-specific variables** (`app: CFLAGS = -O0`) are scoped to a target
  and are not recorded; the grammar gives them a list of targets before the
  variable name.
- **Non-GNU makefiles with space-indented recipes** (Watcom `wmake`, as in
  `spaces.mk`) are misread: the grammar takes a recipe line such as
  `  wlib -q -n libttf.lib` for a rule and records `wlib`, and the real target
  above it (`all` in `spaces.mk`, which has a `.symbolic` prerequisite and a
  `&` continuation before it) can be lost. Names that begin with `-`, `@` or
  `+` or contain a backslash cannot be targets and are filtered out; a plain
  word like `wlib` cannot be told from a real target.
- **Repeated names.** `all`, `clean`, `test` and `install` appear in nearly
  every Makefile; each is recorded once per file.
