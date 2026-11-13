# Market Leader Gap Analysis: Deficits, Inaccuracies, and Structural Shortfalls

This document details where RepoCortex remains deficient, inaccurate, or architecturally trailing relative to market leaders in code intelligence for AI coding agents (*Serena, GitNexus, codebase-memory-mcp, codegraph, jCodeMunch, Repowise, and CodeScene*).

It synthesizes findings from:
- The [13 November 2026 Language Verification](verification/2026-11-13/README.md)
- The [Agent and Impact Benchmarks](BENCHMARKING.md)
- The [Competitive Analysis](COMPETITIVE_ANALYSIS.md)
- The [Roadmap Known Issues](ROADMAP.md#known-issues)

---

## Update, 14 November 2026

Measured after this document was written. The gaps stand; four figures moved.

- **Round 2 of the agent benchmark has been run, and found no advantage**
  ([BENCHMARKING.md](BENCHMARKING.md#5-agent-benchmark)). On callers of a method
  that four to seven types define, and on tests one call away, an agent with
  search alone was right on 92 of 92 answers; agents that also had RepoCortex
  were right on 91, with 3.7% more tokens and 13% more time. The tool had bound
  5 of 14, 8 of 11 and 19 of 37 of the true callers, said its answer was
  incomplete, and each agent then did the whole search. Gap 1 is therefore not
  "unproven" on those cases; it is measured, and it is Gap 2 that causes it.
- **Elasticsearch resolve**: CPU time is a third of what it was (2,991 s to
  1,003 s, identical output); about 17 minutes of wall time on a loaded
  machine, where the 35 minutes quoted below was measured on a quiet one. It
  is still single-threaded.
- **Linux kernel**: 25.6 minutes to index and 4.7 to resolve on the build of
  13 November, not about two hours. Kernel-style C is read better (on a
  4,650-file subset, files with syntax errors 2,092 to 1,151); the whole tree
  has not been re-measured.
- **`IDisposable` on VS Code**: 638 of 651 references are bound, and the
  subtypes question, which did not finish in 30 minutes, answers in about a
  second.

**What follows for priorities.** The first item of the roadmap in section 4 is
done and its answer changes the order of the rest. An incomplete answer is
worth nothing to an agent that can search, so the work that matters is the
work that makes an answer complete enough to act on without checking: receiver
types (item 2), and for Go a type-checked graph built in, which also gives
interface implementations (item 5). Scale work on repositories the size of the
Linux kernel, and further grammars, do not move this result.

---

## 1. Executive Summary

While RepoCortex has built an impressive foundation—206 tree-sitter grammars bundled, 91 with definitions extracted, zero-network/zero-CGO execution, honest confidence tiers, and multi-protocol parity (MCP, REST OpenAPI, CLI)—**it does not yet outperform or match market leaders in real-world agent efficacy, deep type resolution, or predictive impact analysis.**

The primary deficits fall into six categories:
1. **The Value Gap (Agent ROI):** In published benchmarks, RepoCortex provided **zero measured token or time advantage** over basic lexical search (`ripgrep`).
2. **The Shallow Graph Gap (Syntactic Heuristics vs. Type Inference):** Without an external compiler index (SCIP), the fallback `low` confidence tier is inaccurate (66% pooled precision; Python at 27%), overloaded functions bind incorrectly, and calls on values, chained builders, and Go interfaces fail to resolve.
3. **The Impact Analysis Deficit:** Real-world git-commit blast radius accuracy is poor (F1 between 0.000 and 0.156), frequently lagging behind a naive "same-directory" baseline.
4. **Scale Bottlenecks on Large Repositories:** Large Java codebases experience massive resolve stalls (Elasticsearch takes 35 minutes), and the candidate cap drops hundreds of thousands of references into ambiguous limbo rather than resolving by proximity.
5. **Context Overhead & Schema Bloat:** 28 exposed tools cost ~31 KB of prompt context, compared to competitors offering 1–6 compact front-door tools.
6. **Active Inconsistencies & Tool Bugs:** Key commands like `rename` fail on defaults (`--floor exact`), and risk scales diverge between planning and impact tools.

---

## 2. Competitive Landscape Overview

| Competitor | Core Strength | Where RepoCortex Trails |
|---|---|---|
| **Serena** | Real LSP integration (gopls, rust-analyzer, tsserver) | 100% compiler-grade precision on types, completions, implementations, and call hierarchies out of the box without requiring manual SCIP generation. |
| **codebase-memory-mcp** | Native C-speed graph engine; 11 typed languages; bundled local embedding model | Sub-millisecond queries on massive repos; hybrid semantic search; built-in type resolution without external indexers. |
| **GitNexus** | Dynamic execution-flow tracing, API contract shape checks, taint analysis | Validated 68.4% SWE-bench solve rate (vs 37% bare agent); execution-flow ("process") discovery. |
| **codegraph** | Rust kernel with 1-tool front door | Claims 88% fewer tool calls and 62% fewer tokens; fast cold indexing of Linux kernel (<12 min vs RepoCortex's ~2h). |
| **jCodeMunch** | 6-tool front door routing 90+ tools; token savings tracking | Minimal schema overhead in agent prompts; explicit token savings accounting. |
| **CodeScene** | Behavioral git mining & temporal coupling | High-precision PR impact prediction, team knowledge distribution, and code health safeguards. |

---

## 3. Deep-Dive Gap Analysis

### Gap 1: Agent Efficiency & Benchmark Proof (The "Value Gap")

Market leaders justify their integration into coding agents by publishing clear token reductions and benchmark improvements.

* **Round 1 Benchmark showed no advantage over ripgrep:**
  On direct-caller lookups for uniquely named functions, agents using RepoCortex used **0.7% more tokens, 4 more tool calls, and 23% more time** than agents using search alone ([BENCHMARKING.md](BENCHMARKING.md#5-agent-benchmark)).
* **Explicit prohibition on marketing token savings:**
  The repository's benchmarking documentation explicitly notes:
  > *"Do not quote a token saving for RepoCortex. There is no measurement that supports one."*
* **Round 2 (hard cases) was run on 14 November and showed no advantage either:**
  Ambiguous method names and tests one call away, in a thousand-file Go repository: search alone 92 of 92, with RepoCortex 91 of 92, 3.7% more tokens. Interface dispatch, deeper transitive impact and large monorepos remain untested. See the update at the top.

---

### Gap 2: Semantic Resolution & Type Inference vs. Syntactic Heuristics

When external SCIP compiler indexes are absent, RepoCortex relies on tree-sitter queries and syntactic heuristics. The 13 November verification demonstrated severe limitations:

* **The `low` confidence tier is inaccurate:**
  Measured across 14 non-tuned projects:
  - **Python:** 8 of 30 correct (**26.7%**)
  - **Kotlin:** 9 of 30 correct (**30.0%**)
  - **C#:** 14 of 30 correct (**46.7%**)
  - **Java:** 18 of 30 correct (**60.0%**)
  - **Pooled:** 184 of 280 correct (**65.7%**)
* **Same-arity overloads bind incorrectly:**
  The resolver matches calls to definitions using argument count rather than argument types. In Java, 12 of 30 sampled `low` edges were wrong overloads; in Kotlin, 21 of 30; in C#, 15 of 30; in C++, `FormatKV` calls with 47 references all bound to the first overload.
* **Lack of type inference for receiver chains ("Calls on Values"):**
  - **C#:** Chained calls starting from `new` or a factory method (e.g. `new LoggerConfiguration().WriteTo.Sink(...)`) are not extracted (costing 4 of 8 recall sites in Serilog).
  - **TypeScript / JavaScript:** Receiver calls on context or request objects (`c.text()`, `ctx.req`, `this.html()`) cannot resolve their type, binding to arbitrary same-named symbols in the codebase (e.g., 17 edges into `HonoRequest.header` came from generic receivers `c`, `ctx`, or `this`).
  - **JavaScript:** Calls through `this` inside object-literal methods remain unbound (14 of 31 recall sites in `koa`).
* **Go interface implementations are missing:**
  The index produces **0** `EXTENDS` or `IMPLEMENTS` edges for Go. Calling `find_implementations` on any Go interface returns `UNKNOWN`.
* **Module-level script calls are dropped:**
  Calls made at the top level of a module or script have no enclosing symbol and generate **no call edges** (measured in `express`: 2,015 of 2,180 resolved references had no caller; in `zod`: 2,952 of 7,540).

---

### Gap 3: Impact Analysis & PR Blast Radius Deficit

Market leaders like *GitNexus* and *CodeScene* excel at predicting what a pull request touches. In RepoCortex, `analyze_impact` was evaluated against real git commits across five repositories:

* **Extremely low F1 scores (0.000 to 0.156):**
  - **RepoCortex (Go):** Precision 11.0%, Recall 16.0%, **F1: 0.130**
  - **flask (Python):** Precision 15.9%, Recall 15.4%, **F1: 0.156**
  - **gson (Java):** Precision 8.9%, Recall 15.2%, **F1: 0.112**
  - **nest (TypeScript):** Precision 0.0%, Recall 0.0%, **F1: 0.000**
  - **zod (TypeScript):** Precision 25.0%, Recall 5.0%, **F1: 0.083**
* **Outperformed by a trivial directory baseline:**
  A naive baseline that simply assumed *"a change impacts other source files in the same directory"* frequently tied or beat RepoCortex's graph-based F1 (e.g., on `flask`, same-directory achieved 42.1% recall with an F1 of 0.149 vs graph F1 of 0.156; on `nest`, both tied at 0.000).

---

### Gap 4: Monorepo Scale & The "Candidate Cap" Bottleneck

* **Elasticsearch resolve stall (35 minutes):**
  Cold resolve on Elasticsearch (`51,910` files, `769,904` symbols) took **2,087.9 seconds (~35 minutes)**, 7× longer than the Linux kernel, largely due to high candidate counts in Java.
* **Massive reference abandonment via candidate caps:**
  To guarantee bounded queries, RepoCortex imposes a hard cap on candidates. When a symbol name is common, it abandons binding entirely:
  - **VS Code:** 421,929 references left unbound. All 651 references to `IDisposable` were initially left ambiguous because 11 definitions exceeded the cap.
  - **Elasticsearch:** 714,763 references left unbound.
  Competitors resolve common names using import tables, package boundaries, and directory proximity.
* **Kernel C & Preprocessor Macro Blindspots:**
  On the Linux kernel (`84,603` files), **35,894 files had syntax errors** because tree-sitter cannot parse C preprocessor macros, function-like definitions, and inline assembly, yielding an effective reference binding rate of only **43.8%**.
* **Cold CLI latency on large repos:**
  In a fresh CLI invocation, `get_task_context` scales poorly: 7.5s on VS Code, 14.0s on Elasticsearch, and 31.3s on Linux.

---

### Gap 5: Agent Ergonomics & Context Overhead

* **Schema overhead in agent context:**
  RepoCortex defines 28 tools (~31 KB wire definition). Although default profiles expose 7 tools, the full tool schemas consume significant context window tokens before the agent performs any work.
  - *Comparison:* `codegraph` exposes 1 primary tool; `jCodeMunch` exposes a 6-tool front door.
* **Refusal of Local Hybrid / Vector Retrieval:**
  RepoCortex adheres to a strict non-goal: *"No model, no embeddings, no network."* While this guarantees privacy, zero CGO, and determinism, it leaves agents unable to perform conceptual search (e.g., *"where is user authentication validated?"*) when they do not know the exact symbol names. Competitors (*GitNexus, codebase-memory-mcp, Repowise, Graphify*) provide local BM25 + embedding hybrid search.

---

### Gap 6: Active Bugs & Inconsistencies

Several active bugs from [ROADMAP.md#known-issues](ROADMAP.md#known-issues) create friction for AI agents:

1. **`rename` rejected on defaults:**
   Definition sites remain marked `high` even after compiler SCIP import. As a result, `repocortex rename` with default `--floor exact` rejects every rename attempt; agents must know to pass `--floor high`.
2. **Divergent risk scales:**
   `plan-change` and `analyze_impact` report risk on different numeric and categorical scales without indicating the conversion.
3. **Incremental indexing diff bug:**
   Incremental indexing does not assign a new run ID. Running `diff` between two commits can end up comparing a run with itself unless `--full` is used.
4. **Field and constructor references omitted:**
   Types used as struct fields or constructor parameters are not recorded in the references table.
5. **Maven dependency false negatives:**
   Maven coordinates are not translated to Java package namespaces; all Maven dependencies report as *"not imported"*.
6. **Portal UI flaws (`ui --next`):**
   Refactor plan caller lines display `:undefined`; the `?sym=` parameter is ignored; "What changed" expects query parameters `dfrom`/`dto` instead of documented `?from=`/`?to=`.

---

## 4. Remediation Roadmap for Market Leadership

To close the gap with market leaders, development should prioritize:

1. **Round 2 Benchmark Execution:**
   Benchmark hard, multi-hop tasks (ambiguous symbols, interface dispatch, test reachability) on public repositories, documenting verifiable token reductions and accuracy improvements against pure search.
2. **Lightweight Type Propagation for Receiver Chains:**
   Enable receiver type resolution for builder patterns (`builder.setX().build()`) and common web context objects (`c.json()`, `ctx.req`).
3. **Argument-Type Aware Overload Resolution:**
   Inspect argument literal types or parameter count + types to disambiguate overloads in Java, C#, and Kotlin.
4. **Proximity-Based Disambiguation over Candidate Drops:**
   Instead of abandoning references when candidates exceed the cap, rank candidates by import distance, package match, and directory proximity.
5. **Go Interface Structural Dispatch:**
   Implement structural interface checking in Go to link struct methods with interface method declarations, enabling `find_implementations`.
6. **Unified Compact Front-Door Tool:**
   Provide a unified front-door tool (e.g. `cortex_query` or `get_task_context`) that lowers initial schema prompt overhead to under 6 KB.
