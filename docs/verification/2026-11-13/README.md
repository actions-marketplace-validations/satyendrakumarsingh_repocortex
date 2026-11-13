# Language verification, 13 November 2026

A second look at what RepoCortex extracts and resolves, made after the defects
the [record of 12 November](../2026-11-12/README.md) listed were worked on. It
answers four questions: do the fixes hold on code the work was not tuned on,
what did they break, how good is definition extraction in 36 further languages,
and how does the tool behave on two very large repositories. Where a result is
unflattering it is stated as plainly as a good one.

Every number here has a denominator and comes from a result file in this
directory or from a command whose output is quoted. Percentages are computed
from the counts in those files.

## Contents

- [Summary](#summary)
- [How it was measured](#how-it-was-measured)
- [Definition extraction, 14 languages, before and after](#definition-extraction-14-languages-before-and-after)
- [Call resolution, 14 languages, before and after](#call-resolution-14-languages-before-and-after)
- [Definition extraction, 36 more languages](#definition-extraction-36-more-languages)
- [Large repositories](#large-repositories)
- [The final build against the round-2 build](#the-final-build-against-the-round-2-build)
- [Defects found, and what was done](#defects-found-and-what-was-done)
- [What is still wrong](#what-is-still-wrong)
- [Limits of this record](#limits-of-this-record)
- [Files](#files)

## Summary

| Claim or earlier figure | Measured here |
|---|---|
| `high` is 95.1% to 100% precise, by hand, on nine projects (`docs/accuracy.md`, 12 November) | On 14 projects the work was not tuned on, round 1 (build `53246ac`) measured `high` at 396/420 (94%) pooled, from 73% (cpp, 22/30) to 100%. Round 2 (build `8748b29`, fresh samples) measured 405/420 (96%) pooled, from 73% (cpp, 22/30) to 100%. `high` is at least 95% on 8 of 14 projects in round 1 and 11 of 14 in round 2 |
| `medium` is at least 60% precise (`docs/accuracy.md`) | Round 1: 207/272 (76%) pooled over the 10 projects that have a `medium` edge. Round 2: 249/259 (96%) pooled over 9. Kotlin moved from 5/30 (17%) to 26/30 (87%) because most of its overload cases moved down to `low`; the two rounds are not like for like |
| `low` is a hint that a name exists | Round 2: 184/280 (66%) pooled. python 8/30; kotlin 9/30; c_sharp 14/30; java 18/30 correct (same-arity overloads and calls on values of unknown type) |
| Call recall on the files sampled was 0% to 87% (12 November) | Round 1: 393/529 (74%) overall, 35% (c_sharp) to 100% (tsx) by project. Round 2: 450/529 (85%) overall, 55% (javascript) to 100% (tsx) |
| PHP, Kotlin and Scala extract no references | Recall now 33/40 PHP, 40/40 Kotlin (round 1: 27/40), 30/40 Scala (round 1: 20/40) |
| Definitions in 14 languages: the five largest files of one project each | Round 1: 2,996 of 3,113 expected found correctly, 153 spurious. Round 2: 3,081 of 3,106, 12 spurious |
| Definitions in 72 languages: pooled precision 90%, recall 45% (12 November) | 36 further languages, three files each, build `f783fa5`: 3,495 of 4,103 expected found correctly (85% recall), 107 spurious (97% precision). 27 of 36 reach 90% on both. The lowest recall: fsharp 12%, make 15%, powershell 54%, groovy 57% |
| The first line of an answer is true of the code | Round 1: 33 of 41 answers. Round 2: 40 of 42 |
| Round 1 wrong edges (121 examples listed) | 44 gone, 19 bound to the right target, 15 moved to a lower tier, 43 still wrong at the same tier |
| A very large repository indexes in bounded time and memory | Kubernetes (build `bdc86c9`): 16,486 files indexed, index 259.76 s wall, 2.39 GB peak resident, resolve 57.39 s. VS Code (build `a75b887`): 20,448 discovered, index 342.24 s, 1.54 GB peak resident, resolve 172.14 s. Both on a machine under load |
| The final build (`61f5992`) against the round-2 build (`8748b29`), same 14 projects | 8 projects have the same edge set and the same symbols. 6 differ; each change is itemised below. Net edges: c 5108 to 5106; cpp 1203 to 1203; kotlin 5814 to 5824; php 3163 to 3162; tsx 1254 to 1247; typescript 2078 to 2058 |

## How it was measured

Each result file is the product of an independent check, by reading the source
files, following a written brief. The checker did not write the code under
test and did not change it. The briefs are not in this directory because they
name directories on the machine they were run on; their method is set out here.

| Round | Builds | What was checked |
|---|---|---|
| 1 | `53246ac` (every file records it in `repocortex_commit`) | Calls and definitions on 14 projects, one per language |
| definitions | `f783fa5` | Definition extraction only, 36 further languages |
| 2 | `8748b29` | The same 14 projects at the same commits and the same files as round 1, after the fixes |
| edge comparison | `8748b29` and `61f5992` | The full edge set and symbol set of the 14 projects under the two builds, compared by a script, not by reading |

Builds, by the merge that brought each in: `53246ac` is the merge of the C and C++ macro mask, `f783fa5` the resolver fixes from the first verification, `8748b29` the scale fixes, and `61f5992` the last merge before this record, the definitions for the grammar-limited languages.

**Round 1 and round 2, per language.** The project is cloned at depth 1 and
indexed with `init`, `index`, `resolve`, `status`; the clone commit is in
`project_commit`. These projects were chosen as code the tool was not tuned on.

- **A. Definitions.** The five largest non-test, non-generated source files of
  the language, skipping any over 1,500 lines, fixed by that rule before any
  output was seen. Each is read in full and every definition a developer would
  expect an index to hold is listed (functions, methods, types, type aliases,
  and top-level constants where the language treats them as declarations). The
  symbols table is then compared. Correct: right name, right kind family, and a
  line within the declaration's first lines (an annotation line directly above
  counts, and is counted separately as `line_on_attribute`). Missed: absent.
  Spurious: extracted but not a definition, or a duplicate of one.
- **B. Call precision.** For each confidence tier that exists, 30 `CALLS` or
  `INSTANTIATES` edges drawn by a fixed rule (`ORDER BY (id * 2654435761) %
  4294967296 LIMIT 30`), restricted to callers in files of the language. Each is
  judged by reading the call site and the target. Strict: a call bound to a
  same-named function of another class, module or package is wrong, and so is
  a library or standard-library call bound to a repository symbol. An edge
  that cannot be settled after reading the code and imports is counted as
  `unclear`, which is not right. A tier with fewer than 30 edges is judged whole.
- **C. Call recall.** In the same five files, the first 8 call sites per file, in
  source order, whose target is defined in the repository (calls, method calls,
  constructor calls, JSX usages), listed by hand before the index is consulted.
  Each is then looked up: an edge to the right definition, an edge to a wrong
  one, or no edge.
- **D. Hierarchy.** Up to 20 `EXTENDS` or `IMPLEMENTS` edges drawn by the same
  rule and judged by reading the declaration, and every declared supertype
  relationship with a target in the repository in the five files.
- **E. Answers.** Three real questions per language (`find_callers`,
  `analyze_impact`, `find_implementations`), the first lines copied verbatim and
  judged for whether the first line is true of the code and admits its limits.

Round 2 repeats this with these differences: the same project, commit and files;
a fresh sample of 30 per tier drawn by `ORDER BY (id * 40503) % 65521 LIMIT 30`;
every wrong example round 1 listed looked up individually (gone, moved to a lower
tier, bound to the right target, or still wrong); the same recall sites; the same
three commands; and a list of regressions.

**Definitions in 36 further languages.** One well-known open-source project per
language (real configuration from well-known projects for data formats), cloned
at depth 1 or fetched as a tarball. Three files: the three largest of the language
under 800 lines, not tests, generated or vendored (fewer where the project has
fewer). The three are indexed alone. The rule the tool states for the language
(`fixtures/langs/<language>/expected.tsv`) decides what is a definition and what
kind it has. A form the tool's own `KNOWN_GAPS.md` lists is counted as
`missed_known_gap`, not `missed`. Where the checker disagreed with the rule it
did not change how it counted; the disagreement is under `rule_objections`.

**Edge comparison.** Both builds index and resolve each of the 14 projects at the
same commit; the edges (kind, tier, call site, target) and the symbols (file, line,
kind, name) are written out and compared line by line. Nothing in it was judged
by reading, so it shows what changed and not whether the change is right; the
round-2 readings of these projects are the evidence for that.

## Definition extraction, 14 languages, before and after

Expected, correct, missed and spurious over the five files per project. Round
2 recounted the expected set, so the two `expected` columns can differ.

| Language | Project | Round 1: expected / correct / missed / spurious | Round 2: expected / correct / missed / spurious |
|---|---|---|---|
| go | spf13/cobra | 90 / 90 / 0 / 13 | 90 / 90 / 0 / 0 |
| java | square/javapoet | 340 / 340 / 0 / 0 | 340 / 340 / 0 / 0 |
| typescript | honojs/hono | 291 / 271 / 20 / 0 | 291 / 291 / 0 / 0 |
| tsx | tremorlabs/tremor | 143 / 139 / 4 / 0 | 143 / 139 / 4 / 0 |
| javascript | koajs/koa | 107 / 107 / 0 / 1 | 107 / 107 / 0 / 1 |
| python | pallets/click | 281 / 281 / 0 / 7 | 263 / 263 / 0 / 7 |
| ruby | rack/rack | 327 / 305 / 22 / 0 | 327 / 327 / 0 / 0 |
| c | jqlang/jq | 223 / 219 / 4 / 0 | 223 / 220 / 3 / 0 |
| cpp | google/benchmark | 167 / 135 / 32 / 1 | 166 / 149 / 17 / 4 |
| c_sharp | serilog/serilog | 299 / 288 / 11 / 102 | 299 / 299 / 0 / 0 |
| rust | sharkdp/fd | 135 / 128 / 7 / 0 | 137 / 137 / 0 / 0 |
| php | Seldaek/monolog | 130 / 130 / 0 / 0 | 130 / 130 / 0 / 0 |
| kotlin | square/moshi | 210 / 201 / 9 / 3 | 218 / 218 / 0 / 0 |
| scala | com-lihaoyi/os-lib | 370 / 362 / 8 / 26 | 372 / 371 / 1 / 0 |
| **all** |  | 3,113 / 2,996 / 117 / 153 | 3,106 / 3,081 / 25 / 12 |

Spurious symbols in round 1 were concentrated: C# 102 (return types and `if`/`return`
recorded as methods), Scala 26 (local and anonymous-class definitions), Go 13
(function-local `var` declarations). In round 2 the only language with more than
a handful is C++ (4, `= default` and `= delete` declarations recorded as
functions). The per-file counts, examples of each missed and spurious symbol,
and whether each round 1 example is fixed are in the result files.

## Call resolution, 14 languages, before and after

### Precision by tier

Right / judged, with the percentage in brackets and the number wrong and
unclear where there were any. Round 1 and round 2 are different samples of 30
(round 2 drew a fresh one), so a difference between the columns is not a paired
comparison. Where a tier holds 30 or fewer edges, both rounds judged all of them.

**`high`**

| Language | Round 1 | Round 2 |
|---|---|---|
| go | 30/30 (100%) | 30/30 (100%) |
| java | 30/30 (100%) | 30/30 (100%) |
| typescript | 30/30 (100%) | 29/30 (97%), 1 wrong |
| tsx | 30/30 (100%) | 30/30 (100%) |
| javascript | 25/30 (83%), 5 wrong | 30/30 (100%) |
| python | 29/30 (97%), 1 wrong | 30/30 (100%) |
| ruby | 28/30 (93%), 1 wrong, 1 unclear | 28/30 (93%), 1 wrong, 1 unclear |
| c | 30/30 (100%) | 30/30 (100%) |
| cpp | 22/30 (73%), 5 wrong, 3 unclear | 22/30 (73%), 8 unclear |
| c_sharp | 30/30 (100%) | 30/30 (100%) |
| rust | 28/30 (93%), 2 wrong | 30/30 (100%) |
| php | 27/30 (90%), 3 wrong | 30/30 (100%) |
| kotlin | 29/30 (97%), 1 wrong | 26/30 (87%), 4 wrong |
| scala | 28/30 (93%), 2 wrong | 30/30 (100%) |
| **pooled** | 396/420 (94%) | 405/420 (96%) |

**`medium`**

| Language | Round 1 | Round 2 |
|---|---|---|
| go | none | none |
| java | none | none |
| typescript | 24/30 (80%), 4 wrong, 2 unclear | 25/30 (83%), 4 wrong, 1 unclear |
| tsx | none | none |
| javascript | 0/2 (0%), 2 wrong | none |
| python | 28/30 (93%), 1 wrong, 1 unclear | 30/30 (100%) |
| ruby | 30/30 (100%) | 30/30 (100%) |
| c | 29/30 (97%), 1 wrong | 30/30 (100%) |
| cpp | 30/30 (100%) | 30/30 (100%) |
| c_sharp | 18/30 (60%), 11 wrong, 1 unclear | 19/19 (100%) |
| rust | 30/30 (100%) | 30/30 (100%) |
| php | none | none |
| kotlin | 5/30 (17%), 25 wrong | 26/30 (87%), 4 wrong |
| scala | 13/30 (43%), 17 wrong | 29/30 (97%), 1 wrong |
| **pooled** | 207/272 (76%) | 249/259 (96%) |

**`low`**

| Language | Round 1 | Round 2 |
|---|---|---|
| go | 12/18 (67%), 6 wrong | 12/18 (67%), 6 wrong |
| java | 20/30 (67%), 10 wrong | 18/30 (60%), 12 wrong |
| typescript | 2/4 (50%), 2 wrong | 6/7 (86%), 1 wrong |
| tsx | 0/1 (0%), 1 wrong | 0/1 (0%), 1 wrong |
| javascript | 1/3 (33%), 2 wrong | 4/4 (100%) |
| python | 8/30 (27%), 20 wrong, 2 unclear | 8/30 (27%), 22 wrong |
| ruby | 25/30 (83%), 5 wrong | 27/30 (90%), 3 wrong |
| c | none | none |
| cpp | 28/30 (93%), 2 wrong | 26/30 (87%), 4 wrong |
| c_sharp | 6/30 (20%), 24 wrong | 14/30 (47%), 15 wrong, 1 unclear |
| rust | 8/10 (80%), 2 wrong | 8/10 (80%), 2 wrong |
| php | 24/30 (80%), 5 wrong, 1 unclear | 26/30 (87%), 3 wrong, 1 unclear |
| kotlin | 22/30 (73%), 8 wrong | 9/30 (30%), 21 wrong |
| scala | 25/30 (83%), 5 wrong | 26/30 (87%), 4 wrong |
| **pooled** | 181/276 (66%) | 184/280 (66%) |

Unflattering rows: in round 1, `high` was below 90% for javascript 25/30, cpp 22/30; in round 2, for cpp 22/30, kotlin 26/30.
C++ round 2 has no wrong `high` edge in the sample but 8 are unclear, and an unclear edge is not counted right.

Tier counts changed a great deal between rounds, which the percentages hide.
For Kotlin, `CALLS` edges from `.kt` files went from high 598, medium 413, low 159 to
high 636, medium 63, low 419 as overload cases moved down to `low`. For C#, `CALLS`
went from high 844, medium 53, low 338 to high 937, medium 19, low 450.
Counts per kind and tier are in each file under `index.edges_by_kind_and_tier`.

### Recall

Sites are the first 8 per file whose target the repository defines; the list is
the same in both rounds. "None" is no edge at all.

| Language | R1 sites | R1 right / wrong / none | R1 recall | R2 sites | R2 right / wrong / none | R2 recall |
|---|---:|---:|---:|---:|---:|---:|
| go | 37 | 37 / 0 / 0 | 100% | 37 | 37 / 0 / 0 | 100% |
| java | 40 | 33 / 1 / 6 | 83% | 40 | 35 / 1 / 4 | 88% |
| typescript | 32 | 19 / 0 / 13 | 59% | 32 | 25 / 0 / 7 | 78% |
| tsx | 40 | 40 / 0 / 0 | 100% | 40 | 40 / 0 / 0 | 100% |
| javascript | 31 | 17 / 0 / 14 | 55% | 31 | 17 / 0 / 14 | 55% |
| python | 40 | 36 / 0 / 4 | 90% | 40 | 38 / 0 / 2 | 95% |
| ruby | 32 | 22 / 0 / 10 | 69% | 32 | 25 / 0 / 7 | 78% |
| c | 40 | 40 / 0 / 0 | 100% | 40 | 40 / 0 / 0 | 100% |
| cpp | 40 | 23 / 1 / 16 | 58% | 40 | 26 / 1 / 13 | 65% |
| c_sharp | 40 | 14 / 0 / 26 | 35% | 40 | 32 / 2 / 6 | 80% |
| rust | 37 | 32 / 0 / 5 | 86% | 37 | 32 / 0 / 5 | 86% |
| php | 40 | 33 / 0 / 7 | 83% | 40 | 33 / 0 / 7 | 83% |
| kotlin | 40 | 27 / 1 / 12 | 68% | 40 | 40 / 0 / 0 | 100% |
| scala | 40 | 20 / 4 / 16 | 50% | 40 | 30 / 0 / 10 | 75% |
| **all** | 529 | 393 / 7 / 129 | 74% | 529 | 450 / 4 / 75 | 85% |

Sites are few (at most 40 per language), and the first eight sites of a file are
not a random sample, so a recall figure describes those files.

### Hierarchy

Edges is the number of `EXTENDS` and `IMPLEMENTS` edges whose source is in the
language. Sampled is how many were judged and how many of those pointed at the
declared supertype. Declared is the number of supertype relationships with a
target in the repository found by reading the five files, and how many had the
right edge.

| Language | Round 1 | Round 2 |
|---|---|---|
| go | 0 edges; sampled 0/0; declared 0/0 | 0 edges; sampled 0/0; declared 0/0 |
| java | 9 edges; sampled 9/9; declared 0/0 | 9 edges; sampled 9/9; declared 0/0 |
| typescript | 88 edges; sampled 20/20; declared 58/58 | 87 edges; sampled 19/20; declared 58/58 |
| tsx | 4 edges; sampled 4/4; declared 4/5 | 4 edges; sampled 4/4; declared 4/5 |
| javascript | 0 edges; sampled 0/0; declared 0/0 | 0 edges; sampled 0/0; declared 0/0 |
| python | 79 edges; sampled 20/20; declared 13/30 | 101 edges; sampled 20/20; declared 30/30 |
| ruby | 29 edges; sampled 20/20; declared 9/9 | 29 edges; sampled 20/20; declared 9/9 |
| c | none in C |  |
| cpp | 15 edges; sampled 15/15; declared 1/1 | 33 edges; sampled 20/20; declared 1/1 |
| c_sharp | 86 edges; sampled 20/20; declared 4/5 | 99 edges; sampled 20/20; declared 5/5 |
| rust | 3 edges; sampled 3/3; declared 0/0 | 3 edges; sampled 3/3; declared 0/0 |
| php | 242 edges; sampled 20/20; declared 4/4 | 238 edges; sampled 19/20; declared 4/4 |
| kotlin | 60 edges; sampled 20/20; declared 8/8 | 60 edges; sampled 20/20; declared 8/8 |
| scala | 68 edges; sampled 20/20; declared 39/43 | 71 edges; sampled 20/20; declared 44/48 |

Two wrong hierarchy edges were found in round 2: PHP `extends \DateTimeImmutable`
(the built-in, written with a leading backslash) bound to a project class of the
same name, and TypeScript `interface ExtendableEvent extends Event` bound to a
repository type alias. Go still records no hierarchy edge at all.

### What happened to the wrong edges round 1 listed

Round 2 looked up each wrong example round 1 listed (up to five per language, so
this is a subset of the wrong edges, not all of them).

| Language | Examples | Gone | Right target | Lower tier | Still wrong |
|---|---:|---:|---:|---:|---:|
| go | 6 | 0 | 0 | 0 | 6 |
| java | 11 | 0 | 5 | 0 | 6 |
| typescript | 8 | 3 | 0 | 0 | 5 |
| tsx | 3 | 0 | 2 | 0 | 1 |
| javascript | 9 | 9 | 0 | 0 | 0 |
| python | 20 | 12 | 0 | 0 | 8 |
| ruby | 8 | 6 | 0 | 0 | 2 |
| c | 2 | 1 | 1 | 0 | 0 |
| cpp | 8 | 1 | 0 | 5 | 2 |
| c_sharp | 10 | 2 | 1 | 3 | 4 |
| rust | 4 | 2 | 0 | 0 | 2 |
| php | 9 | 4 | 0 | 3 | 2 |
| kotlin | 13 | 3 | 4 | 2 | 4 |
| scala | 10 | 1 | 6 | 2 | 1 |
| **all** | 121 | 44 | 19 | 15 | 43 |

"Lower tier" means the edge is still wrong but now carries a lower tier, so it
misleads less; "still wrong" means the same edge at the same tier.

### The answers an agent gets

Three questions per language. Round 1: the first line was judged true of the code
in 33 of 41 answers. Round 2: 40 of 42.

Not true in round 1: cpp (2), go, javascript, kotlin, python, rust, scala. Not true in round 2: go, typescript. In round 1, all but the Go
answer stated a count of callers, subtypes or direct uses that the code contradicts
without saying so; the Go answer said `UNKNOWN` and admitted that the index records
no implementation, which is true and still does not answer the question. In round 2
Go is unchanged and the TypeScript answer lists a function as a subtype. Several
round 2 answers were judged true but carried a warning that did not apply: names
found in documentation, import lists and generic constraints were counted as places
a function might be passed as a value. Those false alarms are listed under defects.

### Index time

| Language | Files | Symbols R1 / R2 | Index seconds R1 / R2 | Resolve seconds R1 / R2 |
|---|---:|---:|---:|---:|
| go | 66 | 1,030 / 934 | 1.36 / 1.7 | 0.22 / 0.27 |
| java | 50 | 1,251 / 1,251 | 0.95 / 1.47 | 0.31 / 0.36 |
| typescript | 565 | 4,768 / 4,806 | 4.83 / 7.34 | 1.12 / 1.75 |
| tsx | 198 | 1,736 / 1,736 | 1.17 / 1.39 | 0.23 / 0.26 |
| javascript | 111 | 500 / 500 | 0.72 / 0.88 | 0.16 / 0.2 |
| python | 178 | 2,989 / 2,949 | 1.37 / 1.71 | 0.32 / 0.4 |
| ruby | 186 | 1,691 / 1,752 | 1.28 / 1.94 | 0.39 / 0.5 |
| c | 399 | 1,743 / 1,753 | 1.86 / 2.26 | 0.35 / 0.43 |
| cpp | 224 | 1,966 / 2,024 | 60.94 / 3.06 | 0.22 / 0.28 |
| c_sharp | 282 | 2,051 / 1,957 | 1.81 / 2.27 | 0.23 / 0.33 |
| rust | 60 | 972 / 1,009 | 0.6 / 1.15 | 0.15 / 0.3 |
| php | 262 | 2,222 / 2,222 | 1.16 / 1.7 | 0.39 / 0.47 |
| kotlin | 183 | 2,783 / 2,740 | 1.72 / 2.43 | 0.58 / 0.79 |
| scala | 102 | 931 / 861 | 0.87 / 0.98 | 0.23 / 0.29 |

Index and resolve times are the checkers' single runs; the load on the machine
is not recorded, so read them as orders of magnitude. The one large difference is
C++ (60.94 s to 3.06 s): in round 1 `benchmark.cc` hit its iteration limit, `benchmark_test.cc`
hit the 1-minute wall-clock ceiling and `re.h` was stopped early.

## Definition extraction, 36 more languages

Build `f783fa5`. 119 files in all. Precision here is correct / (correct + spurious);
recall is correct / expected. "Known gap" is the count of misses the
language's `KNOWN_GAPS.md` already lists. "Line on attribute" counts correct
symbols whose recorded line is an annotation or description above the
declaration.

| Language | Project | Commit | Files | Expected | Correct | Missed | Known gap | Spurious | Precision | Recall | Line on attribute | Parsed ok |
|---|---|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| bash | Bash-it/bash-it | 72ea9e2 | 3 | 161 | 161 | 0 | 0 | 1 | 99% | 100% | 0 | 3/3 |
| clojure | clojure/clojure | 33fb064 | 3 | 138 | 138 | 0 | 0 | 0 | 100% | 100% | 0 | 3/3 |
| cmake | opencv/opencv | d45a6e48 | 3 | 19 | 19 | 0 | 0 | 0 | 100% | 100% | 0 | 3/3 |
| css | pure-css/pure | 24084b96 | 3 | 50 | 50 | 0 | 0 | 0 | 100% | 100% | 0 | 2/3 |
| dart | dart-lang/shelf | b175e9f | 3 | 52 | 52 | 0 | 0 | 0 | 100% | 100% | 0 | 1/3 |
| elixir | phoenixframework/phoenix | 789f851 | 3 | 135 | 134 | 1 | 0 | 0 | 100% | 99% | 0 | 3/3 |
| erlang | ninenines/cowboy | 29c1f22 | 3 | 121 | 91 | 27 | 0 | 3 | 97% | 75% | 0 | 3/3 |
| fortran | fortran-lang/fpm | 73bfaa0 | 3 | 50 | 46 | 0 | 4 | 0 | 100% | 92% | 0 | 3/3 |
| fsharp | demystifyfp/FsToolkit.ErrorHandling | d7110b0 | 3 | 186 | 23 | 115 | 0 | 48 | 32% | 12% | 0 | 0/3 |
| gdscript | godotengine/godot-demo-projects | b761b4cd | 3 | 102 | 101 | 0 | 1 | 0 | 100% | 99% | 0 | 3/3 |
| graphql | 99designs/gqlgen | gqlgen 4c4858a4 ; swapi-graphql 8c0bf586 | 4 | 95 | 95 | 0 | 0 | 5 | 95% | 100% | 52 | 4/4 |
| groovy | grails/grails-core | 57f8d989 | 3 | 127 | 72 | 55 | 0 | 0 | 100% | 57% | 56 | 0/3 |
| hcl | terraform-aws-modules/terraform-aws-vpc | b3abd6df | 4 | 351 | 351 | 0 | 0 | 0 | 100% | 100% | 0 | 4/4 |
| html | mozilla/pdf.js | bf3e12f1 | 4 | 375 | 375 | 0 | 0 | 0 | 100% | 100% | 0 | 4/4 |
| json | SchemaStore/schemastore | cbba8f7b | 3 | 46 | 46 | 0 | 0 | 0 | 100% | 100% | 0 | 3/3 |
| julia | JuliaStats/StatsBase.jl | 4866245 | 3 | 185 | 174 | 9 | 2 | 0 | 100% | 94% | 0 | 2/3 |
| less | Semantic-Org/Semantic-UI | 597843ab | 3 | 6 | 6 | 0 | 0 | 0 | 100% | 100% | 0 | 3/3 |
| lua | luarocks/luarocks | 85bfe36 | 3 | 76 | 76 | 0 | 0 | 0 | 100% | 100% | 0 | 3/3 |
| make | micropython/micropython | d2d095d0 | 3 | 142 | 22 | 120 | 0 | 0 | 100% | 15% | 0 | 1/3 |
| markdown | rust-lang/book | 2771df16 | 3 | 42 | 42 | 0 | 0 | 0 | 100% | 100% | 0 | 3/3 |
| matlab | altmany/export_fig | c6c48b0 | 3 | 27 | 27 | 0 | 0 | 0 | 100% | 100% | 0 | 3/3 |
| nim | dom96/jester | 8a491a8e | 3 | 51 | 51 | 0 | 0 | 0 | 100% | 100% | 0 | 3/3 |
| objc | AFNetworking/AFNetworking | d9f589c | 3 | 118 | 80 | 38 | 0 | 13 | 86% | 68% | 0 | 1/3 |
| ocaml | janestreet/base | 49a861e | 3 | 408 | 311 | 97 | 0 | 0 | 100% | 76% | 0 | 3/3 |
| powershell | dahlbyk/posh-git | bbc5ac3 | 3 | 52 | 28 | 1 | 0 | 24 | 54% | 54% | 0 | 2/3 |
| proto | protocolbuffers/protobuf | protobuf 24c2df63 ; grpc/googleapis: raw master download, no commit | 8 | 95 | 95 | 0 | 0 | 0 | 100% | 100% | 0 | 8/8 |
| r | tidyverse/dplyr | 819c99f | 3 | 55 | 55 | 0 | 0 | 0 | 100% | 100% | 0 | 3/3 |
| scss | jgthms/bulma | c5cd04ba | 3 | 120 | 118 | 2 | 0 | 10 | 92% | 98% | 0 | 0/3 |
| solidity | OpenZeppelin/openzeppelin-contracts | efaa6f78 | 3 | 134 | 124 | 10 | 0 | 0 | 100% | 93% | 0 | 3/3 |
| sql | supabase/postgres | 47dbfca | 3 | 11 | 9 | 2 | 0 | 0 | 100% | 82% | 0 | 0/3 |
| svelte | huntabyte/shadcn-svelte | 28f5a3be | 3 | 61 | 61 | 0 | 0 | 2 | 97% | 100% | 0 | 3/3 |
| swift | Alamofire/Alamofire | b3e74e8 | 3 | 219 | 179 | 40 | 0 | 0 | 100% | 82% | 31 | 1/3 |
| toml | helix-editor/helix | helix a714a818 ; starship: raw master download, no commit | 5 | 110 | 110 | 0 | 0 | 0 | 100% | 100% | 0 | 5/5 |
| vue | vuejs/vuetify | vuetify 4932ba8b ; vue-element-admin 6858a9ad | 4 | 66 | 66 | 0 | 0 | 1 | 99% | 100% | 0 | 4/4 |
| yaml | gitlab-org/gitlab-foss | b8cb14c7 | 3 | 10 | 10 | 0 | 0 | 0 | 100% | 100% | 0 | 3/3 |
| zig | zigtools/zls | 066658e | 3 | 107 | 97 | 10 | 0 | 0 | 100% | 91% | 0 | 3/3 |
| **all** |  |  | 119 | 4103 | 3495 | 527 | 7 | 107 | 97% | 85% | 139 |  |

"Parsed ok" counts files whose `parse_status` is `ok`: read to the end with no
syntax error. A file can be `partial` and lose nothing (CSS and Dart did), and
a file can lose most of its definitions while the status says only `partial`
(Make, Groovy, Swift, F#).

Languages with no missed and no spurious symbol in the files sampled (18 of 36): clojure, cmake, css, dart, fortran, gdscript, hcl, html, json, less, lua, markdown, matlab, nim, proto, r, toml, yaml.
Several of those samples did not exercise the whole rule; each result file says which
forms its three files did not contain.

The rest, with the cause the check recorded. Figures are correct / expected from the table.

- **fsharp** (23 of 186, 48 spurious): members declared `member inline` that take arguments were recorded as fields, not methods (16 per file), and no type, module or namespace header was extracted in any of the three files.
- **make** (22 of 142): two of the three files (the `stm32` and `mimxrt` Makefiles of micropython) yielded no symbol at all. One line using a substitution reference, `$(BOARD_DIR:/=)`, makes the grammar stop building rule and assignment nodes for the rest of the file. About 120 definitions were lost.
- **groovy** (72 of 127): one 794-line class with an annotation stack and `implements` yielded none of its 50 definitions. All three files were `partial`.
- **objc** (80 of 118, 13 spurious): the grammar fails from the first `@interface` in `AFURLSessionManager.h` (97 error nodes) and none of its 38 declarations was extracted; 12 `FOUNDATION_EXPORT` extern declarations were recorded as variables.
- **powershell** (28 of 52, 24 spurious): 24 functions declared `function script:name` were recorded with the scope prefix in the name, so a lookup of the bare name finds nothing.
- **erlang** (91 of 121): only functions were extracted. No `-module`, no `-type` or `-opaque`, no `-define`, and the three `-record` declarations came out as functions.
- **ocaml** (311 of 408): items inside `include struct ... end` and `include sig ... end` were not extracted, which hides about 54 definitions in one file and about 39 in another.
- **swift** (179 of 219): `DownloadRequest.swift` yielded 13 of its 41 definitions (97 error nodes from line 1); `Combine.swift` lost the members of three `private final class Inner` bodies and two `private typealias` lines.
- **julia** (174 of 185): `signalcorr.jl` is `partial` and lost 7 of its 38 definitions, six of them top-level short-form functions; two short-form operator methods in `weights.jl` were missed.
- **solidity** (124 of 134): constants (`constant` and `immutable` state variables) were never extracted, 6 of the 10 misses.
- **zig** (97 of 107): in a file that is itself a struct (`const Self = @This();`) every file-level field was missed (10).
- **scss** (118 of 120, 10 spurious): all three Bulma 1.x files are `partial`; two mixins were lost to error recovery.
- **sql** (9 of 11): a `DO` block containing two `CREATE FUNCTION` statements yielded nothing. Only functions were exercised; the files held no table, view or type.
- **graphql** (95 of 95, 5 spurious): enum values were recorded as functions; a type with a description is recorded at the description line (52 of 95 symbols).
- **vue** and **svelte** (1 and 2 spurious): a method or getter in an object literal inside a function body was recorded as a method of the component.
- **elixir**, **bash**, **fortran**, **gdscript**: one miss or spurious symbol each, or only misses that the language's `KNOWN_GAPS.md` lists (fortran 4, gdscript 1).

### F# and Groovy with the source mask

Measured on the final build; figures are added when the runs complete.

## Large repositories

Two repositories far larger than anything above, indexed with the tool and
queried through the command line. Both runs were on a machine carrying other
work (the load average is in the result files), so times are upper bounds.
Resident memory is `maximum resident set size` from `/usr/bin/time -l`.

### Kubernetes (build `bdc86c9`)

Result file: `large-repositories/results-kubernetes-build-bdc86c9.txt`.

| Measure | Value |
|---|---|
| Files in the index | 16486 |
| Symbols | 274,838 |
| Relationships | 238,507 |
| Parsed | 15196 (15109 clean, 87 with syntax errors) |
| Index time (wall / user / sys) | 259.76 / 650.43 / 150.60 s |
| Index peak resident memory | 2.39 GB |
| Resolve time (wall / user / sys) | 57.39 / 71.87 / 6.26 s |
| Resolve peak resident memory | 0.27 GB |
| Index size on disk | 636.9 MiB |
| Files over the size limit | 5 |
| `find_callers NewKubeletCommand`, CLI, one process | 4.2 s |

### VS Code (build `a75b887`, schema v15)

Result file: `large-repositories/results-vscode-build-a75b887.txt`. This build
predates the second look at names over the candidate cap (commit `3ca1c04`, which
names this repository), the stored coverage figures (`6ba89a1`) and the answer
indexes (`5c77f65`). Read the numbers below as the before; the after is the
sub-section "Large repositories on the final build".

| Measure | Value |
|---|---|
| Files discovered / indexed / skipped | 20448 / 18757 / 1691 |
| Symbols | 443,519 |
| References | 1,885,744 |
| Parsed | 18757 (18638 clean, 119 with syntax errors) |
| Index time (wall / user / sys) | 342.24 / 652.12 / 147.29 s |
| Index peak resident memory | 1.54 GB |
| Resolve time (wall / user / sys) | 172.14 / 178.90 / 22.29 s |
| Resolve peak resident memory | 0.60 GB |
| References resolved, by confidence | high 417,659, medium 40,731, low 12,112 |
| Ambiguous | 598,513, of which 426,853 had more candidates than the cap allows and were left unbound |
| Unresolved | 427,878, of which 285,976 shown to leave the repository |
| Edges | 445,753 (high 417,659, medium 19,291, low 8,803) |
| Index size on disk | 1.1 GiB |

Questions asked of that index, as the result file records them:

```text
(CLI, one process each, load ~8):
find_implementations IDisposable (lifecycle.ts): 7.9 s, "0 implementing" (all 651 references named IDisposable are ambiguous: 11 definitions, over the candidate cap)
find_callers createDecorator (instantiation.ts): 9.1 s, 750 callers
analyze_impact createDecorator: 12.2 s; find_tests: 12.6 s; find_callers toDisposable: 9.6 s (2318 callers)
get_task_context: 17.3 s; search_code: 0.10 s; list_dead_code: 11.1 s
resolve: 426,853 references had more candidates than the cap allows and were left unbound; ambiguous 598,513; supertype refs ambiguous 7,853 vs resolved 13,635
```

Plain-name questions on that index (`find_callers name=createDecorator`,
`find_callers name=dispose`, `find_implementations name=IDisposable`) were
answered in about 0.1 s with `AMBIGUOUS`, a list of the definitions that share
the name, because a name shared by 3, 2,722 and 11 definitions selects none of
them. `list_dead_code` took 11.05 s and opened `UNRELIABLE` because only 59% of the references that name
something the index defines were bound.

What the VS Code run shows is unflattering and is the reason the later changes
were made: most of the unbound references are to names with more candidates than
the cap (426,853 of 598,513 ambiguous references), and calls made on a value
rather than on a named module need type inference the tool does not do.

### Large repositories on the final build

Measured on the final build; figures are added when the runs complete.

## The final build against the round-2 build

Build `8748b29` (the one round 2 measured) against build `61f5992`, on the 14 projects
at the commits both rounds used. Each project was indexed and resolved by both
builds and every edge and every symbol compared. Files: `edge-comparison/`.

| Language | Project | Commit | Edges before | Edges after | Added | Removed | Symbols added | Symbols removed |
|---|---|---|---:|---:|---:|---:|---:|---:|
| c | jqlang/jq | 5c001b1 | 5,108 | 5,106 | 0 | 2 | 0 | 2 |
| c_sharp | serilog/serilog | 43105b3 | 2,326 | 2,326 | 0 | 0 | 0 | 0 |
| cpp | google/benchmark | cf4d7fe | 1,203 | 1,203 | 1 | 1 | 0 | 13 |
| go | spf13/cobra | adbc881 | 2,146 | 2,146 | 0 | 0 | 0 | 0 |
| java | square/javapoet | b9017a9 | 2,855 | 2,855 | 0 | 0 | 0 | 0 |
| javascript | koajs/koa | b7ead6f | 40 | 40 | 0 | 0 | 0 | 0 |
| kotlin | square/moshi | 889013e | 5,814 | 5,824 | 97 | 87 | 0 | 0 |
| php | Seldaek/monolog | 3bed304 | 3,163 | 3,162 | 0 | 1 | 0 | 0 |
| python | pallets/click | e38188e | 3,206 | 3,206 | 0 | 0 | 0 | 0 |
| ruby | rack/rack | da90464 | 1,305 | 1,305 | 0 | 0 | 0 | 0 |
| rust | sharkdp/fd | 24d90b2 | 784 | 784 | 0 | 0 | 0 | 0 |
| scala | com-lihaoyi/os-lib | 8871ab0 | 1,853 | 1,853 | 0 | 0 | 0 | 0 |
| tsx | tremorlabs/tremor | ca4d588 | 1,254 | 1,247 | 0 | 7 | 0 | 0 |
| typescript | honojs/hono | f6fbaf0 | 2,078 | 2,058 | 8 | 28 | 0 | 0 |

Identical edge set and symbol set: c_sharp, go, java, javascript, python, ruby, rust, scala (8 of 14).

Each change, from the files. An edge is identified by its kind, call site and
target; "same site, different target" means the call site had an edge in both
builds but to a different definition.

**c (jqlang/jq).**

- Edges: 2 edges removed with no replacement (2 medium CALLS).
  Both removed edges are `clearerr(...)` calls in `src/util.c` (lines 271 and 306) that had bound to a function of the same name in `src/inject_errors.c`, a test-only override of the C library function. Commit `0e72738` lists `clearerr` among the C library names.
  Symbols removed: `Makefile.am` line 138, kind function, named `if`; `Makefile.am` line 36, kind variable, named `if`. Both were make directives in `Makefile.am` read as a variable and a function named `if` (commit `c5d9055`).

**cpp (google/benchmark).**

- Edges: 1 edge changed tier on an unchanged target (1 low to high).
  The one tier change is `INSTANTIATES` of `PerfCounters` at `src/perf_counters.h` line 97, low to high.
  13 symbols removed, all kind `function`: `CheckHandler` x2, `LambdaBenchmark`, `LogType`, `PerfCounters` x3, `ScopedPauseTiming` x2, `TestReporter`, `ThreadAffinityGuard` x2, `ValueUnion`. Commit `1d84158`: `X() = default;` and `X(const X&) = delete;` were recorded as functions named after the class, and are declarations without a body.

**kotlin (square/moshi).**

- Edges: 26 edges changed tier on an unchanged target (26 low to high); 61 edges at the same site now reach a different target (59 high INSTANTIATES, 2 high CALLS); 10 edges at sites that had none (8 high CALLS, 2 high INSTANTIATES).
  59 `INSTANTIATES` edges changed target. 58 of them went from `KotlinJsonAdapterFactory` in `com/squareup/moshi/` to the one in `com/squareup/moshi/kotlin/reflect/`, the class the calling files import; commit `2503e23` describes the first as a deprecated shim. The other 1 is `Rfc3339DateJsonAdapter`, from `com/squareup/moshi/` to `com/squareup/moshi/adapters/`.
  10 `high` `CALLS` edges to top-level functions of `internal/Util.kt` (`createJsonQualifierImplementation`, `getSupertype`, `knownNotNull`, `missingProperty`, `unexpectedNull`) appear where there was none or where a `low` edge to `AdapterGenerator.unexpectedNull` had been.

**php (Seldaek/monolog).**

- Edges: 1 edge removed with no replacement (1 medium EXTENDS).
  The removed edge is `EXTENDS` (medium) from `JsonSerializableDateTimeImmutable` to `Monolog\DateTimeImmutable`: the source says `extends \DateTimeImmutable`, the PHP built-in, and the leading backslash had been dropped. Commit `aba5f47`.

**tsx (tremorlabs/tremor).**

- Edges: 7 edges removed with no replacement (7 high CALLS).
  The removed edges bind a JSX tag written under an alias of an external package (`Recharts*` and similar) to a component of the same name in the repository: `AreaChart` x1, `BarChart` x1, `Legend` x4, `LineChart` x1. Commit `7f66c15`.

**typescript (honojs/hono).**

- Edges: 8 edges at the same site now reach a different target (8 high CALLS); 20 edges removed with no replacement (20 high CALLS).
  Of the removed edges, 15 point at a method of `ClientResponse` (a repository type) from calls on a platform `Response` or `Request` (commit `3ecf7fd`), and 13 point at a function in the same file as the call, an import alias bound to the function it is written in (commit `7f66c15`). 8 edges at the same sites now reach the function the file imports.

These changes agree with what the independent readings of round 2 found wrong:
the clearerr and `if` defects in C, the `= default` functions in C++, the
`\DateTimeImmutable` edge in PHP, the import-alias edges in TSX and
TypeScript, the `clone()` edge in TypeScript, and the shim binding in Kotlin.
Nothing here was re-read by an independent check: the final build's precision is
inferred from the edge difference, and any new wrong edge it added at a site
that had none would not show as a regression.

## Defects found, and what was done

Each row names a defect found by a check in this record and the commit that
changed it. "Verified" means the round 2 check read the result and found it fixed.
"Fixed since" means the commit came after round 2's build (`8748b29`); the evidence is the
commit and the edge comparison above, and no independent check has re-read it.

| Language | Defect | Fix | Status |
|---|---|---|---|
| C | Functions named after control-flow keywords (`if`, `switch`) | `e30a52a` | Verified: no C symbol named `if` or `switch` in round 2 |
| C | C library calls bound to a test fault-injection interposer: `fgets` and others | `68c53e3` | Verified for `fgets`, `fclose`, `ferror`, `fread`, `fwrite` |
| C | `clearerr` still bound to the interposer (2 medium edges) | `0e72738` | Fixed since; both edges absent in the edge comparison |
| make | `if` from automake conditionals recorded as a variable and a function | `c5d9055` | Fixed since; both symbols absent in the edge comparison |
| C# | A file with `#if` default-interface blocks read as errors, the interface missing, return types and `if` recorded as methods | `2790380`, `c35fb54`, `621788e` | Verified: round 2 definitions 299 expected, 299 correct, 0 spurious |
| C# | `new X(...)` never recorded, so no `INSTANTIATES` | `c35fb54`, `6bbd209` | Verified: constructor sites found in round 2 |
| C#, Java, Kotlin, Scala, C++ | An overloaded name bound to its first declaration at a high tier | `1b72934`, `300282d` | Partly: now `low` unless the argument count decides. Argument types are not used, so the wrong overload is still bound (see below) |
| C# | A receiver named like a static class bound to the class (`log.Warning` to `Log.Warning`), 12 wrong of 21 medium edges read | `17d7afa` | Verified moved to `low`; the edges are still wrong at `low` |
| C++ | Definitions lost behind `#if` groups and macro-exported classes; a stalled parse held the 60 s ceiling | `2790380`, `60a2594`, `6ebb8f2` | Verified: index 60.94 s to 3.06 s, no incomplete file; `sysinfo.cc` 30 of 38 definitions against 20 of 39 |
| C++ | Test and registration macros recorded as functions named `TEST` | `9d3c251` | Verified: named `Suite.Name` |
| C++ | `= default` and `= delete` declarations recorded as functions | `1d84158` | Fixed since; 13 symbols absent in the edge comparison |
| Go | Function-local `var` and `const` recorded as file symbols | `ab21996` | Verified: spurious 13 to 0 |
| Java | `this(...)` and `super(...)` delegation not recorded | `1b91995` | Verified: `CodeWriter` recall 8 of 8 (was 6) |
| JavaScript | A library function bound to a same-named getter or method of the file (`vary`, `fresh`, `accepts`, `new URL`) | `adfc2b4` | Verified: all 9 listed edges gone, no right edge lost |
| Kotlin | No parameter counts, top-level properties not symbols, nested class constructors unresolved, production code bound to Java test classes | `300282d`, `6fe4c32`, `0eb29a2`, `2508dd3` | Verified: definitions 218 of 218, recall 40 of 40 |
| Kotlin | `KotlinJsonAdapterFactory()` bound to the deprecated shim in an outer package instead of the imported class (3 of 30 `high` sampled) | `2503e23` | Fixed since; 59 `INSTANTIATES` edges changed target in the edge comparison, 58 of them from the shim to the imported class |
| PHP | A call on an object whose class overrides a method bound to the base method at `high`; PHPUnit mock builder bound to `LogRecord::with` | `17bbca5`, `80957d1` | Verified: now `low` with the descendants named; mock calls bind to nothing |
| PHP | Supertype written `\X` bound to a class of the same name in the file's namespace | `aba5f47` | Fixed since; the one edge absent in the edge comparison. Calls and `new` written with a leading backslash are not covered |
| Python, Ruby, Rust, Kotlin | Library names bound to a lone repository namesake (standard-library attribute calls in Python, `format!` in Rust, `require` in Kotlin) | `68c53e3`, `166d98a` | Verified: the 8 Python examples round 1 listed are gone, and `format!` binds to nothing in Rust. Library calls bound to repository members survive at lower tiers in Scala and Ruby |
| Python | Subscripted base classes (`ParamType[str]`) gave no `EXTENDS` edge | `6780369` | Verified: declared relationships found 30 of 30 (was 13 of 30) |
| Ruby | Setters, operators and upper-case methods not symbols; `include` bound as a call; `super` not recorded | `6085cae` | Verified: definitions 327 of 327 |
| Rust | `format!` bound to a test function; statics, associated types and inline modules not symbols; `macro_rules!` invocations unbound | `68c53e3`, `8a3ddbc`, `969c25a` | Verified: definitions 137 of 137 |
| Scala | Calls written `os.name(...)` unresolved or bound to the wrong object; `java.io` supertypes bound to same-named nested classes | `a7c2352`, `166d98a`, `6f0bb23`, `db6aea1` | Verified: `high` 30/30, supertype swap gone |
| TypeScript | `#private` members and namespaces not symbols; `this.#x()` unrecorded | `7fc00ef` | Verified: definitions 291 of 291, recall 25 of 32 (was 19) |
| TypeScript, TSX | Round 2 found the fix for renamed imports (`7fc00ef`, `3a48430`) bound an alias of an external package to a same-named repository symbol, often the enclosing function: 7 wrong `high` edges in TSX, 13 in TypeScript | `7f66c15` | Fixed since; the 7 TSX edges and the 13 TypeScript alias edges round 2 listed are absent in the edge comparison |
| TypeScript | A platform `Response` or `Request` method bound to a repository `ClientResponse` method | `3ecf7fd` | Fixed since for the cases the edge comparison shows |
| Answers | A function reached only because it is passed by name (koa's `respond`) reported with certainty high and no test reaching it; "0 direct" stated on the first line while low-confidence callers existed (C++ `IsZero`); a risk of `low` over an analysis cut at its depth (Go) | `6cab1f1` | Verified for the examples round 1 gave |
| Answers | A warning that a function may be passed as a value, fired by documentation, import lists, generic constraints and the function's own prototype | `9786de1`, `de731ca` | Fixed since; not re-read |
| Answers | A function listed as a subtype because of an anonymous class inside it | `f7a96bb` | Fixed since; not re-read |
| Answers | A correct, complete answer told not to be presented, because the language-wide rate was low | `e18ba91` | Fixed since; not re-read |
| Answers | Three sentences in `analyze_impact` and `find_callers` answers that were untrue | `aa59ce5` | Fixed since; not re-read |
| Definitions | Make, Solidity, GraphQL, SCSS, PowerShell, Erlang, Zig, Objective-C, Vue, Svelte, Bash, Elixir: missed or spurious symbols the 36-language check listed | `1235cd5`, `c42506d`, `d7fccbe`, `1f080ba`, `983b431`, `77e6620`, `2f62d1a`, `3f24c89`, `3046876`, `a66c5a5`, `997bd6b` | Fixed since the 36-language check; not re-read by an independent check |
| Definitions | F#, Groovy, Swift, OCaml, Julia: definitions lost to constructs the grammar cannot read | `3152579`, `71df469`, `24233f3`, `9f52948`, `8cc57c2` | Fixed in part; figures for F# and Groovy are in the placeholder above |
| Scale | On VS Code every one of 651 references to `IDisposable` stayed ambiguous because the name has 11 definitions, over the cap | `3ca1c04`, `7736265` | Fixed since the VS Code run above; the commit reports 30 of 30 `high` and 28 of 30 `medium` new edges read right on a sample from VS Code. The final-build run is the placeholder above |

Not every defect had a fix. See the next section.

## What is still wrong

From round 2, the edge comparison, the 36-language check and the commits. Each
figure comes from the file named.

- **The `low` tier is poor.** Right / judged in the round 2 sample: Python 8/30, Kotlin 9/30, C# 14/30, Java 18/30. The dominant causes are overloads of the same arity (the binder uses the argument count and not the argument types) and calls on a value whose type is unknown.
- **Same-arity overloads.** Round 2 wrong low edges: Java 12 of 30 (round 1 10), Kotlin 21 of 30, C# 15 of 30. C++ `FormatKV` calls (47 references) all still bind to the first overload, now at `low`.
- **Go has no interface-implementation edges.** The index held 0 `EXTENDS` or `IMPLEMENTS` edges for Go in both rounds, so `find_implementations` answers `UNKNOWN`.
- **A call at module level has no caller symbol,** so no edge. Resolved references with no enclosing symbol, measured 13 November before the final build: express 2,015 of 2,180, zod 2,952 of 7,540, taxonomy 20 of 429 (`docs/accuracy.md`, module-level calls).
- **Method calls in a chain that starts from `new` or a call** (`new LoggerConfiguration().WriteTo.Sink(s)`) are not extracted in C#: 4 of 8 recall sites in one file (round 2, C#).
- **Calls on a value need type inference.** Java calls on a receiver whose type the index records are left unresolved (4 of 40 recall sites). TypeScript `c.text` and `this.html` style calls: 7 of 32 recall sites, all ambiguous or unresolved. On VS Code, 426,853 references were left unbound for having more candidates than the cap (build `a75b887`; the final-build figure is the placeholder above).
- **A TSX arrow function followed by a ternary on the next line** (`const calculateHeight = (height: number | undefined) =>` then `height ? ... : ...`) is read as a syntax error: 4 chart files, 4 definitions missed (round 2, TSX).
- **A Scala method the grammar drops without a message.** `SubProcess.joinPumperThreadsHook` (`private[os] def`, `SubProcess.scala` line 218) was extracted in round 1 and is not in round 2.
- **Kotlin secondary constructors are not extracted** (commit `0127387`).
- **A receiver named like a type** still binds a standard-library object to a repository type in Ruby at `high`: `@parser.parse(uri)` on a `URI::Parser` bound to `Multipart::Parser.parse` (round 2, Ruby). In C# the rule was moved to `low`, where its edges are still wrong.
- **Hierarchy kind.** Kotlin `class A : B` where `B` is a class without a constructor call is recorded `IMPLEMENTS`, not `EXTENDS` (2 of 20 sampled, target correct). Scala traits are recorded `EXTENDS`; the 4 `object X extends XMacros` relationships in the five Scala files have no edge.
- **TSX and TypeScript.** `SingleProps extends Omit<PickerProps, ...>` has no edge. TypeScript `c.header(...)` on a `Context` binds to `HonoRequest.header`: 17 of the 54 edges into that method come from receivers `c`, `ctx` or `this`.
- **JavaScript.** Calls through `this` inside object-literal methods are still not bound (14 of 31 recall sites). Only 40 `CALLS` edges exist in the 111-file koa project, and 59.3% of the references that name something defined are bound, so every JavaScript answer carries the `UNRELIABLE` word.
- **Definitions in languages whose grammar cannot read common constructs.** With build `f783fa5`: F# 23 of 186 expected found correctly (48 spurious, mostly members of the wrong kind); Groovy 72 of 127 (one 794-line file gave none of its 50); Swift 179 of 219; Julia 174 of 185; Make 22 of 142; Objective-C 80 of 118; OCaml 311 of 408. Later commits address each (see defects); the figures after them are not in this record except where the placeholders say.
- **C and C++ headers with `#if` through a declaration or an initialiser list** lose every definition after it (`fmt/core.h`, `leveldb/util/env_posix.cc`; `docs/accuracy.md`, C and C++ section). A guard refuses a mask that would blank most of a file (commit `60a2594`: `fmt`'s `test/enum-test.cc` went from 23 definitions to 1 before it).
- **Plain `.txt` files are not text-searchable.** A `.txt` file with no Vim help markers has no grammar and is skipped (commit `2695151`).
- **Eight languages were left without a definition query on purpose:** Perl, Haskell, Pascal, COBOL, Elm, Nix, Verilog and Assembly. `repocortex languages --verbose` on the final build lists every one of them under "text only", so their files are searchable by text and yield no symbols.
- **Platforms.** `install.ps1` has not been executed on Windows (commit `d28e6a4`: "Not verified"). The workflow `ci.yml` runs the tests on Linux, macOS and Windows; this record has no result from any of those runs for the work described here.
- **Unmeasured since the changes:** the demo images in `docs/assets` were last regenerated at v1.0.1 and the portal screenshots on 11 November.

## Limits of this record

- Samples are small: five files per project, at most 40 recall sites, 30 edges per
  tier. They show direction and systematic failure, not a rate. A 95% interval
  around 50% on a sample of 30 is plus or minus 18 points.
- One project per language. A different project would give different figures.
- The 14 projects of round 2 are the 14 of round 1, at the same commits and with
  the same files. Round 2 shows that the fixes held on code the fixes were found
  on; it is not a fresh sample and it says little about how the tool does on a
  project nobody has looked at. Some fixes were made after reading these exact
  projects, so their round 2 figures are the least independent in this record.
- The `high` samples drew from the whole tier, test files included. Several
  projects are mostly test code, and the medium tier of some is dominated by test
  helpers, so a project-level figure says little about library code alone.
- The edge comparison and the 36-language check measure different builds from the
  ones round 1 and round 2 measured. Nothing after `8748b29` was re-read by an independent check, except through the edge comparison.
- The 36-language check used three files per language, and for several the
  files did not exercise parts of the rule (the result files say which).
- Nothing here measures the `exact` tier, which needs an imported compiler index.
- Index and resolve times are single runs on a shared machine.
- The large-repository runs are one run each, on a loaded machine, on builds
  that predate later changes made for those repositories.

## Files

| Path | Holds |
|---|---|
| `round1/<language>.json` | Round 1, calls and definitions, 14 files (build `53246ac`) |
| `definitions/<language>.json` | Definition extraction, 36 files (build `f783fa5`) |
| `round2/<language>.json` | Round 2, the same 14 projects (build `8748b29`), with `round1_wrong_examples` and `regressions` |
| `large-repositories/` | The two large-repository result files |
| `edge-comparison/summary.txt` | One line per project: edges and symbols under the two builds |
| `edge-comparison/<language>.added`, `.removed` | Edges only in the final build, and only in the round-2 build, as `kind|tier|file|line|target` |
| `edge-comparison/<language>.sym_added`, `.sym_removed` | The same for symbols, as `file|line|kind|name` |

The briefs the checks followed are not included: they contain the paths of the
machine they ran on.
