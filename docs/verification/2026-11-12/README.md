# Language verification, 12 November 2026

What RepoCortex actually extracts and resolves, language by language, checked
by hand against real public code. This record exists because the figures it
replaces were not true: the project said it parsed 206 languages and resolved
calls at 94 to 100% precision. Neither held up.

Every number here has a denominator. Where a sample is small it says so.

## Contents

- [Summary](#summary)
- [How it was measured](#how-it-was-measured)
- [The 206 grammars](#the-206-grammars)
- [Definition extraction, 72 languages](#definition-extraction-72-languages)
- [Call resolution, 14 languages](#call-resolution-14-languages)
- [Defects found](#defects-found)
- [Limits of this record](#limits-of-this-record)
- [Files](#files)

## Summary

| Claim | Measured |
|---|---|
| 206 languages parsed | 86 grammars can produce a definition. The other 120 are skipped: no symbols, and the file is not text-searchable either |
| Definitions extracted for every parsed language | Of 72 definition-only languages checked: 10 good, 25 usable, 15 poor, 22 broken. Pooled precision 90%, recall 45% |
| 11 languages resolve calls across files, 3 more with types | PHP, Kotlin and Scala extract no references at all. Ruby, C#, Rust and C++ record no inheritance |
| Call precision 94 to 100% without a compiler index | High-confidence edges measured 54% to 100% depending on language. Only C and TSX reached the documented range, each with a caveat |
| Call recall 33 to 66% | 0% to 87% on the files sampled; 12.5% for JavaScript, 22% for C++ |

## How it was measured

The binary was built from commit `3709c38` (before the Python resolver change in
`eed8962`, so the Python figures describe the older resolver).

1. **Every grammar was probed** for a definition query by constructing its
   parser. The result is in `definition-query-probe.tsv`.
2. **A first pass** indexed GitHub Linguist's `samples/` tree (3,443 files) and
   counted files and symbols per detected language (`linguist-first-pass.json`).
3. **Definitions, 72 languages.** For each, three to twelve real files were
   indexed on their own. A reviewer read each file, listed the definitions a
   developer would expect a code index to hold (functions, methods, types,
   modules, named top-level constants), and compared them with the symbols
   extracted. A symbol is correct when a definition of that name exists within
   two lines of the recorded line. Files came from Linguist's samples and, where
   those were thin, from shallow clones of public projects named in each result
   file.
4. **Calls, 14 languages.** One public project per language was indexed and
   resolved. Precision: bound call edges were sampled per confidence tier (about
   20 each, every k-th row by id) and each was checked by reading the call site
   and the target. Recall: every call to a function defined in the repository
   was listed in five source files (about 40 sites) and looked up in the index.
5. **Verdicts** for definitions: good is precision and recall of at least 90%;
   usable is both at least 70%, or a clear majority found with clean precision;
   poor is below that; broken is recall under 10% or nothing extracted from
   valid files.

The reviewers were sub-agents of an AI assistant, not people. Each wrote a JSON
result with its file list, conventions, counts and examples; those files are in
this directory and every sampled edge id is recorded so a check can be repeated.

## The 206 grammars

| Group | Grammars |
|---|---|
| No definition query: files are skipped | 120 |
| Definition query present | 84 |
| Single-file components read through their script block (Vue, Svelte) | 2 |

A skipped file contributes nothing. Searching for a function name that appears
in a Perl file returns no match, because skipped files are not text-indexed.
`repocortex languages` lists all 120 under "parsed: definitions and full-text
search only", which is false for them.

Programming languages among the 120 include Ada, Agda, assembly, AWK, COBOL,
Emacs Lisp, Elm, Fennel, Forth, Haskell, Janet, Jsonnet, Make, Nix, Pascal,
Perl, Prolog, PureScript, Racket, Tcl, Verilog and Wolfram. Formats include
YAML, JSON, TOML, XML, HTML, Markdown, Dockerfile and INI.

## Definition extraction, 72 languages

Pooled over 4,039 expected definitions: 2,141 extracted, 1,929 correct, 212
wrong, 2,200 missed.

| Language | Files | Expected | Extracted | Correct | Wrong | Missed | Precision | Recall | Verdict |
|---|---|---|---|---|---|---|---|---|---|
| apex | 5 | 39 | 36 | 36 | 0 | 3 | 100% | 92% | good |
| cmake | 5 | 17 | 17 | 17 | 0 | 0 | 100% | 100% | good |
| erlang | 7 | 126 | 114 | 114 | 0 | 12 | 100% | 90% | good |
| fish | 7 | 20 | 20 | 20 | 0 | 0 | 100% | 100% | good |
| graphql | 3 | 60 | 64 | 63 | 1 | 4 | 98% | 93% | good |
| jq | 5 | 222 | 221 | 214 | 7 | 8 | 97% | 96% | good |
| matlab | 6 | 10 | 10 | 10 | 0 | 0 | 100% | 100% | good |
| powershell | 5 | 38 | 58 | 58 | 0 | 0 | 100% | 100% | good |
| solidity | 4 | 70 | 67 | 67 | 0 | 3 | 100% | 96% | good |
| thrift | 4 | 85 | 82 | 80 | 2 | 5 | 98% | 94% | good |
| bash | 5 | 50 | 41 | 41 | 0 | 9 | 100% | 82% | usable |
| cairo | 5 | 51 | 29 | 29 | 0 | 22 | 100% | 57% | usable |
| clojure | 5 | 45 | 36 | 36 | 0 | 9 | 100% | 80% | usable |
| commonlisp | 6 | 76 | 57 | 57 | 0 | 19 | 100% | 75% | usable |
| crystal | 4 | 55 | 50 | 43 | 7 | 12 | 86% | 78% | usable |
| cuda | 5 | 21 | 15 | 15 | 0 | 6 | 100% | 71% | usable |
| d | 5 | 124 | 84 | 84 | 0 | 44 | 100% | 65% | usable |
| dart | 6 | 89 | 72 | 72 | 0 | 17 | 100% | 81% | usable |
| elixir | 4 | 103 | 62 | 62 | 0 | 41 | 100% | 60% | usable |
| glsl | 5 | 62 | 34 | 34 | 0 | 28 | 100% | 55% | usable |
| groovy | 5 | 50 | 44 | 40 | 4 | 10 | 91% | 80% | usable |
| hack | 6 | 34 | 26 | 26 | 0 | 8 | 100% | 76% | usable |
| hcl | 5 | 22 | 52 | 21 | 31 | 1 | 40% | 96% | usable |
| hlsl | 5 | 54 | 28 | 28 | 0 | 26 | 100% | 52% | usable |
| mojo | 4 | 40 | 34 | 34 | 0 | 6 | 100% | 85% | usable |
| ocaml | 5 | 48 | 53 | 45 | 8 | 3 | 85% | 94% | usable |
| proto | 5 | 54 | 61 | 61 | 0 | 0 | 100% | 100% | usable |
| scheme | 5 | 45 | 34 | 34 | 0 | 11 | 100% | 76% | usable |
| squirrel | 5 | 33 | 39 | 28 | 11 | 5 | 72% | 85% | usable |
| starlark | 5 | 32 | 24 | 24 | 0 | 8 | 100% | 75% | usable |
| svelte | 9 | 43 | 20 | 20 | 0 | 23 | 100% | 46% | usable |
| swift | 4 | 45 | 32 | 32 | 0 | 13 | 100% | 71% | usable |
| v | 8 | 60 | 54 | 49 | 5 | 11 | 91% | 82% | usable |
| vue | 12 | 71 | 38 | 38 | 0 | 33 | 100% | 54% | usable |
| zig | 5 | 27 | 22 | 22 | 0 | 5 | 100% | 82% | usable |
| arduino | 6 | 44 | 17 | 17 | 0 | 27 | 100% | 39% | poor |
| bitbake | 5 | 11 | 3 | 3 | 0 | 8 | 100% | 27% | poor |
| faust | 6 | 92 | 24 | 24 | 0 | 75 | 100% | 18% | poor |
| hare | 3 | 53 | 15 | 13 | 2 | 40 | 87% | 24% | poor |
| haxe | 5 | 45 | 14 | 14 | 0 | 31 | 100% | 31% | poor |
| julia | 6 | 86 | 19 | 19 | 0 | 67 | 100% | 22% | poor |
| lua | 5 | 58 | 25 | 25 | 0 | 33 | 100% | 43% | poor |
| luau | 5 | 57 | 13 | 8 | 5 | 49 | 62% | 14% | poor |
| move | 5 | 78 | 33 | 33 | 0 | 45 | 100% | 42% | poor |
| nim | 6 | 130 | 16 | 16 | 0 | 114 | 100% | 12% | poor |
| prisma | 5 | 20 | 170 | 49 | 121 | 9 | 29% | 55% | poor |
| sql | 5 | 23 | 24 | 17 | 7 | 8 | 71% | 65% | poor |
| teal | 5 | 58 | 7 | 7 | 0 | 51 | 100% | 12% | poor |
| templ | 5 | 45 | 5 | 4 | 1 | 41 | 80% | 9% | poor |
| wgsl | 5 | 62 | 19 | 19 | 0 | 43 | 100% | 31% | poor |
| angular | 4 | 0 | 0 | 0 | 0 | 0 | n/a | n/a | broken |
| authzed | 5 | 414 | 0 | 0 | 0 | 414 | n/a | 0% | broken |
| bicep | 5 | 87 | 0 | 0 | 0 | 87 | n/a | 0% | broken |
| capnp | 5 | 36 | 0 | 0 | 0 | 36 | n/a | 0% | broken |
| circom | 5 | 13 | 1 | 1 | 0 | 12 | 100% | 8% | broken |
| css | 5 | 5 | 0 | 0 | 0 | 5 | n/a | 0% | broken |
| cue | 4 | 30 | 0 | 0 | 0 | 30 | n/a | 0% | broken |
| devicetree | 5 | 30 | 0 | 0 | 0 | 30 | n/a | 0% | broken |
| fortran | 6 | 35 | 0 | 0 | 0 | 35 | n/a | 0% | broken |
| fsharp | 5 | 43 | 0 | 0 | 0 | 43 | n/a | 0% | broken |
| gdscript | 4 | 31 | 0 | 0 | 0 | 31 | n/a | 0% | broken |
| gleam | 5 | 29 | 0 | 0 | 0 | 29 | n/a | 0% | broken |
| gn | 4 | 13 | 0 | 0 | 0 | 13 | n/a | 0% | broken |
| less | 5 | 28 | 0 | 0 | 0 | 28 | n/a | 0% | broken |
| objc | 5 | 22 | 0 | 0 | 0 | 22 | n/a | 0% | broken |
| odin | 5 | 32 | 6 | 6 | 0 | 31 | 100% | 3% | broken |
| puppet | 5 | 7 | 0 | 0 | 0 | 7 | n/a | 0% | broken |
| r | 5 | 21 | 0 | 0 | 0 | 21 | n/a | 0% | broken |
| rescript | 5 | 130 | 0 | 0 | 0 | 130 | n/a | 0% | broken |
| scss | 5 | 28 | 0 | 0 | 0 | 28 | n/a | 0% | broken |
| tlaplus | 5 | 83 | 0 | 0 | 0 | 83 | n/a | 0% | broken |
| vhdl | 5 | 39 | 0 | 0 | 0 | 39 | n/a | 0% | broken |

"Correct" can exceed "expected" where the tool extracted real declarations the
reviewer had not required (enum members, columns). Several rows depend on a
convention; each result file states the convention and the alternative figure.
Three to twelve files per language is a small sample: read each row as a
description of those files.

What the rows have in common:

- **When something is extracted it is usually right.** Wrong symbols are
  concentrated in a few languages (Prisma, HCL, SQL, Squirrel, OCaml, Crystal).
- **Recall fails by construct.** The library derives each definition query from
  a fixed list of common node names. Plain functions and classes are found;
  anything a language spells its own way is not: Nim exported procedures, Lua
  methods, Julia short-form functions, Elixir guarded clauses, Swift
  initialisers, R function assignments, Bicep resources.
- **A query that compiles is not a query that matches.** Twenty-two languages
  with a query extract nothing, or almost nothing, from valid code.

## Call resolution, 14 languages

| Language | Project | Definitions P / R | High | Medium | Low | Recall | Notes |
|---|---|---|---|---|---|---|---|
| go | gin | 97% / 100% | 90% (18/20) | 14% (14/103, whole tier) | 13% (1/8) | 68% (25/37) | Package qualifier ignored: `errors.New` binds to `gin.New` |
| java | gson | 100% / 97% | 80% (16/20) | 60% (12/20) | 0% (0/20) | 52% (22/42) | Constructor calls not recorded; first-declared overload chosen |
| typescript | zod | 100% / 88% | 90% (27/30) | 55% (16/29) | 0/1 | 44% (15/34) | Import paths not followed |
| tsx | taxonomy | 100% / 69% | 100% (51/51) | none | none | 34% (14/41) | JSX usage never recorded |
| javascript | express | 100% / 41% | 54% (7/13, all edges) | 0% (0/59 references) | manifest lines only | 12.5% (5/40) | CommonJS not handled |
| python | flask | 100% / 99% | 65% (13/20) | 10% (2/20) | 65% (13/20) | 62% (23/37) | Older resolver; receiver object stored as call target |
| ruby | sinatra | 100% / 95% | 90% (18/20) | 30% (6/20) | none | 57% (21/37) | No inheritance or mixins recorded |
| c | redis | 43% / 46% | 100% (24/24) | 100% (23/23) | 23% (3/13) | 87% (33/38) | 40% of `src/` bytes not indexed; see below |
| cpp | leveldb | 98% / 56% | 85% (17/20) | 70% (14/20) | 75% (15/20) | 22% (9/41) | Out-of-line methods not extracted |
| c_sharp | Newtonsoft.Json | 100% / 86% | 85% (17/20) | 60% (12/20) | 80% (16/20) | 37% (18/49) | 43 files time out |
| rust | ripgrep | 100% / 70% | 65% (13/20) | 75% (15/20) | 0% (0/20) | 46% (32/70) | No trait implementations recorded |
| php | guzzle | 100% / 93% | no edges | no edges | no edges | 0% (0/40) | No references extracted |
| kotlin | okio | 98% / 65% | no edges | no edges | no edges | 0% (0/49) | No references extracted |
| scala | os-lib, munit, gears | 85% / 72% | no edges | no edges | no edges | 0% (0/40) | No references extracted |

Precision is strict: the edge must point at the declaration the call invokes.
Each tier sample of 20 carries roughly ±20 points. Lenient figures, where a
reviewer gave one, are in the result files.

The C row needs care. Its sampled precision is perfect and its sampled recall is
high because the five files were small utilities. Over all of redis `src/`, nine
files above the size limit and eight parse timeouts leave about 40% of the C
bytes out of the index; pointer-returning functions are never extracted (0 of
796 by pattern count, including `zmalloc` and `createObject`); and 72
medium-confidence edges bind a call to another program's `static` function
because the real definition was in a file that timed out.

## Defects found

Ordered by how many languages they affect.

1. **The receiver or qualifier of a call is ignored.** A call binds to any
   same-named symbol in the file or directory. Go `errors.New` to the project's
   `New`; Java variable `reader` to a method `reader`; TypeScript
   `Object.defineProperty` to a project method; Rust `self.inner.foo()` to the
   enclosing `foo`; Python `_json.dumps` to the project's `dumps`.
2. **A call outside an extracted function produces no edge.** `find_callers`
   answers from edges, so callers at module top level or inside an anonymous
   callback are missing: 63% of resolved TypeScript calls on zod.
3. **Three "resolved" languages extract no references:** PHP, Kotlin, Scala.
4. **No inheritance for Ruby, C#, Rust, C++ and PHP,** so
   `find_implementations` returns nothing.
5. **Receiver identifiers are stored as calls.** `reader` in `reader.peek()` is
   a call reference: 34% of Java references, 30% of Python call edges.
6. **Overloads are not told apart.** Argument count is zero on every C# and C++
   reference; Java picks the first-declared overload.
7. **The parse limit is five seconds of wall-clock time and the parse is not
   stopped when it expires.** Which files fail depends on machine load. A
   676-byte valid C file reproduces a timeout on its own.
8. **Files over 262,144 bytes are skipped without being named:** redis
   `server.c`, `module.c` and seven more.
9. **File extension alone picks the grammar.** `.m` is always MATLAB (so
   Objective-C is unreachable), `.h` always C (so C++ classes in headers are
   lost), `.fs` Forth (F# skipped), `.hh` C++ (Hack misparsed), `.pp` Pascal
   (Puppet skipped), `.v` V (Verilog and Coq misparsed). A `#!` line is not
   read.
10. **`parse_status` does not report parsing.** `ok` means at least one symbol
    or reference was recorded; `partial` means none were. A file with 150 syntax
    errors can be `ok`, and a valid file of constants is `partial`. No `partial`
    file carries an error text.
11. **Files marked `ok` lose definitions silently:** C# `JContainer.cs` after
    line 360, Crystal classes after macros, Scala 3 files after an indented
    block.
12. **Spurious symbols:** 11% of C# methods are named after the return type;
    C `struct X` in a parameter list is stored as a class definition.
13. **Kinds are unreliable:** methods as functions, objects as functions,
    Protocol Buffers messages as functions.
14. **The `exported` flag follows a capital first letter,** not the `export`
    keyword: 14 of 1,539 TypeScript functions in zod are flagged.
15. **Qualified names omit the owner.** `Error.JSON` and `errorMsgs.JSON` share
    one name, as do all overloads.
16. **Answers contradict themselves:** "COVERAGE TOO LOW ... Do not present what
    follows as an answer" beside "verdict: strong".
17. **`analyze_impact` reports "risk low"** for symbols whose callers the index
    could not bind, and returns an internal error for an ambiguous name.

## Limits of this record

- Samples are small: a handful of files per language and about 20 edges per
  tier. They show direction and systematic failure, not a precise rate.
- One project per language. A different project would give different figures.
- The checks were made by AI reviewers reading source. Their conventions and
  every sampled id are recorded so a person can repeat any row.
- Python was measured before the resolver change merged the same day.
- Nothing here measures the `exact` tier, which needs an imported compiler
  index.

## Files

| File | Holds |
|---|---|
| `G1.json` to `G6.json` | Definition extraction, 12 languages each |
| `R1.json` to `R7.json` | Call resolution, two languages each |
| `definition-query-probe.tsv` | Every grammar and whether it has a definition query |
| `linguist-first-pass.json` | Files and symbols per language over Linguist's samples |
