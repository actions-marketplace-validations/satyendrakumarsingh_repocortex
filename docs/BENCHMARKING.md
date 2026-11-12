# Benchmarking

How to reproduce every number this project publishes, and what each one does and
does not mean.

This page is operational. The **method** for resolution accuracy — why judgement
is restricted to unambiguous lines, why recall is not reported, where the
remaining errors come from — is in [accuracy.md](accuracy.md), and this extends
that document rather than restating it. The performance numbers and the
measurements that chose each default are in [performance.md](performance.md).

The reason both exist: until they did, every confidence label in this tool was a
design intention rather than a measurement. If you disagree with a number, the
commands below are how you produce a competing one.

---

## What is measurable, and with what

| Claim | Command | Answers |
|---|---|---|
| Confidence tiers mean something | `doctor --resolution-precision` | precision per tier, against compiler ground truth |
| Retrieval returns the right code | `bench <cases.yml>` | recall and precision against hand-written cases |
| Indexing is reproducible | `index --full --digest` | one sha256 over a run's content |
| The graph is internally sound | `doctor --verify-index` | dangling edges, orphans, unproven `exact` claims |
| Operations cost what we say | `index`, `resolve`, query commands | wall clock, printed by the tool itself |
| Impact analysis predicts what a change touches | `scripts/bench/impactaccuracy` | precision, recall and F1 against a repository's own commit history |

The first five run against your repository, not ours. That is the point of
shipping them as commands rather than as a table. The sixth is a harness in this
repository that drives the binary through its command line; it is described in
[section 6](#6-figures-other-tools-publish).

## Prerequisites

```bash
CGO_ENABLED=0 go build -o /tmp/repocortex ./cmd/repocortex   # Go 1.26+
```

Zero CGO is not incidental: it is what makes `go install` work without a C
toolchain, and it means the binary you benchmark is the binary that ships.

Ground-truth indexers, pinned to the versions every published figure was taken
against:

| Tool | Version |
|---|---|
| `scip-go` | 0.2.7 |
| `scip-typescript` | 0.3.14 |
| `scip-python` | 0.6.6 |

`repocortex indexers install --yes` provisions these at pinned versions with
verified digests. `repocortex indexers list` tells you what applies to a
repository and what is missing, without installing anything.

## Corpora

Two documents use different corpora, deliberately, and neither set is
interchangeable with the other.

**[accuracy.md](accuracy.md) — real third-party code.** `--depth 1` clones of
prometheus (Go), nest (TypeScript), scrapy and requests (Python), plus
RepoCortex's own source. Third-party code the resolver was never tuned against is
the only kind that can falsify a resolution claim; RepoCortex's own numbers are
the least trustworthy in the table for exactly that reason, and they are the
highest.

**[performance.md](performance.md) — three corpora, each measuring a different
thing.** `fixtures/payment-platform` (9 files) measures fixed costs — process
start, migrations, transaction setup — rather than throughput. RepoCortex's own
source measures a real mid-size repository. A 5,000-file synthetic Go corpus
(`go5k`, ~20 MB, 210,000 symbols) measures scaling, and is byte-for-byte
reproducible because each file's content is a pure function of its index.

**Do not measure accuracy on the synthetic corpus.** Its identifiers repeat
across packages by construction, so `resolve` reports 0 resolved and 80,000
ambiguous. It measures the throughput of the binding path and says nothing about
binding correctness. For the same reason a synthetic *Java* corpus was
deliberately not used to measure Java: uniform generated code makes resolution
easy and would report an inflated precision, which is worse than no number.

---

## 1. Resolution precision

The measurement that turns the confidence labels from an intention into a
number. Four commands:

```bash
repocortex index && repocortex resolve
scip-go --output index.scip          # compiler ground truth
repocortex scip import index.scip      # lands as REFERENCES/exact/compiler
repocortex doctor --resolution-precision
```

Order matters: index, resolve, then import. A heuristic `CALLS` edge is
adjudicated by asking the compiler what the reference at that source position
resolves to; agreement means both name the same definition.

Reproduced on this repository's own working tree — 476 files discovered, 422
indexed, 6,259 symbols, against a 12.7 MB `scip-go` 0.2.7 index:

```
  TIER    VERDICT  JUDGED  AGREE  DISAGREE  PRECISION
  high    act on   5,190   5,086  104       98.0%
  medium  show     49      17     32        34.7%
  low     ignore   263     0      263       0.0%
  all     —        5,502   5,103  399       92.7%

COVERAGE
  Judged 5,502 of 11,703 heuristic CALLS edge(s) (47.0%); 6,201 were not adjudicated
  at all and are neither right nor wrong above.
  Against 25,736 compiler fact(s), on the 14,280 line(s) the compiler resolved to
  exactly one target; 4,844 ambiguous line(s) were excluded.

LANGUAGES
  LANGUAGE    FILES  CALLS EDGES  COMPILER FACTS  STATUS
  go          399    11,683       25,736          measured
  java        8      18           0               NOT MEASURED — no SCIP index imported for it
  typescript  1      2            0               NOT MEASURED — no SCIP index imported for it
```

[accuracy.md](accuracy.md) records **97.8% / 34.1% / 0.0%** over 4,619 judged
edges for RepoCortex's own source. The run above is the same measurement on a
larger, later tree, and it agrees to within a fraction of a point. **The two
numbers are not comparable as a before/after** — different file counts, different
`scip-go` output, different resolver. That is the single most important
discipline on this page, and it has its own section below.

### What this number is

- **It is precision.** SCIP records that one symbol *references* another, not
  that it *calls* it, so the compiler's edge set is a superset of the resolver's
  by construction. Dividing by it would penalise the resolver for doing exactly
  what it was designed to do. Recall for calls needs a ground truth that
  distinguishes call sites from type references, which `scip-go` does not emit.
- **It is not a claim about proven edges.** Compiler-backed edges are not scored
  against themselves. They are `exact` because a typechecker produced them, and
  `doctor --verify-index` separately enforces that nothing may claim `exact`
  without `source=compiler`.
- **It covers less than the whole graph, and says how much.** 47.0% of heuristic
  `CALLS` edges were adjudicated in the run above. The rest are neither right
  nor wrong in that table.
- **It names what it could not measure.** A language with heuristic edges and no
  imported SCIP index is listed as `NOT MEASURED` rather than omitted. Unmeasured
  is not the same as accurate.

### Why coverage is not higher

Judgement is restricted to lines the compiler resolved to **exactly one target**,
and that restriction is load-bearing because neither looser option is sound.
Matching by line alone is too generous — on RepoCortex's own source, 25.8% of
referenced lines carry more than one distinct target, so a heuristic edge could
be scored against a reference it was never about. Matching by line *and* column
is too strict — the two tools do not share a column convention (for
`svc.Method()` the resolver records the column of `svc` and `scip-go` records
the column of `Method`), and requiring exact columns once discarded two thirds
of the comparison without either tool being wrong.

Everything undecidable is reported as uncovered rather than scored. A
measurement that quietly counts cases it cannot decide is worse than a smaller
one that says so.

### Gating on it in CI

`doctor`'s exit status is deliberately **not** affected by the measurement. Add
`--json` and the report lands under `resolution_precision` alongside the checks,
for you to threshold yourself.

---

## 2. Retrieval quality

```bash
repocortex bench fixtures/bench/payment.yml
repocortex bench fixtures/bench/payment.yml --fail-on-regression   # exit 1 on any failure
```

Reproduced against a copy of the bundled payment fixture:

```
CASE                              RESULT  RECALL  TOKENS  MISSING
change-transaction-status-column  pass    100%    1567    —
idempotent-callback               pass    100%    1368    —
notification-failure-handling     pass    100%    1388    —

3/3 passed, mean recall 100%.
```

A case names a task, the code a correct answer must contain (`expect`), and the
code it must not (`forbid`):

```yaml
- id: idempotent-callback
  task: Add idempotency to payment callback processing
  expect: [CallbackController, TransactionService, TransactionRepository, transactions]
  forbid: [MarketingReport]
  max_tokens: 8000
```

**The `forbid` list is what makes the score mean anything.** Recall alone is
trivially gamed by returning the whole repository, and a forbidden item
retrieved is reported as a precision failure rather than averaged into the mean.

**What it does not measure:** whether an agent completed the task. Only whether
the context contained what a human said it should, and whether it stayed inside
its budget. Those are different claims, and conflating them would make a
retrieval regression indistinguishable from a bad day at the model provider.

To benchmark your own repository, write your own cases. The bundled file is a
worked example, not a standard.

---

## 3. Determinism

```bash
repocortex index --full --digest
```

The digest is a sha256 over one run's content. The property is verified **both
ways**: stable across worker counts, and *sensitive* to a single added symbol. A
digest that never changed would pass the stability half while detecting nothing.

Reproduced on this repository's working tree, three full rebuilds from an empty
database:

```
Digest: 74f82dd64551014cbaad1be87228d2ac720c7ac0ca49ccfd856eb0bfe4a5d54b  13493ms  (workers=1)
Digest: 74f82dd64551014cbaad1be87228d2ac720c7ac0ca49ccfd856eb0bfe4a5d54b   7346ms  (workers=4)
Digest: 74f82dd64551014cbaad1be87228d2ac720c7ac0ca49ccfd856eb0bfe4a5d54b   5731ms  (workers=8)
```

Three rules make this possible, and `internal/diffgraph` inherits all three:
rows are read under an explicit total `ORDER BY` over content columns rather
than by rowid; autoincrement ids are excluded and edges reference their
endpoints by `(file path, fqn, start line, start col)`; and no timestamps are
covered, only content hash and size.

This is not a formality. Running the precision measurement above is what
surfaced a determinism defect nothing else could: three distinct digests across
six repeats over byte-identical source, caused by a "first declaration wins"
rule reading from a `SELECT` with **no `ORDER BY`** — and an unordered `SELECT`
has no first. [accuracy.md](accuracy.md) has the full account. The transferable
rule is that `ORDER BY` is a correctness clause whenever a query feeds a
first-wins rule, and that a digest checked only in CI against small fixtures
will not catch an ordering bug that needs a real name collision to appear.

Also worth running alongside:

```bash
repocortex doctor --verify-index      # graph invariants
go test -race ./...                 # full suite
```

---

## 4. Performance

Every figure in [performance.md](performance.md) names the corpus it came from
and prints three trials rather than an average, because the spread between
trials on a shared developer laptop is the honest picture.

The method, reduced:

```bash
CGO_ENABLED=0 go build -o /tmp/di-docs ./cmd/repocortex
cp -R <corpus> /tmp/work/repo && cd /tmp/work/repo && git init && git commit -am init
for trial in 1 2 3; do
  rm -rf .repocortex && /tmp/di-docs init
  /tmp/di-docs index --full          # full index into an empty database
  /tmp/di-docs index                 # re-index, nothing changed
  echo '// touched' >> <first file>
  /tmp/di-docs index                 # re-index, one file changed
  /tmp/di-docs resolve
done
```

Durations are the ones RepoCortex prints itself, so they exclude process start.
Database sizes are `index.db` on disk. Peak memory is
`/usr/bin/time -l … | grep 'maximum resident set size'`.

Reference points for "is my run in the right range", from performance.md's three
real repositories on `darwin/arm64` native:

| Repository | Files indexed | Symbols | Index time | Peak RSS |
|---|---|---|---|---|
| prometheus (Go) | 1,030 | 21,429 | 8.8 s | 664 MB |
| nest (TypeScript) | 1,929 | 13,916 | 3.7 s | 590 MB |
| spring-boot (Java) | 9,236 | 94,304 | 46.4 s | 756 MB |

Reproduced here on this repository's working tree: 422 files indexed, 6,259
symbols, full index 10.0 s, `resolve` 1.3 s, `scip import` of a 12.7 MB index
1.8 s.

Two things a real repository showed that a synthetic one could not, both
recorded in performance.md: `DefaultDrainThreshold` at 1 GiB meant the pool
never drained on a 1,262-file Go repository and the threshold *was* the peak;
and `MaxInFlightBytes` does not bound memory despite having said so — it bounds
admission, while the drain threshold bounds memory.

---

## 5. Agent benchmark

Does an agent do better with RepoCortex than with search alone? This is the
measurement other code-graph tools publish, and the one that matters most. One
round has been run. Its result is not flattering and is reported as measured.

### Round 1: direct callers of a uniquely named function (2026-11-12)

**Setup.** Five functions in this repository, each with a name defined exactly
once, with 1, 2, 4, 11 and 33 real callers. Each question was asked of two
agents: one limited to ordinary file search and reading, and one also given
`repocortex tool call find_callers`. Ten runs, one per task and arm, the same
model for all (Claude Sonnet), each agent starting cold.

**Ground truth** was computed without RepoCortex: a 90-line Go program walks
every file's syntax tree and lists the functions whose bodies contain a call
expression naming the target. For a name defined once, that is exact.

| Function (callers) | Arm | Correct | Tokens | Tool calls | Seconds |
|---|---|---|---|---|---|
| `staleAgainst` (1) | search only | 1 / 1 | 58,481 | 4 | 14.6 |
| | with RepoCortex | 1 / 1 | 58,808 | 4 | 21.0 |
| `checkListenAddr` (2) | search only | 2 / 2 | 58,546 | 3 | 12.1 |
| | with RepoCortex | 2 / 2 | 59,815 | 4 | 21.5 |
| `mergeJSONKey` (4) | search only | 4 / 4 | 59,641 | 3 | 12.5 |
| | with RepoCortex | 4 / 4 | 59,635 | 4 | 16.8 |
| `openReadOnly` (11) | search only | 11 / 11 | 59,649 | 3 | 21.3 |
| | with RepoCortex | 11 / 11 | 59,477 | 3 | 14.2 |
| `openQuery` (33) | search only | 33 / 33 | 60,958 | 3 | 16.1 |
| | with RepoCortex | 33 / 33 | 61,722 | 5 | 21.0 |
| **Total** | search only | 51 / 51 | 297,275 | 16 | 76.6 |
| | with RepoCortex | 51 / 51 | 299,457 | 20 | 94.5 |

**Result: no advantage.** Both arms were right every time. The agents with
RepoCortex used 0.7% more tokens, four more tool calls and 23% more time,
because each called the tool and then checked its answer with a search.

**What this does and does not show.**

- For a function whose name is unique, one search for `name(` is already a
  complete and exact answer, and an agent finds it in one step. A code graph
  cannot beat that, and this round confirms it does not.
- About 58,000 of each run's tokens are the agent's own fixed overhead, so the
  task itself cost a few thousand tokens in either arm. A benchmark of small
  lookups cannot show a large saving whichever way it goes.
- Five tasks, one run each, in one 400-file Go repository. It is an indication,
  not a statistic.
- It does not test where a search stops being enough: a name defined many
  times (`Close`, `Run`, `handle`), a method reached through an interface, what
  a change reaches several calls away, which tests cover a symbol, or a
  repository too large to search and read. Those are the cases the tool is
  built for, and none has been measured yet.

**Do not quote a token saving for RepoCortex.** There is no measurement that
supports one.

### Round 2, not yet run

The next round needs a ground truth that search cannot produce, which means a
type-checked one: build the truth from a compiler index (SCIP) on public
repositories, then ask for callers of ambiguous method names, the transitive
reach of a change, and the tests that reach a symbol. Report correctness first,
then tool calls and tokens.

### Reproducing round 1

1. Index the repository: `repocortex init && repocortex index && repocortex resolve`.
2. Compute the truth for each target with a syntax-tree walk over the same
   files (exclude `fixtures/`).
3. Give each agent one question: "find every function or method whose body
   directly calls `<name>`, defined in `<file>`; answer `CALLERS: a,b,c`".
   Forbid the search-only arm from running `repocortex` or reading
   `.repocortex`; give the other arm the `find_callers` command.
4. Record tokens, tool calls and wall time from the agent runtime, and score
   the answer line against the truth.

---

## 6. Figures other tools publish

Three figures other code-graph tools publish and this project had not: how well
impact analysis predicts a change, how long an update takes after two files
change, and how long the Linux kernel takes to index. All three were measured on
2026-11-12 at commit `6fadd51`, on an Apple M4 Pro with 14 cores and 24 GiB that
other work was sharing. The harnesses are in `scripts/bench/`.

The two timings are in
[performance.md](performance.md#figures-other-tools-publish-measured-here), with
the commands that produced them. The accuracy measurement is here, because its
method needs more explaining than its result.

### The binary that was measured

```bash
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -trimpath -o /tmp/bench/repocortex ./cmd/repocortex
/tmp/bench/repocortex version     # repocortex dev (commit none, built unknown, darwin/arm64)
```

The Go toolchain on the measuring machine is `darwin/amd64` under Rosetta 2
([pitfall 1](#pitfall-1-check-your-goarch-before-you-benchmark-anything)), so
the binary was cross-compiled to the machine's own architecture. An amd64 build
was not measured.

### Impact-analysis accuracy

**The question.** A commit changed a function. Did `analyze_impact`, asked about
that function at the commit's parent, name the other files the commit went on to
change?

```bash
git clone --depth 400 https://github.com/pallets/flask /tmp/bench/flask
go run -tags bench ./scripts/bench/impactaccuracy \
    -repo /tmp/bench/flask -bin /tmp/bench/repocortex -n 1000 -budget 20m
```

The harness is a Go program behind the `bench` build tag, so the default build,
vet and lint do not compile it. It uses the standard library only and drives the
binary through its command line, so it scores any build, including a released
one. Give it a clone made for the purpose: it checks out revisions and deletes
`.repocortex`. `-out results.jsonl` writes one line per commit examined.

**Method.** For each non-merge commit, newest first:

1. List the files it modified or deleted. A file it added did not exist at the
   parent, so nothing indexed there could name it; added files are left out of
   both sides.
2. Check out the parent, delete `.repocortex`, and run `init`, `index` and
   `resolve`. Every commit is scored against a fresh full index of its own
   parent, which is why the run is slow and why N is small.
3. Keep the changed files the index holds in a language RepoCortex binds
   references for (the `typed` and `resolved` tiers of `repocortex languages`).
   A commit with fewer than two is skipped, and so is one with more than 50.
4. For each such file, the seed symbols are the innermost functions, methods,
   types, fields, constants and variables whose lines overlap a hunk of the
   diff.
5. Call `analyze_impact` for each seed symbol by its fully-qualified name, once
   per confidence floor (`min_confidence=high`, `medium`, `low`). The predicted
   set is the files of the rows returned, minus the seed file.
6. The truth is the other source files the commit changed.

Counts are summed over commits and then divided (micro-averaged). Three
aggregations are printed:

| Aggregation | Seed |
|---|---|
| One seed per commit, any file | The file with the most changed lines |
| One seed per commit, not a test file | The same rule over the files that are not tests |
| Every changed file as a seed | Each file of the commit in turn, against the rest |

The second exists because the first turned out to ask a question with an empty
answer. A commit that fixes a function and extends its test usually changes more
lines in the test, and nothing depends on a test. In gson the first rule chose a
test file in 37 of 65 commits. A test file is recognised by its path (`tests/`,
`*_test.go`, `test_*.py`, `*.spec.ts`, `*Test.java` and the like); the rule is in
`isTestPath`.

Every tally is printed beside a **baseline that needs no index**: predict the
other source files in the seed's own directory. It is scored on the same commits
against the same truth.

**The known bias, in both directions.** Files that change together are not all
dependents of one another, so a correct analysis is charged with "missing" files
it had no reason to name: recall is understated. And a commit does not touch
everything that depends on a change, so a real dependent that kept compiling is
scored as a false positive: precision is understated too. The truth is partial
on both sides. These figures compare one build with another on one history, and
the tool with the baseline. They are not the probability that a reported file
breaks.

**Results.** One seed per commit, not a test file; every row the tool returned.
`medium` is the floor `analyze_impact` applies when none is given.

| Repository | Language | Commits | Floor | Predicted | Correct | Wrong | Missed | Precision | Recall | F1 |
|---|---|---|---|---|---|---|---|---|---|---|
| RepoCortex | Go | 14 | high | 73 | 8 | 65 | 42 | 11.0% | 16.0% | 0.130 |
| | | | medium | 77 | 8 | 69 | 42 | 10.4% | 16.0% | 0.126 |
| | | | low | 77 | 8 | 69 | 42 | 10.4% | 16.0% | 0.126 |
| | | | same directory | 591 | 11 | 580 | 39 | 1.9% | 22.0% | 0.034 |
| flask | Python | 74 | high | 198 | 31 | 167 | 216 | 15.7% | 12.6% | 0.139 |
| | | | medium | 239 | 38 | 201 | 209 | 15.9% | 15.4% | 0.156 |
| | | | low | 239 | 38 | 201 | 209 | 15.9% | 15.4% | 0.156 |
| | | | same directory | 1,147 | 104 | 1,043 | 143 | 9.1% | 42.1% | 0.149 |
| gson | Java | 57 | high | 342 | 30 | 312 | 194 | 8.8% | 13.4% | 0.106 |
| | | | medium | 384 | 34 | 350 | 190 | 8.9% | 15.2% | 0.112 |
| | | | low | 384 | 34 | 350 | 190 | 8.9% | 15.2% | 0.112 |
| | | | same directory | 851 | 35 | 816 | 189 | 4.1% | 15.6% | 0.065 |
| nest | TypeScript | 12 | high | 0 | 0 | 0 | 19 | — | 0.0% | 0.000 |
| | | | medium | 4 | 0 | 4 | 19 | 0.0% | 0.0% | 0.000 |
| | | | low | 5 | 0 | 5 | 19 | 0.0% | 0.0% | 0.000 |
| | | | same directory | 69 | 0 | 69 | 19 | 0.0% | 0.0% | 0.000 |
| zod | TypeScript | 10 | high | 3 | 1 | 2 | 19 | 33.3% | 5.0% | 0.087 |
| | | | medium | 4 | 1 | 3 | 19 | 25.0% | 5.0% | 0.083 |
| | | | low | 4 | 1 | 3 | 19 | 25.0% | 5.0% | 0.083 |
| | | | same directory | 200 | 6 | 194 | 14 | 3.0% | 30.0% | 0.055 |

The other two aggregations, at the default floor, with the baseline's F1 on the
same seeds:

| Repository | Aggregation | Seeds | Predicted | Correct | Wrong | Missed | Precision | Recall | F1 | Baseline F1 |
|---|---|---|---|---|---|---|---|---|---|---|
| RepoCortex | any file | 14 | 53 | 6 | 47 | 44 | 11.3% | 12.0% | 0.117 | 0.047 |
| | every changed file | 56 | 185 | 18 | 167 | 374 | 9.7% | 4.6% | 0.062 | 0.053 |
| flask | any file | 80 | 208 | 32 | 176 | 225 | 15.4% | 12.5% | 0.138 | 0.150 |
| | every changed file | 301 | 621 | 161 | 460 | 1,976 | 25.9% | 7.5% | 0.117 | 0.264 |
| gson | any file | 65 | 241 | 25 | 216 | 223 | 10.4% | 10.1% | 0.102 | 0.080 |
| | every changed file | 300 | 634 | 55 | 579 | 3,506 | 8.7% | 1.5% | 0.026 | 0.167 |
| nest | any file | 12 | 4 | 0 | 4 | 19 | 0.0% | 0.0% | 0.000 | 0.000 |
| | every changed file | 16 | 4 | 0 | 4 | 29 | 0.0% | 0.0% | 0.000 | 0.000 |
| zod | any file | 11 | 4 | 1 | 3 | 21 | 25.0% | 4.5% | 0.077 | 0.048 |
| | every changed file | 15 | 5 | 1 | 4 | 30 | 20.0% | 3.2% | 0.056 | 0.065 |

Direct callers only, against every row (direct, indirect and dispatch), for the
non-test seed at the default floor:

| Repository | Rows | Predicted | Correct | Precision | Recall | F1 |
|---|---|---|---|---|---|---|
| RepoCortex | direct | 14 | 4 | 28.6% | 8.0% | 0.125 |
| | all | 77 | 8 | 10.4% | 16.0% | 0.126 |
| flask | direct | 165 | 30 | 18.2% | 12.1% | 0.146 |
| | all | 239 | 38 | 15.9% | 15.4% | 0.156 |
| gson | direct | 216 | 32 | 14.8% | 14.3% | 0.145 |
| | all | 384 | 34 | 8.9% | 15.2% | 0.112 |
| nest | direct | 2 | 0 | 0.0% | 0.0% | 0.000 |
| | all | 4 | 0 | 0.0% | 0.0% | 0.000 |
| zod | direct | 2 | 1 | 50.0% | 5.0% | 0.091 |
| | all | 4 | 1 | 25.0% | 5.0% | 0.083 |

What each run covered:

| | RepoCortex | flask | gson | nest | zod |
|---|---|---|---|---|---|
| Start revision | `6fadd51` | `49943b5` | `a961b0f` | `956af70` | `19c40d2` |
| Clone | full history | `--depth 400` | `--depth 400` | `--depth 400` | `--depth 400` |
| Files in the index at the start revision | 723 | 235 | 315 | 2,756 | 743 |
| Symbols | 10,743 | 2,152 | 5,687 | 19,293 | 6,255 |
| Relationships | 20,640 | 1,650 | 9,255 | 4,904 | 3,055 |
| Budget | 20 min | 20 min | 20 min | 20 min | 40 min |
| Stopped by | history exhausted | budget | budget | budget | budget |
| Commits scanned | 40 | 393 | 209 | 31 | 17 |
| Commits scored | 14 | 80 | 65 | 12 | 11 |
| Skipped: fewer than two files modified or deleted | 19 | 164 | 65 | 4 | 4 |
| Skipped: fewer than two changed source files | 5 | 147 | 78 | 14 | 2 |
| Skipped: more than 50 changed source files | 1 | 0 | 0 | 0 | 0 |
| Skipped: no indexed symbol on a changed line | 0 | 2 | 1 | 1 | 0 |
| Skipped: no parent in the clone | 1 | 0 | 0 | 0 | 0 |
| Commits with a non-test seed | 14 | 74 | 57 | 12 | 10 |
| `analyze_impact` calls | 243 | 3,165 | 2,052 | 81 | 93 |
| Answers that dropped rows at the 50-row cap | 21 | 75 | 90 | 0 | 0 |
| Answers whose walk stopped at a limit | 86 | 465 | 203 | 7 | 8 |
| Failed calls | 0 | 0 | 0 | 0 | 0 |
| Non-test seeds with no file predicted (default floor) | 2 of 14 | 32 of 74 | 31 of 57 | 10 of 12 | 8 of 10 |
| Seconds indexing parents, of the total | 255 of 403 | 405 of 1,201 | 602 of 1,202 | 1,143 of 1,356 | 1,346 of 2,679 |

The machine was loaded throughout, and far more so once the kernel index
recorded in performance.md started. The flask, gson and RepoCortex runs were
almost over by then; the nest run overlapped it and the zod run was entirely
inside it, which is why the two TypeScript runs scored so few commits. A fresh
index of nest took 16 s early in the session and between 31 s and 249 s per
commit during its run, and zod's took between 42 s and 260 s. zod was given
40 minutes for that reason. The commit counts are what this machine could do on
the day, not what the harness needs.

**What the figures say.**

- **F1 is between 0.000 and 0.156.** Where there are enough commits to say
  anything (flask and gson), about one predicted file in six to one in eleven
  was changed by the commit, and about one changed file in seven was predicted.
  Read with the bias above, both are understated, and both are still low.
- **Against the same-directory rule the result depends on the seed.** With a
  non-test seed the tool has the higher F1 on four repositories and ties at zero
  on nest, by a wide margin on RepoCortex and gson and by 0.156 to 0.149 on
  flask. With any file as the seed it is behind on flask (0.138 to 0.150). With
  every changed file as a seed it is ahead only on RepoCortex, and on flask and
  gson the rule finds 5.6 and 17.8 times as many of the changed files (908
  against 161, and 980 against 55). Much of a change set is a neighbourhood,
  and a call graph is not a neighbourhood.
- **Precision is where the tool is ahead.** On every repository with a correct
  prediction, the tool's precision is 1.7 to 8 times the rule's for a non-test
  seed. Its recall is lower on all of them: it names few files, and the rule
  names many.
- **The confidence floor barely moves the result.** `medium` and `low` give the
  same counts on four of the five repositories and differ by one file on nest.
  `high` removes between 1 and 42 predicted files. These indexes carry few
  `low` edges, so the floor has little to remove.
- **Indirect rows cost precision.** Direct callers alone are 1.1 to 2.8 times as
  precise as the full answer on RepoCortex, flask and gson. They find fewer
  files, and on gson the wider answer finds two more correct files for 166 more
  wrong ones.
- **The tool often has nothing to say.** 43% of the non-test seeds on
  flask, 54% on gson, 8 of 10 on zod and 10 of 12 on nest predicted no file at
  all. An empty answer costs recall and nothing else, so these F1 figures are
  set more by how often the tool had no caller to report than by how often a
  reported caller was wrong.
- **TypeScript is the weakest of the four languages**, and the sample is the
  smallest. nest scored no correct file in 12 commits, and neither did the
  baseline: nest keeps each package's tests in a separate `test/` directory,
  so the files next to a seed are rarely the ones its commit touched. zod
  scored one correct file in 10 commits. `repocortex resolve` bound 20.5% of
  nest's references that name something the index defines.
- **N is small.** 10 to 14 commits is an anecdote. 57 to 74 is enough to see
  the order of magnitude and not enough to rank two builds that differ by a
  point.

**Not comparable to the published figure.** code-review-graph publishes F1 0.693,
precision 0.546 and recall 1.000 over 13 commits in 6 repositories. Its README,
read on 2026-11-12, describes the truth as the changed files plus the files with
call or import edges into them, taken from the same graph the predictor walks,
and calls the recall of 1.0 a circular upper bound. That measures whether a
blast radius agrees with the graph it was computed from. The measurement here
asks whether it agrees with what a developer went on to change, which is a
harder question with a noisier answer. Neither number says anything about the
other, and the 0.1 here is not evidence that one tool is five times worse.

**What the run found.** Not fixed; recorded for the next change to the resolver.

| Finding | Measured |
|---|---|
| An overloaded method has no callers in the index | gson's `Gson.fromJson` has 11 overloads. All 792 references to `fromJson`, in 92 files, are recorded as `ambiguous` and bound to none of them. `analyze_impact` answers "0 direct, 0 indirect. Risk low (score 20)" for the library's main entry point. TypeScript overload signatures do the same: nest's `Reflector.getAllAndMerge` is declared three times, and all 12 references to it are `ambiguous` |
| Overloads cannot be told apart by name | All 11 share the fully-qualified name `gson/src/main/java/com/google/gson/Gson.java:fromJson`. Asking about `fromJson` returns "matches 14 symbols; re-ask with a fully-qualified name" followed by the same name five times. 972 of gson's 3,561 methods share a fully-qualified name with another symbol |
| An ambiguous name is reported as an internal failure | The same answer carries `error_code: internal` and "RepoCortex failed internally", for a request the caller can correct |
| The documented default floor is wrong | `docs/reference/mcp-tools.md` and the tool's own schema say `min_confidence` defaults to `low` for `analyze_impact`. It defaults to `medium` |

To reproduce the first row:

```bash
git clone --depth 1 https://github.com/google/gson && cd gson
repocortex init && repocortex index && repocortex resolve
repocortex tool call analyze_impact symbol=fromJson
repocortex tool call analyze_impact symbol=gson/src/main/java/com/google/gson/Gson.java:fromJson
repocortex query "SELECT resolution, COUNT(*) FROM symbol_references WHERE index_run_id = :run AND name = 'fromJson' GROUP BY resolution"
```

### Reproducing the two timings

Both scripts print every run, average nothing, and need a clone they are allowed
to modify.

```bash
# Incremental update: cold index, then three two-file edits, then the same under `watch`.
git clone --depth 1 https://github.com/django/django /tmp/bench/django
scripts/bench/incremental.sh /tmp/bench/repocortex /tmp/bench/django \
    django/utils/text.py django/db/models/query.py

# Cold index, one step at a time, with wall time and peak memory per step.
git clone --depth 1 https://github.com/torvalds/linux /tmp/bench/linux
scripts/bench/cold-index.sh /tmp/bench/repocortex /tmp/bench/linux
```

`cold-index.sh` stops a step that passes `BENCH_MAX_SECONDS` (default 7,200) or
`BENCH_MAX_RSS_MB` (default none) and reports how far it got. Run it on an idle
machine if you can: the figures in performance.md were taken on a shared one and
say so.

## Pitfalls

### Pitfall 1: check your `GOARCH` before you benchmark anything

Every performance figure published before one particular revision of
performance.md was silently emulated. The development machine ran an amd64 Go
toolchain under Rosetta 2 on Apple Silicon for the entire life of the
repository, and Go's benchmark headers print `goarch: amd64` and
`cpu: VirtualApple @ 2.50GHz` — the emulated view, not the silicon.

Measured cost of the emulation on RepoCortex's own source:

| Operation | amd64 under Rosetta | arm64 native | Native is |
|---|---|---|---|
| Full index | 1,801 ms | 1,264 ms | 1.43x faster |
| Re-index, nothing changed | 124 ms | 67 ms | 1.84x faster |
| Re-index, one file changed | 206 ms | 129 ms | 1.60x faster |

Query-side commands gain more, because they are dominated by process start:
startup went 64.4 ms → 20.4 ms (3.2x) on a 135,000-symbol corpus.

```bash
go env GOARCH ; uname -m      # if these disagree, your numbers are a ceiling
```

Treat the published figures as roughly 1.4–3x above what a native build
delivers. Released binaries are unaffected — CI cross-compiles `darwin/arm64`.

### Pitfall 2: re-baseline on the same tree before comparing anything

This is the pitfall that most often produces a wrong conclusion, and
accuracy.md's own "Changes since the table above was measured" section exists
because of it.

A `--depth 1` prometheus clone of 727 Go files scores **86.7%** on an unmodified
tree against the documented **86.8%** for a 1,030-file checkout. That difference
is the corpus, not a regression. Every figure in that section is therefore paired
with a baseline re-measured on the same tree and the same `scip-go` index, rather
than compared against the published number — because **a −2.4pp regression
measured against the wrong baseline reads as −0.1pp of rounding.**

The same applies to the run reproduced at the top of this page: 98.0% here and
97.8% in accuracy.md are two different trees, not a change.

### Pitfall 3: `--full` discards resolved edges

`repocortex index --full` rebuilds from scratch and drops the call graph. Re-run
`repocortex resolve` afterwards, or every relationship query will report that no
references have been resolved yet. A plain incremental `repocortex index` does not
have this effect.

Benchmark scripts that loop over `index --full` and then query without resolving
are measuring an empty graph.

### Pitfall 4: one index database holds one committed run

`registry.CommitRun` publishes a run and deletes the rows of the one it replaces.
So `repocortex diff` across two commits normally needs **two index files** — the
working index and a second index built at the other ref — which is what
`--from-index` and `--to-index` are for:

```bash
repocortex diff --from-index ../release-1.4/.repocortex/index.db --from 8f3c1ab --to 42
```

A ref that matches no indexed run fails and says so; it never falls back to
whichever run happens to be committed, because a diff against the wrong run is
worse than no diff at all.

### Pitfall 5: a filter that removes candidates is not automatically safe

Recorded here because it will recur, and because it was learned by measurement
rather than review. Two resolver changes adopted from a competitor both looked
plainly safe and neither survived measurement:

- "A bare call cannot name a method in Go or Rust" — true of the languages,
  false of the parser, which leaves `Receiver` empty when its query could not
  express the receiver. Every chained Go call was read as bare, and
  `repocortex callers Save` went from two bound callers to zero. **Every unit test
  passed throughout**, because they set `Receiver` directly; only the end-to-end
  CLI path exercised the parser.
- "Production code never binds to a test definition" — cost **2.4pp** of `high`
  precision. Shrinking the candidate set tipped references across the
  disambiguation thresholds, so 426 previously-ambiguous references resolved: 64
  right, **362 wrong**. It converted honest ambiguity into confident wrong
  answers.

The rule the resolver now states: removing candidates also removes the ambiguity
that was protecting the caller. "Fewer candidates" and "fewer wrong edges" are
different claims, and only the second is worth having.

### Pitfall 6: benchmark with a stamped, matching build

Every index run records the build that wrote it, and a query from a different
build warns on stderr before answering. If you are comparing two measurements
and one of them printed this, the comparison is between two sets of resolution
rules:

```
level=WARN msg="This index was written by repocortex dev and you are running 9.9.9-test.
Resolution rules change between builds, so these results may come from different rules
than this binary would produce. Re-index to refresh."
```

Re-index rather than reasoning past it. `-q` silences the warning and does not
fix the cause.

---

## Reporting a number that disagrees

Include: the corpus and how it was obtained (`--depth 1` clone at a named
commit), `go env GOARCH` and `uname -m`, the `scip-*` version, the RepoCortex
build, and the coverage line from `doctor --resolution-precision` — not only the
precision line. A precision figure without its coverage cannot be compared to
anything.

If a claim in these documents cannot be reproduced, the convention this project
follows is to **delete the number rather than restate it with a caveat**, keep
the reasoning it supported, and name the commit so the original evidence is one
`git show` away. Where a document and the code disagree, the code is the answer.
