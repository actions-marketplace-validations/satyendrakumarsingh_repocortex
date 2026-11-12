# Resolution accuracy

RepoCortex labels every relationship with a confidence, and until this document
existed those labels were a design intention rather than a measurement. This is
the measurement.

Everything here is reproducible with the tool itself. If you disagree with a
number, run the four commands in "Method" against a repository you know and
send the disagreement.

## Result

Heuristic `CALLS` edges adjudicated against a compiler-backed index, across
three languages and five repositories — **30,975 adjudicated edges**:

| Confidence | Go | TypeScript | Python | judged |
|---|---|---|---|---|
| `high` | **86.8 – 96.1%** | **87.0%** | **87.1 – 94.3%** | 24,284 |
| `medium` | **5.7 – 18.5%** | **14.6%** | **37.5%** | 6,473 |
| `low` | **2.0 – 2.8%** | **0.0%** | **0.0%** | 218 |

Java was measured later, separately, on a smaller pool: 100.0% `high` on 82 of
82 judged edges, 32.7% recall — see "Java: measured" below.

Per repository:

| Repository | Language | Files | `high` | `medium` | `low` | all tiers | judged |
|---|---|---|---|---|---|---|---|
| prometheus | Go | 1,030 | 86.8% | 5.7% | 2.0% | 65.2% | 16,985 |
| nest | TypeScript | 1,013 | **87.5%** | 17.4% | 0.0% | 75.1% | 1,669 |
| scrapy | Python | 480 | **91.8%** | 44.3% | 0.0% | 80.6% | 8,574 |
| requests | Python | 19 | 94.3% | — | — | 94.3% | 228 |

Python and TypeScript improved after the tier order below was corrected;
Go is unchanged, which was the point of making the order language-dependent
rather than picking one.

### Where the remaining errors come from

The `high` disagreements were bucketed against `scip-python` on scrapy, and
one shape dominated: 45% of them were a reference bound to a same-named symbol
in a NEIGHBOURING file rather than the one the file imported.

    tests/test_addons.py
      9:  from scrapy.utils.test import get_crawler
      58:     crawler = get_crawler(...)        # no local definition

Directory proximity was consulted before the import list, so an unrelated
`get_crawler` in another test file won — at high confidence, 144 times from
that one name. Fixing it moved Python from 87.1% to 91.8%.

The correction is that the ORDER is language-dependent, not that imports
always win:

  - Where a directory IS the package (Go, Java, Kotlin) an unqualified name
    resolves in-package; the imported one would have to be written `pkg.Foo()`.
    Proximity first. Reversing this cost prometheus 0.8 points.
  - Where each file is its own module (Python, TypeScript, JavaScript)
    adjacency implies nothing. Imports first.

A second fix followed from the same measurement: a relative import names a
FILE (`./sibling` → `sibling.ts`), and the import matcher only compared
directory segments, so every relative import missed and proximity answered
instead. Matching file names — only in the languages where an import can name
one — took TypeScript from 87.0% to 87.5%.

Ground truth: `scip-go` 0.2.7, `scip-typescript` 0.3.14, `scip-python` 0.6.6.

### What this says

**`high` is worth acting on, in every language measured. `medium` is not.**

`high` lands between 86.8% and 96.1% across Go, TypeScript and Python — five
repositories, three ecosystems, three different compiler-backed indexers. That
consistency is the useful part: the label means roughly the same thing wherever
you meet it, so a consumer can trust it to rank without knowing which language
produced it.

`medium` ranges from 5.7% to 37.5% and is the tier to distrust. It is also not
rare — 6,473 of the adjudicated edges — which is why the all-tiers figure sits
between 65% and 94% while `high` alone stays near 90%. An agent consuming every
edge equally gets the low number. One filtering with `--min-confidence high`
gets the high one.

`low` is 0–2.8%. It is a hint that a name exists, nothing more.

Two claims this project used to make are corrected by the above. `medium` was
documented as "probable"; at 5.7% on prometheus it is not. `high` was "wrong
only under shadowing or reflection"; at ~87% roughly one edge in eight is wrong
for reasons not yet characterised, and saying so is what makes characterising
them possible.

## Method

```bash
repocortex index && repocortex resolve
scip-go --output index.scip          # compiler ground truth
repocortex scip import index.scip      # lands as REFERENCES/exact/compiler
repocortex doctor --resolution-precision
```

A heuristic `CALLS` edge is adjudicated by asking the compiler what the
reference at that source position resolves to. Agreement means both name the
same definition.

**Judgement is restricted to lines the compiler resolved to exactly one
target.** That restriction is the load-bearing methodological choice, and it
exists because neither looser option is sound:

- *Matching by line alone is too generous.* On RepoCortex's own source, 25.8% of
  referenced lines carry more than one distinct target, so a heuristic edge
  could be scored against a reference it was never about.
- *Matching by line and column is too strict.* The two tools do not share a
  column convention: for `svc.Method()` the resolver records the column of
  `svc` and `scip-go` records the column of `Method`, and the gap varies with
  the receiver's name. Requiring exact columns discarded two thirds of the
  comparison (5,805 judgeable edges fell to 3,541) without either tool being
  wrong.

On unambiguous lines the answer is defined whatever either tool believes about
columns. Everything else is reported as uncovered rather than scored — a
measurement that quietly counts cases it cannot decide is worse than a smaller
one that says so.

## What this is not

**It is not recall** — but recall *is* now measured, separately, by
`repocortex accuracy`. This paragraph used to end the discussion, and it was half
right.

The objection is sound as an objection to the **denominator**: SCIP records that
one symbol *references* another, not that it *calls* it, so dividing by every
SCIP reference penalises the resolver for correctly declining to emit a `CALLS`
edge at a type annotation. What the paragraph then concluded — that a suitable
ground truth does not exist — was wrong. We know which references sit at call
sites, because the parser records that independently of whether the resolver
could bind them. Restricted to lines where the parser saw a call *and* the
compiler resolved that line to exactly one definition, every element of the set
is a call whose correct target is known, and an edge we did not produce there is
a missed caller.

That restriction is the whole soundness argument, so the report states it inline
rather than in a footnote.

The measurement changes the unit: precision here counts *edges*, while a missed
call site has no edge to count, so `repocortex accuracy` collapses each call site
to one outcome and computes precision, recall and F1 over that. Every tier then
shares one denominator, tier recalls sum to the all-tiers recall, and F1 is a
harmonic mean of two fractions over the same set.

**On this repository the number that matters is the one precision could not
show**: `high` is 98.6% precise and catches **61.5%** of adjudicable calls.
2,855 of 8,315 known call sites produce no edge at all — 2,504 unresolved, 351
ambiguous. For a change-safety product that is the important half: an empty
`repocortex callers` result means *unknown*, not *none*, and roughly a third of
known calls are in that gap.

It is still not recall over the calls in the repository. The call-site oracle is
RepoCortex's own parser, so a call it never parsed is neither scored nor missed;
the alternative would be the compiler's own call graph, which SCIP does not
emit.

**It is not a claim about proven edges.** Compiler-backed edges are not scored
against themselves. They are `exact` because a typechecker produced them, and
`doctor --verify-index` enforces that nothing may claim `exact` without
`source=compiler`.

**Java is measured, later and on a small pool.** See "Java: measured" below
for the numbers, and for how small the pool is.

**Five repositories is still a small sample**, and one of them is this
project's own code, which the resolver was developed against — repocortex's 96.1%
is the least trustworthy number in the table for that reason, and it is the
highest. Prometheus, nest and scrapy are third-party code the resolver has
never been tuned against, and they agree with each other at 86.8%, 87.0% and
87.1%.

## Python: now declared, not inherited from defaults

Python used to have no declared resolution behaviour at all. It appeared in none
of the three language switches the resolver carried, so it silently took the
fallback branch of each — not a decision anyone made, just the shape of an `if`.

It also had a parse gap that made the fallback worse than it looks. Python is a
file-is-a-module language, so the resolver consults imports *before* proximity —
but `internal/parse/queries.go` captured no relative imports, so

```python
from .helpers import X
from ..pkg.mod import Y
```

produced **zero** imports. A probe on a six-import file captured three. A module
importing only relatively therefore had no import tier at all, and fell straight
through to the tier the ordering exists to demote.

The added capture is one pattern:

```
(import_from_statement module_name: (relative_import (dotted_name) @path))
```

Measured against `scip-python` 0.6.6, both corpora `--depth 1` clones, the same
SCIP index used for both arms of each pair:

| corpus | arm | `high` | judged | agree | disagree | imports |
|---|---|---:|---:|---:|---:|---:|
| scrapy (480 files) | without relative imports | 95.4% | 5,259 | 5,017 | 242 | 4,244 |
| scrapy | **with** | 95.4% | 5,275 | 5,033 | **242** | 4,266 |
| requests (37 files) | without relative imports | 95.3% | 148 | 141 | 7 | 233 |
| requests | **with** | **95.6%** | 226 | 216 | 10 | 299 |

Both clear the documented `high` floor of 86.8% comfortably. On scrapy the change
added 16 judgeable high-tier edges and **every one of them was correct** —
`disagree` is unmoved at 242.

**One number moved the wrong way and is not a regression.** On requests the
`medium` tier fell from 92.6% (81 judged) to 60.0% (20 judged), and all-tiers from
94.3% to 92.7%. That is composition, not accuracy: 61 correct edges were
*promoted* out of `medium` into `high`, leaving the weaker ones behind in a much
smaller pool. The tier the product tells you to act on improved on both corpora,
which is the tier the number is for. It is recorded here because a tier moving
30pp is exactly the kind of thing that looks like a defect six months later if
nobody wrote down why it happened.

## Python: the receiver's class, read from the source

Until this change a Python method call was resolved by its NAME. `self.save()`
bound when the file happened to contain exactly one `save`; `account.save()`
bound to that same `save` whatever `account` was; and a class imported from the
package that re-exports it — which is how a library's public names are imported
— was found only by the last-resort tier that asks whether a name is unique in
the repository. The parser knew none of the three things that would have
settled those: which class a definition belongs to, what class a receiver is,
and which name an import brought in.

`internal/parse/pyscope.go` now states them, and
`internal/resolve/infer/static.go` uses them.

**What the parser states.**

- A definition's qualified name follows Python's own `__qualname__`:
  `shop/models/account.py:Account.save`, and `build.<locals>.Item.render` for
  anything defined inside a function. Two classes' `save` are two names.
- A receiver is replaced by a class where the source says what it is, and
  nowhere else: a parameter annotated `account: Account` (also `Optional[...]`,
  `X | None`, the quoted and dotted forms), or a variable the function assigns
  exactly once from `Ledger(...)`. A second assignment, a loop target, a `with
  ... as`, an unpacking or a `global` anywhere in the function withdraws it.
  Nothing reads a function's return type, an attribute's type or a module-level
  variable's.
- A call on an expression with no name — `self.repo.save()`, `make().run()` —
  is recorded as made on `?`, so that an empty receiver means the source wrote a
  bare name. `super().save()` is recorded as made on `super`.
- A call through a local value — a parameter, a loop variable — is marked, and
  is bound to nothing.
- `from shop.models import Account` records that `Account` came from
  `shop.models`, and a relative import keeps its dots.

**What the resolver does with them.**

- `self.x()`, `cls.x()`, `Account.x()` and a receiver the parser replaced with a
  class are asked about THAT class first, ahead of the same-file tier: the
  method the class declares, and failing that the nearest one an ancestor
  declares.
- A bare call cannot name a method, and a definition local to a function is
  visible only inside it. Those candidates are removed, and among what is left
  the innermost definition wins.
- A name imported by name resolves through the import that named it: to the
  module if the module defines it, and otherwise to the one definition of that
  name beneath the package — a re-export. `models.CharField(...)` after
  `from django.db import models`, and `httpx.Client()` after `import httpx`,
  resolve the same way.
- `shop/models/__init__.py` is the module `shop.models`.

**What was taken out of `high`.** A method called on an object whose class
nothing states — `target.save()`, `self.repo.save()` — used to bind to the only
`save` in the file, or the only one in an imported module, and be reported at
`high`. Those bindings are still made. They are reported one level down, as
what they are: a binding on a name. The same-directory tier is off for Python
entirely. The measurement behind both is under "What the old tiers were worth".

### Measured against hand-written ground truth

`fixtures/pycalls` is a small package in which every interesting call site
states, in a comment, the definition it reaches, or that it is dynamic and
nothing may claim it at `high`: 57 call sites with a known target and 13
dynamic ones. It covers the shapes a compiler index cannot adjudicate — the
target of a call through a parameter is a local value, which SCIP does not
record — and it is a regression gate (`TestPythonCallGroundTruth`).

| | `high`: right | `high`: wrong | all tiers: right | wrong | no edge | dynamic calls claimed at `high` |
|---|---:|---:|---:|---:|---:|---:|
| before | 26 | **9** | 36 | 9 | 12 | **4 of 13** |
| after | **55** | **0** | 55 | 1 | 1 | **0 of 13** |

The one wrong edge left is `service.charge(...)` where `service` is what a
function returned: the receiver's class is unknown, the name-only tier picks the
one `charge` in an imported module, and that is the subclass's override. It is
reported at `medium`. The one miss is `Tree.Node().walk()`, a call on the
result of a call.

### Measured against a compiler index

Four repositories, each a `--depth 1` clone of a pinned commit, each indexed
once with `scip-python` 0.6.6 and that one index used for both arms. Figures are
`repocortex accuracy` output, unedited: call sites the parser recorded a call on
and the compiler resolved to exactly one definition.

| repository | call sites | arm | `high` sites | right | wrong | `high` precision | `high` recall | all-tier precision | all-tier recall | no edge |
|---|---:|---|---:|---:|---:|---:|---:|---:|---:|---:|
| scrapy `c7eacdbb` (480 `.py`) | 9,999 | before | 5,230 | 5,017 | 213 | 95.9% | 50.2% | 93.1% | 63.9% | 3,132 |
| | | **after** | 6,732 | 6,709 | 23 | **99.7%** | **67.1%** | 96.2% | 72.6% | 2,446 |
| requests `v2.32.3` (36) | 352 | before | 293 | 283 | 10 | 96.6% | 80.4% | 95.0% | 80.4% | 54 |
| | | **after** | 300 | 295 | 5 | **98.3%** | **83.8%** | 96.4% | 83.8% | 46 |
| httpx `0.28.1` (60) | 1,826 | before | 405 | 376 | 29 | 92.8% | 20.6% | 54.1% | 31.7% | 755 |
| | | **after** | 1,293 | 928 | 365 | 71.8% | 50.8% | 67.0% | 51.6% | 420 |
| django `5.1.4` (2,788) | 44,787 | before | 17,996 | 15,443 | 2,553 | 85.8% | 34.5% | 78.7% | 40.6% | 21,667 |
| | | **after** | 30,359 | 27,579 | 2,780 | 90.8% | 61.6% | 86.3% | 63.0% | 12,084 |

Two rows in that table do not say what they appear to. On httpx `high`
precision FELL, from 92.8% to 71.8%, and on django it reached only 90.8%. Both
are the compiler index being wrong, and the next section is the evidence for
saying so rather than a reason to look away from the number.

One row says something about the baseline: on django, before this change,
`high` measured **85.8%** — under the 86.8% floor this document publishes for
the tier.

### The compiler index is wrong about wildcard re-exports

`httpx/__init__.py` is twelve lines of `from ._models import *`. A test that
writes `httpx.Cookies()` reaches the class `Cookies` in `httpx/_models.py`.
`scip-python` 0.6.6 records that occurrence as a reference to
`` `httpx._models`/Request# `` — a different class in the same module — and does
the same for every name that arrives through a wildcard import: every
`models.CharField(...)`, `models.IntegerField(...)` and `models.BooleanField(...)`
in django's tests is attributed to `DateTimeField`. Read from the SCIP file
directly, not from RepoCortex's import of it:

```
tests/models/test_cookies.py:24   cookies = httpx.Cookies()
  [23 14 19]  scip-python python … httpx/__init__:
  [23 20 27]  scip-python python … `httpx._models`/Request#
```

So an edge that names the right class is scored wrong, and the better the
resolver gets at these calls the lower its measured precision goes. The
baseline's precision on httpx was high because it bound almost none of them at
`high`; it bound them at `low`, where 424 of 618 were scored wrong for the same
reason.

This was checked with something that depends on neither tool. For every call
site where an edge and the compiler index disagree and the call is a bare name
or is qualified by a name the file imported, the import chain is followed with
Python's own `ast` module — `from … import *` and `__all__` included — and the
site is counted right only if the chain ends exactly where the edge points. A
second, smaller class is corrected the same way: where an edge names a function
defined inside the function the call is written in, and the compiler index names
a same-named function defined inside a DIFFERENT function, Python's scoping
makes the compiler's answer impossible. Nothing else is touched. Both
corrections are applied to both arms, whatever tier and reason an edge carries.

| repository | arm | verdicts corrected | `high` right | wrong | `high` precision | `high` recall | all-tier precision | all-tier recall |
|---|---|---:|---:|---:|---:|---:|---:|---:|
| scrapy | before | 0 | 5,017 | 213 | 95.9% | 50.2% | 93.1% | 63.9% |
| | **after** | 17 | 6,726 | 6 | **99.9%** | **67.3%** | 96.4% | 72.8% |
| requests | before | 0 | 283 | 10 | 96.6% | 80.4% | 95.0% | 80.4% |
| | **after** | 0 | 295 | 5 | **98.3%** | **83.8%** | 96.4% | 83.8% |
| httpx | before | 358 | 376 | 29 | 92.8% | 20.6% | 87.5% | 51.3% |
| | **after** | 365 | 1,293 | 0 | **100.0%** | **70.8%** | 93.0% | 71.6% |
| django | before | 842 | 15,573 | 2,423 | 86.5% | 34.8% | 82.4% | 42.5% |
| | **after** | 2,725 | 30,299 | 60 | **99.8%** | **67.7%** | 94.7% | 69.1% |

Read the first table as the measurement and the second as what it means. The
first is what the tool prints; the second is what remains after 3,107 verdicts
the index got wrong are set right by a method anyone can re-run. Neither is
offered without the other.

A third artefact is left in both tables because it cuts both ways and cannot be
corrected, only counted. A call site is adjudicated per LINE, so a line holding
two calls is scored against whichever one the compiler resolved. On requests all
5 `high` sites still scored wrong are of that kind — the compiler's one target
on the line has a different name from the call that was bound — and on scrapy 3
of the remaining 6 are.

### What each rule is worth

Edge by edge rather than site by site, over the four repositories together,
after the two corrections above. The last column is the same figure as the
compiler index scores it.

| tier | why the edge was made | django | scrapy | httpx | requests | all four | agreed | as indexed |
|---|---|---:|---:|---:|---:|---:|---:|---:|
| `high` | imported by name from the module that defines it | 5,584 of 5,584 | 2,318 of 2,320 | 134 of 134 | 135 of 138 | 8,171 of 8,176 | 99.9% | 99.8% |
| `high` | inherited: declared on an ancestor of the receiver's class | 6,650 of 6,714 | 799 of 805 | 49 of 49 | 7 of 7 | 7,505 of 7,575 | 99.1% | 99.1% |
| `high` | imported by name from a package that re-exports it | 5,252 of 5,273 | 789 of 789 | — | — | 6,041 of 6,062 | 99.7% | 99.5% |
| `high` | declared on the enclosing class (`self`, `cls`) | 4,544 of 4,545 | 1,345 of 1,348 | 88 of 88 | 67 of 67 | 6,044 of 6,048 | 99.9% | 99.9% |
| `high` | qualified by an imported package that re-exports it | 4,122 of 4,122 | 8 of 8 | 576 of 576 | 0 of 6 | 4,706 of 4,712 | 99.9% | 36.2% |
| `high` | defined in the same file (a bare name) | 2,727 of 2,729 | 911 of 911 | 187 of 187 | 80 of 80 | 3,905 of 3,907 | 99.9% | 97.6% |
| `high` | declared on the receiver's class (annotation, constructor, or the class written out) | 978 of 978 | 518 of 518 | 259 of 259 | 5 of 5 | 1,760 of 1,760 | 100.0% | 100.0% |
| `high` | qualified by an imported module that defines it | 366 of 367 | 15 of 17 | — | 1 of 1 | 382 of 385 | 99.2% | 99.2% |
| `high` | defined in the enclosing scope, nearest of several | 84 of 84 | 23 of 24 | — | — | 107 of 108 | 99.1% | 98.1% |
| `high` | receiver spells a module's file name | 0 of 3 | — | — | — | 0 of 3 | 0.0% | 0.0% |
| `medium` | one match in an imported module; receiver's class unknown | 420 of 1,204 | 397 of 483 | 6 of 13 | 0 of 2 | 823 of 1,702 | 48.4% | 48.4% |
| `medium` | one match in the same file; receiver's class unknown | 49 of 207 | 37 of 118 | 4 of 17 | 0 of 6 | 90 of 348 | 25.9% | 25.9% |
| `medium` | the class does not declare it, its ancestors did not settle it; one match by name | 89 of 109 | 10 of 11 | — | — | 99 of 120 | 82.5% | 82.5% |
| `medium` | one match in an imported module; a bare name the file never imported | 28 of 46 | — | — | 0 of 2 | 28 of 48 | 58.3% | 58.3% |
| `low` | several matches among imports; receiver's class unknown | 8 of 1,227 | 110 of 237 | 4 of 24 | 0 of 1 | 122 of 1,489 | 8.2% | 8.2% |
| `low` | unique name in the repository, no import evidence | 64 of 100 | 0 of 12 | 0 of 66 | — | 64 of 178 | 36.0% | 33.1% |
| `low` | the class does not declare it, its ancestors did not settle it; several matches by name | 0 of 4 | — | — | — | 0 of 4 | 0.0% | 0.0% |

Over all four, `high` is 38,621 of 38,736 edges, **99.7%** — 91.7% as indexed.
`medium` is 46.9% and `low` 11.1%, which is what those two labels say they are.

The six `high` edges on requests the correction could not reach are
`requests.get(...)` and `requests.Request(...)` in a `src/`-layout repository:
the lines are among the "different name" artefacts above, and the `ast` walk
does not look under `src/`. They are left as the index scores them.

### What the old tiers were worth

The relabelling was decided by this table. It was taken on an intermediate
build, before the labels moved, on scrapy and django, and counts edges for a
method called on an object whose class nothing states:

| what bound it, at `high` | scrapy | django |
|---|---:|---:|
| the only definition of the name in the file | 37 of 113 (33%) | 48 of 168 (29%) |
| the only one in an imported module | 393 of 479 (82%) | 394 of 1,057 (37%) |
| several among the imports (then `medium`) | 110 of 235 (47%) | 7 of 1,098 (1%) |

None of the three clears 86.8%, and on django the first two were most of the
reason `high` measured 85.8%. They are `medium` and `low` now, and the final
table above is the same population after the move.

The same-directory tier, which reports `medium` where a language requires an
import, was right **1 time in 213** on Python across scrapy, requests and httpx:
a Python name a file neither defines nor imports cannot denote a neighbour's
definition. It is off for Python (`LanguageSpec.ProximityIsEvidence`), and a
reference only it could bind is recorded ambiguous.

A call on `super()` whose class has no indexed ancestor declaring the method
used to fall back to the only definition of the name in the file — which is,
more often than not, the method the call is written in. Measured on django
before that fallback was removed: right 0 times in 54.

### Arity for Python: measured, and removed

A Python arity query was built for the same run: a method's `self` taken off the
count, and a signature with a default, a splat or a bare `*` refused outright,
as a call passing a splat is. One binary with it and one without, same trees,
same indexes.

| repository | `high` | `medium` | `low` right, with arity | `low` right, without |
|---|---|---|---:|---:|
| scrapy | identical | identical | 109 | 110 |
| django | identical | identical | 66 | 67 |
| requests, httpx | identical | identical | identical | identical |

It chose between candidates 56 times, all in the lowest tier, and was right
once; without it two more call sites were right. That is the TypeScript result
again (see "A4 on TypeScript" below) for the same reason — arity only ranks
candidates a tier has already accepted, and in Python those are the name-only
bindings for an unknown receiver — and it got the same decision. The query, its
dialect and the method bookkeeping were deleted.

### The tier

`repocortex languages` now lists Python as `typed`. The tier is derived, not
listed (`internal/parse/tiers.go`), and until now the derivation had one route:
import, receiver and arity queries. Python has no arity query, for the reason
just given, so the derivation gained a second route — a scope analysis that
states the receiver's class — and the tier's description changed from "receiver
and arity discrimination" to "receiver discrimination by arity or class".

That is a change to what the word means, made on these numbers:

| | `high` precision | `high` recall |
|---|---:|---:|
| Python, four repositories, as indexed | 71.8 – 99.7% | 50.8 – 83.8% |
| Python, four repositories, corrected | 98.3 – 100.0% | 67.3 – 83.8% |
| Go (own source) | 98.6% | 60.4% |
| TypeScript | 99.5% | 66.4% |
| Java (spring-petclinic-rest) | 100.0% | 32.7% |

If the correction is not accepted the as-indexed row is the one to read, and on
it Python clears the 86.8% floor on three repositories of four and not on httpx.
The promotion rests on the corrected row, and on the correction being checkable.

### Cost

django, 2,788 Python files, `index --full` then `resolve`, the two binaries
alternated within each of three rounds on one machine:

| phase | binary | wall clock, three rounds | CPU, three rounds |
|---|---|---:|---:|
| `index --full` | before | 30.4 – 34.6 s | 73.8 – 88.2 s |
| | after | 24.2 – 35.9 s | 63.4 – 85.7 s |
| `resolve` | before | 6.8 – 8.9 s | 7.6 – 9.7 s |
| | after | 8.5 – 11.5 s | 9.2 – 12.4 s |

The machine was doing other work throughout, which is why the wall clock ranges
overlap and the CPU column is the one to compare. Indexing is unchanged within
that noise: 6,817 files discovered, 2,340 indexed, 41,688 symbols in both arms.
`resolve` costs about a quarter more CPU — round by round 9.7 to 12.4 s, 7.6 to
9.7 s and 7.7 to 9.2 s — and most of it is the hierarchy lookups for parked
references and the SQLite writes for edges it did not use to find. On scrapy
(480 files) neither phase moved outside the noise.

Resolution writes 32% more edges on that tree (57,015 to 75,129) and 24% fewer
reference rows are stored (253,125 to 192,001: the object of `row.save()` is no
longer recorded as a call to `row` when `row` is a local value). Two costs were
found by profiling and removed before these figures were taken: the scoping
filter re-derived a reference's kind once per candidate, and the hierarchy pass
asked the registry for the same class's supertypes once per chunk of parked
references rather than once per run.

### What this does not do

- **A receiver whose class is only inferable.** `crawler = get_crawler()`,
  `self.settings.getbool(...)`, a pytest fixture: 1,088 of scrapy's 2,446
  remaining misses are a method on such a receiver. Reading return annotations
  and attribute types would reach some of them; neither is done.
- **Method resolution order.** Two bases that both declare a method leave the
  call ambiguous, because the hierarchy is stored by name and unordered. Python
  would take the leftmost.
- **A class identified by name alone.** The hierarchy is keyed by class NAME.
  A receiver's own class is pinned to one definition; its ancestors are not, so
  two unrelated classes called `Base` share their ancestry.
- **Aliased members.** `from m import a as b` is not followed.
- **The denominator.** About 1,200 of scrapy's remaining misses are lines where
  the compiler's one target has a different name from every call the parser
  recorded there — `isinstance(x, Response)` resolves `Response`, and
  `isinstance` is a builtin. No call resolver can produce that edge. They count
  as misses for both arms, as they did before.

## Measured on a polyglot repository that is not ours

Every figure above this section was measured on Go, on corpora chosen for
measurement. The obvious objection is that a resolver tuned on Go, measured on
Go, reports Go's numbers — so this section reports the tiers on somebody else's
codebase, in languages that are not Go, adjudicated against those languages'
own compilers.

**The corpus.** A production polyglot estate: 8,151 indexed files, 91,548
symbols, 69,998 relationships. 40.8% Python, 23.7% TypeScript, 8.5% Go, 6.8%
Vue, plus Java and JavaScript. Nothing about it was chosen by us and nothing in
RepoCortex was tuned on it.

**The oracles.** `scip-python` 0.6.6 on one Python service (273 of 273 documents
matched, 5,081 compiler-proven edges) and `scip-typescript` 0.3.14 on one
frontend (1,339 of 1,348 documents, 5,834 edges).

| Tier | Judged | Agree | Disagree | Precision |
|---|---:|---:|---:|---:|
| `high` | 3,534 | 3,395 | 139 | **96.1%** |
| `medium` | 50 | 0 | 50 | **0.0%** |
| `low` | 6 | 1 | 5 | 16.7% |
| all | 3,590 | 3,396 | 194 | 94.6% |

**`high` holds outside Go.** 96.1% here against 86.8–97.8% on the Go corpora.
That is the claim the tier ordering rests on, and until now it had only ever
been demonstrated in the language the resolver was developed in.

**`medium` is 0.0% here, and that is the more useful finding.** Fifty judged
edges, fifty wrong. On Go the same tier measures 35.4%, and 45.8% after arity
disambiguation. So "shown, never acted on" is *understated* for Python and
TypeScript on this corpus: a medium edge here is not weak evidence, it is noise.
A single global range across all languages hides that, and the per-language
split is the honest presentation. It also explains why arity disambiguation —
which targets `medium` — is worth extending beyond Go.

### What this does NOT establish

**Coverage is 5.7%** — 3,590 of 62,961 heuristic `CALLS` edges were adjudicated.
One Python service of 3,100 Python files, and one frontend. The precision figure
is sound for the edges it judged and is not a whole-repository number; the rest
are neither right nor wrong here.


## Java: measured attempt #5, and the first precise diagnosis

Earlier entries here record Java as "blocked: needs gradle". Measured against a
real 2,681-file Java application (Maven, `pom.xml`, Java 25), that description
is wrong on both halves, and the correction matters because it changes what
would have to be fixed.

**What actually happened, in the order the blockers appeared:**

1. **RepoCortex's own gate had no key.** Indexing Java is refused by default
   because it runs the repository's Maven or Gradle build, which executes
   arbitrary code from the repo. The refusal said "re-run with the explicit
   build opt-in" -- and `AllowRepoBuild` appeared nowhere outside tests. No
   flag, no documented config key. The instruction could not be followed, so
   some unknown share of "3 of 3 repos failed" was this, not the indexer.
   Fixed: `repocortex indexers install --allow-repo-build`.

2. **The documented config path is wrong on macOS.** Code comments named
   `~/.config/repocortex/config.yml`; `os.UserConfigDir()` resolves to
   `~/Library/Application Support/repocortex/config.yml` there. A config written
   to the documented path is read by nothing, so the tool repeated the same
   "set accuracy.indexers.runner" instruction after it had been set. Fixed.

3. **The pinned indexer image is older than modern Java.** With the first two
   cleared, `scip-java` ran cpam's Maven build for 7 minutes and exited 1:

       javac 17.0.16
       error: release version 25 not supported

   The pinned image ships JDK 17. Any project targeting Java 21 or 25 is
   unindexable with it, regardless of build tool.

**So the honest statement is: the oracle path works; the pinned image is too
old.** `scip-java` handled Maven without complaint -- gradle was never the
obstacle on this repository. Java precision was still unmeasured when this was
written, and this section said no Java number should be quoted until a JDK
21/25 image was pinned and its digest recorded. What happened instead: the
same JDK 17 image's launcher fetches a Temurin 21 for itself at start-up, and
the number in "Java: measured" below came from that image, digest recorded
there.

**What this cost, and the lesson.** Three failures, ~10 minutes of build time,
and none of the three was detectable from the error in front of us at the time.
All three were knowable BEFORE running anything: a flag either exists or does
not, a config path either resolves or does not, and a project's
`maven.compiler.release` can be compared against the pinned image's JDK without
compiling a line. A tool that only reports incompatibility after a seven-minute
build is making the user run the experiment we should have run for them.

**Java was still unmeasured when this was written; it has since been measured
— see "Java: measured" below. Both the "until a JDK 21/25 image is pinned"
condition above and the reason recorded next are superseded by that
section.** `scip-java`
reported `blocked: needs gradle` on all four Java targets in this repository —
a fourth independent repository where compiler-backed Java indexing did not run.

### How the rest of the field handles this, and why it is not the same problem

The obvious question about a `blocked: needs gradle` is what everyone else does
instead. From the eight teardowns in `docs/internal/research/`: **not one of them uses a
compiler for Java. Not one of them uses a compiler for any language.** There is
no SCIP import, no LSIF, no language server anywhere in the set — the single
mention of LSP is mcp-codebase-index arguing against it on install cost
(`README.md:277`).

They all parse Java with tree-sitter or regex and emit edges from it. That is a
legitimate engineering choice and it is the same choice RepoCortex makes when no
indexer runs. The difference is what happens to the number afterwards.

CodeGraph binds a Java field by name suffix — `endsWith('Service')` emits an
edge at **confidence 0.85**, and `/^[A-Z][a-zA-Z]+$/` emits at **0.70**, which
fires on every PascalCase identifier in the language (`frameworks/java.ts:129-190`).
Those constants are not measurements. Nothing adjudicated them, and there is no
oracle in that project against which they could be. Its own sound alternative —
`inferJavaFieldReceiverType`, which reads the declared field type — sits unused
in the same codebase at `name-matcher.ts:1060`.

So the honest comparison is not "our Java is blocked and theirs works". It is:

  - **RepoCortex's Java was UNMEASURED when this comparison was written.** It
    was indexed, edges were produced, and they carried `high`/`medium`/`low` —
    but no compiler had adjudicated them, so this document said NOT MEASURED
    and the per-language table said it on every run. It has since been
    measured, on 82 edges — see "Java: measured" below.
  - **Theirs is UNMEASURABLE.** They ship a Java confidence number derived from
    a name suffix, with no oracle in the project that could ever check it.

The gap that matters is not the gradle dependency. It is that a tool which
cannot say how accurate it is on a language will also never discover that it is
wrong on that language. `scip-java` failing on four of four repositories is a
real limitation and it is recorded as one; a 0.85 that nothing produced is a
different kind of problem, and it does not show up as a limitation anywhere.

**`repocortex accuracy` took about twenty minutes on this repository when these
figures were first produced. It now takes 4 seconds, for byte-identical output.**

Two joins had nothing to stand on. `edges(evidence_file, evidence_line)` had no
index at all — every other access path on that table is by symbol id, and
nothing had ever needed to look an edge up by location — so adjudicating a call
site against the compiler's edge at the same file and line scanned every edge in
the run. `symbol_references(file_id, start_line)` likewise.

The indexes were worth 2x. The rest was the query SHAPE. It built a `callsite`
CTE of every parsed call site — 305,237 rows — and joined it to the
compiler-adjudicated set of 5,587. SQLite cannot index a CTE, so the plan read
`SCAN c` then `SCAN u`: a nested loop over the product. Inverted so the small
side drives, each of the 5,587 lines does two indexed probes instead. 160s to
0.16s on that query alone.

Recorded because a measurement nobody runs stops being true without anyone
noticing, which is the exact failure this whole document exists to prevent.

### Recall, per language

Precision says how often an edge is right. It says nothing about the calls never
found, and for a change-safety product a missed caller is the expensive error.
Restricted to call sites the compiler resolved to exactly one definition:

| | `high` precision | `high` recall | F1 | adjudicated sites | produced no edge |
|---|---:|---:|---:|---:|---:|
| **Python** (before the scope analysis) | 94.2% | 56.3% | 70.4% | 3,277 | **1,306 (39.9%)** |
| **TypeScript** | 99.5% | 66.4% | 79.6% | 2,330 | **749 (32.1%)** |
| Go (own source) | 98.6% | 60.4% | 75.0% | 8,315 | 2,855 (34.3%) |
| Java (spring-petclinic-rest) | 100.0% | 32.7% | 49.2% | 251 | **169 (67.3%)** |

**TypeScript is the most precise tier measured anywhere in this project** —
99.5%, eight wrong edges in 1,554. Python is the least at 94.2%, and both clear
the 86.8% floor.

The Python row was taken on a private service that cannot be re-measured, and
it predates the scope analysis. What Python measures now, on four public
repositories, is in "Python: the receiver's class, read from the source" above:
`high` precision 98.3–100.0% and recall 67.3–83.8% once the compiler index's
errors on wildcard re-exports are corrected, and 71.8–99.7% and 50.8–83.8% as
the index scores it.

**The recall gap is consistent across Go, Python and TypeScript**: between a third and
two fifths of known call sites produce no edge at all. That number is the same
shape in Go, Python and TypeScript, which says it is a property of heuristic
resolution rather than of any one language's grammar — and it is the number a
consumer needs, because it is what makes an empty `repocortex callers` result mean
*unknown* rather than *none*.

Java, measured later and on a much smaller pool (82 judged edges, 251
adjudicated sites), is the outlier on recall: two thirds of its known call sites
produce no edge, at a precision of 100.0%. The measurement and its caveats are
in "Java: measured" below.

**`medium` is 0.0% in both languages** — 10 judged in Python, 27 in TypeScript,
every one wrong. Precision and recall agree with each other here, which they
should: the tier is contributing nothing on either measure.

## Java: measured

Java is no longer the unmeasured language. Measured against `scip-java` on
[spring-petclinic/spring-petclinic-rest](https://github.com/spring-petclinic/spring-petclinic-rest)
at commit `4cd8e1b0cd42578e882247d8801f6be5d402f118` — a single-module Maven
project on Spring Boot 4.1.1, 109 Java files, `maven.compiler.release=17`
inherited from the Spring Boot parent, no `maven.compiler.executable`
override, `mvn -q -DskipTests compile` green in 42 s on the host. That is
exactly the closing condition the previous version of this section set, and
the first repository tried that met it.

The index came from the pinned image, under the docker runner, with no change
to RepoCortex:

| | |
|---|---|
| Image | `sourcegraph/scip-java@sha256:a05564b9c9951660fc7069b431f1a18f4575b570b7ce09a14936729832c1fc8d` (linux/amd64, run under emulation on arm64) |
| `scip-java version` | `HEAD-20260402-1456` — a snapshot build, not the `0.12.2` the tag claims |
| Toolchain inside | `javac 17.0.16` is the image's default; scip-java's own launcher fetches Temurin 21.0.12.1 through coursier at start-up ("Using JVM version '21'"). Maven's `.m2` honours the `HOME=/out` the runner sets. The JDK download did not: coursier is a JVM program and reads `user.home`, which is `?` for a uid the image has no passwd entry for, so on this measurement 611 MB of Temurin landed in the repository's working tree as `?/.cache/coursier`, untracked and not ignored. The runner now also passes `JDK_JAVA_OPTIONS=-Duser.home=/out` (every JDK 9+ launcher reads it, and it leaves the image's own `JAVA_TOOL_OPTIONS` alone), so the JDK cache goes to the output mount with everything else and a re-index leaves `git status` clean. The runner then prunes that mount before returning — Maven's `.m2`, coursier's archives, anything that is not an index — and removes it outright if no index was produced, so what outlives a run is the `.scip` file and nothing else. Both verified by re-running this indexing after the fix |
| Build driven | `clean verify -DskipTests -Dmaven.test.skip=true ...` as `java.go` constructs it |
| Wall clock | 3 m 27 s from `docker run` to a 2.8 MiB `java.scip` |

Adjudicated with `repocortex accuracy` after `repocortex scip import`, excerpted,
unedited — the block omits the tool's preamble:

```
JAVA
  TIER    SITES  RIGHT  WRONG  PRECISION  RECALL  F1
  high    82     82     0      100.0%     32.7%   49.2%
  medium  0      0      0      —          0.0%    0.0%
  low     0      0      0      —          0.0%    0.0%
  all     82     82     0      100.0%     32.7%   49.2%
  missed  169    —      —      —          —       —
  251 adjudicated call site(s); 169 produced no edge at all.

WHERE THE MISSES COME FROM
  CAUSE       CALL SITES  MEANING
  unresolved  156         the reference was bound to nothing
  ambiguous   13          several candidates and no way to choose, so no edge was claimed

COVERAGE
  Adjudicated 251 of 1,582 parsed call site(s) (15.9%). The rest sit on lines
  no compiler fact covers, and are neither right nor wrong above.
  Against 1,010 compiler fact(s) on 513 unambiguous line(s); 214 ambiguous
  line(s) were excluded. The resolver produced 624 heuristic CALLS edge(s) overall.

LANGUAGES
  LANGUAGE  FILES  CALL SITES  CALLS EDGES  COMPILER FACTS  ADJUDICATED  STATUS
  java      109    1,582       624          1,010           251          measured
```

**`high` clears the 86.8% floor — at 100.0%, 82 of 82 — and it is the only
tier Java has.** `medium` and `low` adjudicated zero call sites, so they are not
0% and not wrong; they are absent, and this table cannot say anything about
them. The recall side is the one to read carefully: **169 of 251 adjudicated
call sites (67.3%) produced no edge at all**, against a third to two fifths for
Go, Python and TypeScript. 156 of those are `unresolved`, 13 `ambiguous`, and
none named a wrong definition — the resolver is silent on Java far more often
than it is mistaken, which is the right failure for a change-safety tool but
twice the silence measured anywhere else. Two caveats keep this from being
quoted as "Java: 100%". The judged pool is small — 82 edges, against 1,554 for
TypeScript and 5,275 for scrapy — and one wrong edge would move it to 98.8%.
And only 15.9% of parsed call sites were adjudicated at all, because
`scip-java` attributes 3,748 references to a file rather than a definition and
the import skips them, so the number describes that slice, not the repository.
The recall table above now carries a Java row with both caveats attached.

### Earlier attempts

This section used to read "attempted, not measured", and the reason was
recorded because it is a property of the product and not only of the effort:
**compiler-backed indexing for Java runs the repository's real build, so it
inherits every constraint that build has.** TypeScript and Python were
straightforward by comparison — `repocortex indexers install --yes` provisioned
`scip-typescript` and `scip-python` and both produced usable indexes on the
first attempt. Java did not, across three repositories and seven builds, though
`scip-java` itself worked when pointed at a two-class Java 17 Maven project:

| repository | build | outcome |
|---|---|---|
| spring-boot | Gradle | no Gradle on the host; RepoCortex correctly reported `blocked: needs gradle` |
| commons-lang | Maven | builds, but emits no SemanticDB. `-source 8` is below what the javac plugin attaches to, and raising it hit javadoc, then checkstyle, then a pom that overrides the compiler configuration scip-java injects |
| petclinic | Maven + Gradle | needs `--build-tool=maven`; then its GraalVM Maven extension needs Java 17 while scip-java falls back through 21 → 17 → 11 and reports only the failure on 11 |

The closing condition written down then — a Maven repository on Java 11 or
later whose build succeeds with `-DskipTests` and does not override
`maven.compiler.executable` — is the one spring-petclinic-rest met. The
candidates tried on the way to it, in order, and why the script rejected each.
One caveat on the last row: petclinic-rest's release level is inherited from
the Spring Boot parent, which the script's `pom.xml` grep cannot see, so its
qualification came from `mvn help:evaluate`, not from the script:

| candidate | build | rejected because |
|---|---|---|
| spring-projects/spring-petclinic-rest | — | does not exist; the clone 404s (the project lives under `spring-petclinic/`) |
| apache/commons-csv | Maven | `maven.compiler.source` 1.8, below the javac-plugin floor |
| apache/commons-text | Maven | same, 1.8 |
| javalin/javalin-samples | Maven | the root is an aggregator POM that declares no compiler level; the script stops there |
| **spring-petclinic/spring-petclinic-rest** | Maven | **qualified** — release 17 via the Spring Boot parent |

A synthetic Java corpus was deliberately not used — uniform generated code
makes resolution easy and would report an inflated precision, which is worse
than no number.

Two RepoCortex defects fell out of the earlier attempts, both fixed before this
measurement:

- `indexers list` printed **`ready`** for a step the planner had blocked,
  because the status column never consulted `Blocked`. The blocked step is the
  Java one, withheld pending an explicit opt-in precisely because indexing it
  executes the repository's build. A safety gate the interface does not show is
  one a user works around by accident.
- The Docker runner passed `-u <uid>` without a writable `HOME`, so `HOME`
  resolved to `/` and coursier died on `AccessDeniedException: /.cache` before
  any build started. The container route is the one the planner recommends
  *instead of* running an untrusted build on the host, and it could not index
  anything.

One host condition cost thirty-five minutes this time and is worth knowing
about, because it looks like a RepoCortex failure and is not: Docker Desktop's
`docker-credential-desktop` helper hung indefinitely from a non-interactive
shell, so every `docker pull` blocked before it reached the daemon, and
RepoCortex's 10-minute provisioning timeout reported the pinned image as
`could not install`. The pull itself is 2.7 GB and took under ten minutes once the
helper was out of the way (`DOCKER_CONFIG` pointed at an empty config, which
makes the digest-addressed pull anonymous and changes nothing about what is
verified). The runner now asks the configured helper for `list` before every
pull, bounded at five seconds, and a helper that does not answer is reported
by name with that workaround instead of as a ten-minute `could not install`.

## Changes since the table above was measured

The resolver changed materially in one working session. The table in **Result** is
still the record for the tree it was measured on; this section records what moved
since, measured the same way, so the two are not confused with one another.

**Re-baseline before comparing.** The corpus here is a `--depth 1` prometheus clone
of **727 Go files**, not the 1,030-file checkout the table above used, and the
unmodified tree scores **86.7%** on it against the documented 86.8%. That difference
is the corpus, not a regression — which is exactly why every figure below is paired
with a baseline re-measured on the same tree and the same 57 MB `scip-go` index,
rather than compared against the published number. A −2.4pp regression measured
against the wrong baseline reads as −0.1pp of rounding.

| change | `high` precision | judged | wrong |
|---|---|---:|---:|
| baseline | 86.7% | 12,496 | 1,666 |
| + validated receiver matching, import aliases | 86.4% | 12,628 | 1,719 |
| **+ anchored import matching** | **90.2%** | 12,236 | **1,205** |

**Anchored import matching is the whole gain**, +3.8pp on its own. It replaced the
bag-of-words segment test in `importSet.covers`, whose doc comment had argued the
opposite: *"Deliberately generous… being too strict here would silently drop real
edges, which is the more expensive mistake."* Measurement disagrees. Generosity was
not free: a loose match that happens to be unique returns `ConfidenceHigh`, so
looseness manufactures confident wrong edges. It removed roughly 4,000 edges overall
and 514 wrong ones from the judged set at a cost of 392 judged-correct.

**Validated receiver matching is neutral here, which is not the same as disproven.**
It only bites where a fully-qualified name carries its owner — Java, Kotlin,
TypeScript. Prometheus is Go, where `model.DefaultFQN` synthesises `path:name` FQNs
with no owner, so the channel it tightens barely fires. It is unmeasured on the
languages it exists for.

### The LanguageSpec refactor: proven by digest, not by percentage

Per-language behaviour used to live in three separate switches on language name
inside `resolver.go` — `packageIsDirectory` (go/java/kotlin), `receiverIsAType`
(java/kotlin/typescript) and `memberSeparators` (java/kotlin/scala | ruby/rust/
cpp/c/php). Three switches, three different language sets, no single place that
said what a language does. `LanguageSpec` replaces them with one declarative
per-language table, selected by a factory held per-`Resolver`.

A refactor of the resolver has to prove it changed nothing, and **precision was
the wrong instrument for that proof here.** RepoCortex's own source *is* the
corpus, so adding two files and shifting line numbers underneath a `scip-go`
index built before the move changes which lines are judgeable at all. That alone
read as 97.8% → 97.6% — a number that looks like a small regression and is
actually an artifact of measuring a moving corpus.

So the check was a controlled A/B on a **frozen** copy: identical tree, identical
SCIP index, only the binary differing. The **post-resolve registry digests were
identical**:

```
a1ebd89167c5ad84916f4f6920a313edcaf83b1e8778201cc75e2f404413903d
```

with precision unchanged at `high` 97.8%, 4,631 judged, 102 disagree.

That is a strictly stronger claim than matching percentages. Equal precision means
the same *number* of edges were right; an equal digest means the resolved graphs
are **byte-identical** — same edges, same tiers, same evidence, same order. Two
different graphs can score the same and differ everywhere, which is precisely
what a percentage cannot rule out and a digest can.

**Determinism re-checked at the same time**: one index digest and one
post-resolve digest across 10 runs at each of 1, 2, 4, 8 and 16 workers.

```
index         9c39d5dc54072878df9e49b904105237ba100fd3feb496c597f43433b0ce1904
post-resolve  a1ebd89167c5ad84916f4f6920a313edcaf83b1e8778201cc75e2f404413903d
```

The general rule worth taking from this: **when refactoring something whose
output is reproducible, compare the output, not a score computed from it** — and
freeze the corpus first, or the measurement moves under the thing being measured.

### The cap-drop counter, and what it immediately found

`MaxCandidates` bounds fan-out, and dropped candidates were folded into
`Ambiguous`. So "this name is too common to weigh" and "two candidates and no
evidence to choose" were the same number. That mattered more once
`repocortex accuracy` began publishing recall, because this was a named blind spot
inside it.

Counters only — nothing in the binding path reads them — and the proof is that
the registry digest is **byte-identical** to the arm without them
(`8c90385d…`), which is stronger than equal percentages.

**On this repository: 1,093 cap drops, 47.3% of the 2,311 ambiguous references.**
Nearly half of "ambiguous" was never a disambiguation problem.

The counter's first act was to show that a feature is inert. `narrowByReceiver`
exists to rescue exactly these over-subscribed names, and it is gated on
`ReceiverDenotesType`, which is false for Go — so **it rescued 0 of 1,093** and
cannot fire on a Go repository at all.

### Common-name suppression: it ships, and it is smaller than it sounds

| tier | baseline | with suppression |
|---|---|---:|
| `high` | 98.0%, 5,475 judged, 108 wrong | **identical** |
| `high` recall | 60.4% | 60.4% |
| `low` | 0.3%, 323 judged, 322 wrong | 0.3%, 296 judged, 295 wrong |

It moves `high` **not at all** — the three numbers are identical, not merely
close. It removes 78 `low` edges, of which the 27 adjudicable ones were 27 wrong
and 0 right, and recall is unchanged at every tier.

Smaller still: `impact.DefaultMinConfidence` is `medium`, so a default consumer
**already never saw these edges**. The win is invisible unless someone passes
`--min-confidence low`. Shipped because it is strictly non-negative and removes
noise, not because it improves a number anyone acts on. Recording that
distinction matters more than the change does.

### A third change that measured badly and was reverted — receiver type inference

A per-file local type environment, so a Go receiver arrives as `Service` rather
than as `s`. The parser half worked exactly as designed — `svc *Service` →
`Service`, `c := &Client{}` → `Client`, and `n := NewService()` correctly left
alone, because recovering a type from a `New` prefix is a convention, not a
declaration.

| arm | `high` prec | `high` recall | `medium` prec | all recall |
|---|---:|---:|---:|---:|
| baseline | **98.6%** | 60.4% | 35.4% | 60.6% |
| type env only | 98.3% | 60.2% | 50.0% | 60.5% |
| + receiver-denotes-type | 98.3% | 60.6% | **54.5%** | **61.0%** |

The last row is the tempting one: `medium` nearly doubles, all-tiers recall
gains 0.4pp, and the cap rescue finally fires. But in `high` it added **18
correct edges and 14 wrong ones — a 56% hit rate into a tier measuring 98.6%.**
That is the A19 mechanism one tier up: dilution of the tier the product tells you
to act on. Reverted.

**The mechanism is reusable and is the point of recording this.** Tier 2
(receiver) sits *above* tiers 3 and 4 (import, same-directory). Go is a
directory-is-package language where proximity is strong evidence — and while Go
receivers arrived as `s` and `c`, tier 2 almost never fired, so proximity
effectively decided. Supplying real type names woke tier 2 up, and a file-stem
coincidence (`Store` matching `store.go`) began out-ranking the correct
in-package definition.

So this is not "Go type inference does not work". It is **"the tier order was
tuned for a receiver that carried no type."** For a directory-is-package
language, inferred type evidence must not outrank package proximity. Retrying
means moving tier 2 below `pickSameDir` where `PackageIsDirectory` holds, and
measuring that.

**One safety property does not hold as stated.** "A mis-inference yields no
edge, never a wrong one" is true for `high` — `receiverMatches` validates rather
than contains, and tier 2 reports High only on a unique survivor. It is false in
general: a receiver matching *two* candidates by file stem takes tier 2's
multi-candidate branch and emits a deterministic preference at `medium`. A test
now pins both halves, so closing that gap is deliberate and widening it fails.

### Two changes that measured badly and were reverted

Both were adopted from a competitor's resolver, both looked plainly safe, and
neither survived measurement. They are recorded because the reasoning that produced
them will recur.

**A bare call cannot name a method in Go or Rust.** True of the languages, false of
our parser. `ref.Receiver == ""` means the receiver query could not express the
receiver, not that the source wrote none — `receiverQueries` captures only a simple
identifier receiver, so `s.repo.Save(id)` arrives with an empty receiver and every
chained call in Go was read as bare. `repocortex callers Save` went from two bound
callers to zero. Every unit test passed throughout, because they set `Receiver`
directly; only the end-to-end CLI path exercised the parser that leaves it empty.

**Production code never binds to a test definition.** Cost **2.4pp** of `high`
precision. It does not merely delete wrong test-file candidates: by shrinking the
candidate set it tips references across the disambiguation thresholds, so references
previously recorded as *ambiguous* now resolve — 426 of them, of which 64 were right
and **362 wrong**. It converts honest ambiguity into confident wrong answers, which
is precisely what `internal/resolve/infer`'s package doc forbids.

The transferable rule, now stated in the resolver: **a filter that removes candidates
is not automatically safe, because removing candidates also removes the ambiguity
that was protecting the caller.** "Fewer candidates" and "fewer wrong edges" are
different claims, and only the second is worth having.

### A4 — arity disambiguation, which ships

When several candidates share a name, the number of arguments at the call site
is evidence about which one was meant. `Save(id)` and `Save(ctx, id, opts)` are
different functions. Until this change the database could not hold that fact:
`symbols.signature` existed and `internal/parse` never populated it for code
symbols, and `symbol_references` had no notion of an argument list at all.

Migration `0010_arity` adds `symbols.param_count` and
`symbol_references.arg_count`, and the count is plumbed parse → pipeline →
registry → resolver.

**It is a RANKING change, not a filter, and that is the whole design.** It is
consulted at exactly three points, every one of them a branch where the resolver
has ALREADY chosen a candidate and is about to report `medium` with the reason
"several matched, preferred deterministically". All it changes is WHICH of those
several is preferred. It cannot remove a candidate, cannot resolve a reference
that would otherwise have been ambiguous, and cannot reach a `high` branch —
every one of those returns before arity is consulted. That is what makes it a
different shape of change from A19, which lost 2.4pp of `high` precisely by
shrinking candidate sets until honest ambiguity tipped into confident wrong
answers.

Measured on RepoCortex's own source, 431 Go files, against an 18 MB `scip-go`
0.2.7 index. One **frozen** tree and one index in both arms, only the binary
differing:

| tier | baseline | with A4 | judged | wrong: baseline → A4 |
|---|---:|---:|---:|---:|
| `high` | 97.8% | **97.8%** | 5,747 | 129 → **129** |
| `medium` | 35.4% | **45.8%** | 48 | 31 → **26** |
| `low` | 0.3% | 0.3% | 309 | 308 → 308 |
| all | 92.3% | 92.4% | 6,104 | 468 → 463 |

`high` is **identical in all three numbers**, not merely close — which is what
the design predicts rather than a result it got lucky with. Recall is unchanged
at every tier (`high` 60.3%, all-tiers 60.5%), and so are the miss counts:
2,891 unresolved and 407 ambiguous in both arms. The same call sites resolved,
at the same tiers, and five of them changed which definition they name. **All
five changed from wrong to right.**

Determinism re-checked at the same time: one index digest and one post-resolve
digest across ten runs at each of 1, 2, 4, 8 and 16 workers.

```
index         d30573e74d317782be62fb613f781e38b0507dc68d093b9541c6e7bf5e5217cd
post-resolve  d6be8fcf1b179f0fe107435e1e494b3b1004db80344ee2cbd4832837c1adbc8d
```

The index digest is byte-identical to the baseline arm's, which is the expected
result and worth stating: `param_count` is an INPUT to resolution and the digest
covers the graph, so indexing is unmoved and only the resolved edges differ.

**Two things this does not claim.**

*The sample is small.* `medium` is 48 judged edges out of the 13,394 heuristic
ones on this corpus, so 10.4pp is five edges. What justifies shipping is not the
size of that number but the shape of the change: it cannot subtract, and every
edge it moved it moved in the right direction. A corpus with a larger `medium`
tier would measure it better, and prometheus is the obvious next one.

*Only Go is enabled.* Counting is language-specific in ways that are easy to get
subtly wrong, and a wrong arity is worse than none because it is evidence
pointing at the wrong candidate. Go declares several parameters in one node —
`func f(a, b int)` is ONE `parameter_declaration` with two `name:` fields, so
counting nodes reports 1 for a function of arity 2. Python writes `self` at the
declaration and not at the call site. TypeScript's optional parameters mean a
declared count of three is satisfied by a call passing one. Java is the language
overload resolution matters most in and the one language here that had no
compiler ground truth for until spring-petclinic-rest. Variadic forms are
refused outright in every language, on both sides, because an open-ended
signature cannot be compared to a call at all.

The encoding carries that refusal. `pkg/model.Arity` and the SQL columns are
offset by one, so **zero means UNKNOWN and a count of n is stored as n+1**. A
plain integer cannot express the difference between "this call passes no
arguments" and "nobody worked out how many arguments this call passes", and
those are opposite facts — the first narrows candidates, the second is the
absence of evidence. Conflating them would let every pre-migration row, every
SCIP-imported symbol and every language without an arity query assert that it is
niladic. That is A18's mistake one field over: reading "the parser could not
express it" as "the source did not write it".

### A4 on TypeScript

The A4 section above ends with the rule that a language ships arity when a
measurement against a compiler index says it helps. This is that measurement
for TypeScript, and the answer was no — not because the count was wrong, but
because it moved nothing.

**The count that was tried.** TypeScript counts ONLY when every parameter is
required. An `optional_parameter` (`b?: number`) or a `spread_element`
(`f(...xs)`) makes the whole list unknown by node type; a `required_parameter`
makes it unknown when it carries a `value:` field (`b = 1`) or its `pattern:`
is a `rest_pattern` (`...xs: number[]`). That is the only honest count, because
the resolver compares by equality and a declared count several call shapes
satisfy is not a count. The unit tests for those five shapes passed, and on
nest the rule produced a known parameter count for 3,061 symbols and a known
argument count for 39,959 references, so the numbers below are a test of the
idea rather than of the plumbing.

**Measured on nest** (`nestjs/nest` at `39fbddae51281ca56bf2fed9123d98e61509066c`,
`--depth 1`), against the root `scip-typescript` 0.3.14 index (9.2 MB, produced
by `repocortex indexers install --yes` without `node_modules`, so third-party
imports do not resolve and the index is smaller than a full build would give).
One tree and one index in both arms, only the binary differing.

Before:

| tier | sites | right | wrong | precision | recall | F1 |
|---|---:|---:|---:|---:|---:|---:|
| `high` | 1,004 | 962 | 42 | 95.8% | 37.5% | 54.0% |
| `medium` | 181 | 160 | 21 | 88.4% | 6.2% | 11.7% |
| `low` | 38 | 7 | 31 | 18.4% | 0.3% | 0.5% |
| all | 1,223 | 1,129 | 94 | 92.3% | 44.1% | 59.7% |
| missed | 1,339 | — | — | — | — | — |

After:

| tier | sites | right | wrong | precision | recall | F1 |
|---|---:|---:|---:|---:|---:|---:|
| `high` | 1,004 | 962 | 42 | 95.8% | 37.5% | 54.0% |
| `medium` | 181 | 160 | 21 | 88.4% | 6.2% | 11.7% |
| `low` | 38 | 7 | 31 | 18.4% | 0.3% | 0.5% |
| all | 1,223 | 1,129 | 94 | 92.3% | 44.1% | 59.7% |
| missed | 1,339 | — | — | — | — | — |

2,562 adjudicated call sites in both arms; 218 unresolved, 1,116 ambiguous and
5 attributed to no caller in both. The two `accuracy --json` outputs are
byte-identical, and so are the two CALLS edge sets: 3,000 edges in each arm
(1,744 `exact`, 540 `high`, 205 `medium`, 511 `low`), and not one differs in
source, target, tier or evidence line. Arity did not move a judged edge, and
it did not move an unjudged one either.

**Decision: it did not ship.** The gate was that `high` precision must not
drop by more than 0.2pp AND either recall or the `high` edge count must rise.
Precision held exactly, but nothing rose, and a change that measurably does
nothing is not a change to carry — every arity query is a grammar-shaped
contract that has to survive every future grammar bump, and the tier table
would be promising `typed` for a discrimination that has never discriminated.
`internal/parse` was reverted; TypeScript stays `resolved`.

Why it moved nothing is worth recording, because it is not the reason the A4
section predicted. That section worried that optional parameters would make
the count WRONG. They did not — the rule refuses those. The reason is where
the count is consulted. `preferByArity` runs at three points, each a branch
where SEVERAL candidates survived a receiver match, an import's local name, or
an explicit import, and one has to be preferred. On nest, TypeScript reaches
those branches almost never: 204 of the 205 heuristic `medium` edges carry the
detail "same directory, but the language requires an import", which is the
branch that fires when EXACTLY ONE same-named neighbour exists — a branch with
nothing to choose between, and which does not consult arity for that reason.
The one remaining `medium` edge is "multiple matches among imports", and its
candidates did not separate by count. So the counts were there (a known
argument count sat under 202 of the 205 `medium` sites) and the rule was
correct, and it was asked exactly once. Meanwhile 1,116 adjudicated sites are
ambiguous with no edge claimed at all, and arity cannot reach those by design:
it prefers among candidates the resolver has already accepted and never
removes one, which is the property that kept `high` unmoved. The ceiling on
TypeScript recall here is candidate selection, not candidate ranking, and the
evidence that would resolve those sites is a receiver type rather than an
argument count.

## A determinism defect the digest caught

Measured on RepoCortex's own source (425 files, `scip-go` 0.2.7 as ground truth):
`high` **97.8%** over 4,619 judged edges, `medium` 34.1%, `low` 0.0%, all tiers
92.7%. Coverage: 4,890 of 10,484 heuristic `CALLS` edges adjudicated against
22,389 compiler facts on the 12,313 lines the compiler resolved to exactly one
target.

Running that measurement surfaced something the precision numbers do not show.
The registry digest **disagreed between runs over byte-identical source**, and
across worker counts — three distinct digests across six repeats.

Indexing was not at fault: index-only runs were identical 4 of 4. `resolve` was,
and it reproduced with the deferred inherited-dispatch pass disabled, so it was
not new work. `repocortex diff` between two divergent runs named the cause exactly:

```
orphaned migrations/0003_data_access.sql:table:table_accesses  inbound 14 -> 0
removed  verifyGraph -[READS]-> 0003_data_access.sql:table:table_accesses
added    verifyGraph -[READS]-> 0005_fk_indexes.sql:table:table_accesses
```

`table_accesses` is created in `0003_data_access.sql` and named again by a
`CREATE INDEX` in `0005_fk_indexes.sql`, so two symbols carry the name.
`tableSymbols` picked the canonical one by "first declaration wins" over a
`SELECT` with **no `ORDER BY`** — and an unordered `SELECT` has no first. Row
order followed the query plan over ids the *parallel* indexing pass assigned, so
the winner changed between runs, moving 40+ `READS`/`WRITES` edges and orphaning
two table nodes.

Ordering by path then line makes "first" mean first in the migration sequence,
which is where the `CREATE TABLE` is — so the canonical node became the correct
one as well as a stable one. After the fix: identical digests across six repeats
and across 1, 2, 4, 8 and 16 workers.

Two things are worth keeping from this. **A digest that is only checked in CI on
small fixtures will not catch an ordering bug that needs a name collision to
appear** — this one needed a real repository with a real migration history.
And **`ORDER BY` is a correctness clause whenever a query feeds a first-wins
rule**, not a formatting preference; the comment above the loop already said
"first declaration wins" while the query left "first" undefined.

## Reproducing

The numbers above came from:

| | |
|---|---|
| Host | Apple M4 Pro, macOS 26.5.1, `darwin/arm64` native build |
| RepoCortex | commit at the time of writing, `CGO_ENABLED=0` |
| Ground truth | `scip-go` 0.2.7, `scip-typescript` 0.3.14, `scip-python` 0.6.6, `scip-java` `HEAD-20260402-1456` (pinned image digest `a05564b9…`, docker runner) |
| Corpora | `--depth 1` clones: prometheus, nest, scrapy, requests, spring-petclinic-rest (`4cd8e1b`) |
| Python scope analysis | `--depth 1` clones of scrapy `c7eacdbb30a4563bbbe04ed02781768f4e19eabb`, requests `v2.32.3` (`0e322af8`), httpx `0.28.1` (`26d48e06`), django `5.1.4` (`2d4add11`); `scip-python index --project-name repocortex-repository --project-version 0.0.0`, as `repocortex indexers` runs it; `index --full --accuracy fast`, `resolve --no-scip`, `scip import`, `accuracy` |

`scip-go` on prometheus takes about eight minutes and produces a 57 MB index.
On RepoCortex's own source it takes nineteen seconds and produces 7 MB.
