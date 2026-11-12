# MCP tools reference

RepoCortex runs as an MCP server over stdio (`repocortex --project <repo> serve`). This page lists the tools the server registers, taken from the server's own `tools/list` response. `TestToolReferenceMatchesTheServer` fails when the page and the server disagree about which tools exist, which profile exposes each, or what arguments each takes.

## Contents

- [Profiles](#profiles)
- [Tools](#tools)
- [Parameters](#parameters)
- [Reading an answer](#reading-an-answer)
- [Choosing between definitions](#choosing-between-definitions)
- [Paging](#paging)
- [What an answer will not state as fact](#what-an-answer-will-not-state-as-fact)
- [Names](#names)
- [Prompts and instructions](#prompts-and-instructions)

## Profiles

Every tool's schema is sent to the agent on every request, so the default profile is small. Set the profile in `.repocortex.yml`:

```yaml
mcp:
  profile: standard   # minimal (default), standard or full
```

| Profile | Tools |
|---|---|
| `minimal` (default) | 7: `get_task_context`, `get_project_overview`, `search_code`, `find_symbol`, `get_symbol`, `find_references`, `find_callers` |
| `standard` | 18: the 7 above and `find_tests`, `analyze_impact`, `get_resolution_coverage`, `plan_rename`, `list_api_endpoints`, `list_db_tables`, `get_config_key`, `get_file_outline`, `minimize_diff`, `plan_change`, `list_blindspots` |
| `full` | 25: the 18 above and `analyze_change`, `find_owners`, `find_callees`, `find_implementations`, `trace_call_path`, `list_dead_code`, `list_hotspots` |

The counts are what `repocortex tool list --profile <name>` prints.

## Tools

| Tool | Profile | Parameters | What it does |
|---|---|---|---|
| `get_task_context` | minimal (default) | `hub_degree_limit`, `max_files`, `min_confidence`, `task` (required) | CALL THIS FIRST when you are about to change code, before exploring the repository by hand. |
| `get_project_overview` | minimal (default) | none | Call this FIRST, before any other RepoCortex tool. |
| `search_code` | minimal (default) | `limit`, `query` (required) | Full-text search across indexed source. |
| `find_symbol` | minimal (default) | `kind`, `min_confidence`, `name` (required) | Locate a symbol's definition by name. |
| `get_symbol` | minimal (default) | `line`, `min_confidence`, `name` (required), `path` | Get a symbol's definition with a bounded source excerpt and its immediate relationships. |
| `find_references` | minimal (default) | `limit`, `line`, `min_confidence`, `name` (required), `offset`, `path` | Find where a symbol is used. |
| `find_callers` | minimal (default) | `limit`, `line`, `min_confidence`, `name` (required), `offset`, `path` | Find the functions and methods that call a symbol — the callers, not merely the files mentioning it. |
| `find_tests` | standard | `hub_degree_limit`, `limit`, `line`, `min_confidence`, `offset`, `path`, `symbol` (required) | Find the tests covering a symbol, and which signal found each one — a proven call edge is far stronger evidence than a matching filename, and the response names which you got. |
| `analyze_impact` | standard | `hub_degree_limit`, `limit`, `line`, `min_confidence`, `offset`, `path`, `symbol` (required) | Before changing a symbol, find out what depends on it. |
| `get_resolution_coverage` | standard | none | Ask this when a find_callers, find_references or analyze_impact result came back shorter than you expected, BEFORE concluding anything from it. |
| `plan_rename` | standard | `limit`, `line`, `min_confidence`, `name` (required), `new_name` (required), `offset`, `path` | Plan a symbol rename. |
| `list_api_endpoints` | standard | `method` | List the HTTP endpoints this service exposes, with the handler that serves each one and where it is declared. |
| `list_db_tables` | standard | `table` | Report which code reads and which code writes each database table, with per-table read/write counts. |
| `get_config_key` | standard | `key` (required) | Look up one configuration key by name: where it is declared and how many places read it, joined into a single mismatch check. |
| `get_file_outline` | standard | `path` (required) | Get a token-efficient structural outline of a source file by pruning implementation bodies while preserving all imports, types, interfaces, structs, function signatures, and docstrings. |
| `minimize_diff` | standard | `diff` (required) | Compress unified git diffs into structured, token-efficient representations. |
| `plan_change` | standard | `hub_degree_limit`, `limit`, `line`, `min_confidence`, `offset`, `path`, `symbol` (required) | Generate an ordered change plan for a symbol: the direct callers the index bound, a risk level, the test files found to cover it, and the steps in order. |
| `list_blindspots` | standard | none | Audit the repository for code intelligence blindspots and hidden risks. |
| `analyze_change` | full | `base` (required), `head`, `min_confidence` | Find out what a range of commits puts at risk: the symbols the change touched, everything those can reach, and the tests among them. |
| `find_owners` | full | `limit`, `line`, `path`, `symbol` (required) | Find who has recent context on the code a change would touch: the people who have lately changed the symbols a change to this one can reach, ranked by how much of that code they have worked on. |
| `find_callees` | full | `limit`, `line`, `min_confidence`, `name` (required), `offset`, `path` | List what a symbol calls: the definitions inside this repository that its body invokes, each with the file and line of the call and how the binding was established (exact, high, medium or low). |
| `find_implementations` | full | `limit`, `line`, `min_confidence`, `name` (required), `offset`, `path` | List a type's hierarchy in both directions: what implements or extends it, and what it implements or extends itself. |
| `trace_call_path` | full | `from` (required), `from_line`, `from_path`, `max_depth`, `to` (required), `to_line`, `to_path` | Find how one symbol reaches another: the shortest chains of calls and references from 'from' to 'to', hop by hop, each hop with its file, line and confidence. |
| `list_dead_code` | full | `include_exported`, `limit`, `min_certainty`, `path` | List definitions that nothing in the index calls or references, graded by how far that can be trusted. |
| `list_hotspots` | full | `commits`, `include_tests`, `limit`, `path` | Rank files by how often they change against how much code they hold, from recent git history, and list for each the files that are usually committed with it. |

## Parameters

### `get_task_context`

CALL THIS FIRST when you are about to change code, before exploring the repository by hand. Give it the task in plain English and it returns the minimum set of files, symbols, tests and database tables the change actually touches, found by walking the call graph rather than by keyword matching. Every included item states why it is there, and notable near-misses are listed with the reason they were excluded. Using this instead of grepping the repository is the point of this server; prefer it over search_code and find_symbol when you are starting a change rather than looking up one known symbol.

| Parameter | Type | Required | Description |
|---|---|---|---|
| `hub_degree_limit` | integer | no | Stop the walk from expanding THROUGH any symbol with more than this many edges. Default 0 (off), which is what every result you have seen so far used. Set it (50 is a reasonable start) when a result is enormous because the walk crossed a logger, a util or a base class; pruned symbols are still listed, only their neighbours are not explored, and the response says how many were cut. |
| `max_files` | integer | no | Cap on returned items (default 20). Raise only if the default answer is genuinely too narrow. |
| `min_confidence` | string | no | Lowest confidence to return: exact, high, medium or low. Default low. Recommended: 'high' for safe automated code changes and blast radius analysis to filter out heuristic false positives. |
| `task` | string | yes | Describe the change you are about to make, in plain English, as you would to a colleague. For example: 'Add idempotency to payment callback processing'. Full sentences work better than keywords — the task is parsed for intent, not matched literally. |

### `get_project_overview`

Call this FIRST, before any other RepoCortex tool. Reports what the project is, how much of it is indexed, which languages it uses, and how much of the code RepoCortex could resolve. The resolution figure is the important one: it tells you whether relationship answers for this repo are trustworthy or mostly inferred. Do not guess at coverage — this tool states it.

No parameters.

### `search_code`

Full-text search across indexed source. Prefer this over your own grep: it searches the index and returns the paths of the matching files, best match first, which costs a fraction of the context. Use it when you know roughly what you are looking for but not where it lives, and to confirm any relationship result that came back empty, incomplete or low-confidence. Punctuation is fine: 'errors.New' and 'Utils::parse' search for those words side by side.

| Parameter | Type | Required | Description |
|---|---|---|---|
| `limit` | integer | no | Most files to return. Default 20, maximum 200. |
| `query` | string | yes | Search terms. Matches identifiers and code text; camelCase identifiers are searchable by their parts, so 'callback' finds handleCallback. Punctuation is allowed: 'errors.New', 'Utils::parse' and '$this->handler' search for those words next to each other. |

### `find_symbol`

Locate a symbol's definition by name. Faster and more precise than searching when you already know the identifier. Every definition of the name is listed with its file and line; when there are several, pass one row's `path` and `line` to the other tools to choose it. If this returns nothing, the symbol may still exist under a different name or in an unindexed file — fall back to search_code rather than concluding it is absent.

| Parameter | Type | Required | Description |
|---|---|---|---|
| `kind` | string | no | Optional filter: class, interface, function, method, field, constant, type_alias. |
| `min_confidence` | string | no | Lowest confidence to return: exact, high, medium or low. Default low. Recommended: 'high' for safe automated code changes and blast radius analysis to filter out heuristic false positives. |
| `name` | string | yes | Symbol name to find, e.g. 'TransactionService' or 'updateStatus'. Exact match on the name, not a substring. |

### `get_symbol`

Get a symbol's definition with a bounded source excerpt and its immediate relationships. Call this after find_symbol instead of reading the whole file. The excerpt is bounded and may be truncated; the response says so explicitly when it is, so check that flag rather than assuming you have the full body. Several definitions sharing the name are all shown; pass `path` and `line` for one.

| Parameter | Type | Required | Description |
|---|---|---|---|
| `line` | integer | no | Line the definition starts on (or any line inside it). Pass it with path to choose between several definitions in one file, such as overloads. A previous answer prints it as path:line. |
| `min_confidence` | string | no | Lowest confidence to return: exact, high, medium or low. Default low. Recommended: 'high' for safe automated code changes and blast radius analysis to filter out heuristic false positives. |
| `name` | string | yes | Symbol name, e.g. 'updateStatus' or 'TransactionService'. |
| `path` | string | no | File (or directory) the definition is in, to choose between several symbols that share the name. Matched exactly when it names a file. |

### `find_references`

Find where a symbol is used. Every result carries a confidence tier: 'exact' means a compiler proved the binding, 'high' and 'medium' and 'low' mean RepoCortex inferred it and may be wrong. Pass min_confidence='high' when a wrong result costs more than a missing one. CRITICAL: an EMPTY result is ambiguous — it may mean nothing references this symbol, or that RepoCortex could not resolve the references that exist. Never conclude a symbol is unused from an empty result; confirm with search_code. The first line says UNKNOWN, UNRELIABLE or INCOMPLETE when the answer is; a name shared by several definitions returns them to choose from (`path`, `line`), and `limit`/`offset` page a long list.

| Parameter | Type | Required | Description |
|---|---|---|---|
| `limit` | integer | no | Most rows to return. Default 50, maximum 200. The answer says how many rows exist in total. |
| `line` | integer | no | Line the definition starts on (or any line inside it). Pass it with path to choose between several definitions in one file, such as overloads. A previous answer prints it as path:line. |
| `min_confidence` | string | no | Lowest confidence to return: exact, high, medium or low. Default low. Recommended: 'high' for safe automated code changes and blast radius analysis to filter out heuristic false positives. |
| `name` | string | yes | Symbol name whose relationships you want, e.g. 'updateStatus' or 'TransactionService'. |
| `offset` | integer | no | Rows to skip, to read past the first page. Default 0. When more rows exist the answer names the offset to pass next. |
| `path` | string | no | File (or directory) the definition is in, to choose between several symbols that share the name. Matched exactly when it names a file. |

### `find_callers`

Find the functions and methods that call a symbol — the callers, not merely the files mentioning it. Use before changing a signature or behaviour. Every result carries a confidence tier, from 'exact' (a compiler proved the binding) down through 'high', 'medium' and 'low' (RepoCortex inferred it and may be wrong); pass min_confidence='high' to see only bindings RepoCortex can defend. CRITICAL: an EMPTY result is ambiguous and unresolved calls are reported separately, so absence of callers here is NOT evidence the symbol is dead. Confirm with search_code before deleting anything. The first line says UNKNOWN, UNRELIABLE, UNVERIFIED or INCOMPLETE when the answer is; a name shared by several definitions returns them to choose from (`path`, `line`), and `limit`/`offset` page a long list.

| Parameter | Type | Required | Description |
|---|---|---|---|
| `limit` | integer | no | Most rows to return. Default 50, maximum 200. The answer says how many rows exist in total. |
| `line` | integer | no | Line the definition starts on (or any line inside it). Pass it with path to choose between several definitions in one file, such as overloads. A previous answer prints it as path:line. |
| `min_confidence` | string | no | Lowest confidence to return: exact, high, medium or low. Default low. Recommended: 'high' for safe automated code changes and blast radius analysis to filter out heuristic false positives. |
| `name` | string | yes | Symbol name whose relationships you want, e.g. 'updateStatus' or 'TransactionService'. |
| `offset` | integer | no | Rows to skip, to read past the first page. Default 0. When more rows exist the answer names the offset to pass next. |
| `path` | string | no | File (or directory) the definition is in, to choose between several symbols that share the name. Matched exactly when it names a file. |

### `find_tests`

Find the tests covering a symbol, and which signal found each one — a proven call edge is far stronger evidence than a matching filename, and the response names which you got. Use before changing behaviour, to know what to run. A symbol reported as untested is a positive finding, not silence, but it is bounded by what the index could resolve; when the calls to the symbol could not be bound the answer says whether it is tested is UNKNOWN, and means it. A name shared by several definitions returns them to choose from (`path`, `line`).

| Parameter | Type | Required | Description |
|---|---|---|---|
| `hub_degree_limit` | integer | no | Stop the walk from expanding THROUGH any symbol with more than this many edges. Default 0 (off), which is what every result you have seen so far used. Set it (50 is a reasonable start) when a result is enormous because the walk crossed a logger, a util or a base class; pruned symbols are still listed, only their neighbours are not explored, and the response says how many were cut. |
| `limit` | integer | no | Most rows to return. Default 50, maximum 200. The answer says how many rows exist in total. |
| `line` | integer | no | Line the definition starts on (or any line inside it). Pass it with path to choose between several definitions in one file, such as overloads. A previous answer prints it as path:line. |
| `min_confidence` | string | no | Lowest confidence to return: exact, high, medium or low. Default low. Recommended: 'high' for safe automated code changes and blast radius analysis to filter out heuristic false positives. |
| `offset` | integer | no | Rows to skip, to read past the first page. Default 0. When more rows exist the answer names the offset to pass next. |
| `path` | string | no | File (or directory) the definition is in, to choose between several symbols that share the name. Matched exactly when it names a file. |
| `symbol` | string | yes | The symbol you are about to change, e.g. 'updateStatus' or 'TransactionService'. |

### `analyze_impact`

Before changing a symbol, find out what depends on it. Separates DIRECT callers (a proven call edge) from INDIRECT reach and from dispatch rows the runtime MAY execute but no call site proves, and names what it CANNOT see — dynamic dispatch, unresolved references, traversal limits. The risk level is 'unknown', never 'low', when the callers could not be determined: the references to the name were left unbound, or the language's calls are not extracted. The first line says so. The result is a LOWER BOUND: treat an empty or short result as 'nothing found', never as proof that changing the symbol is safe. A name shared by several definitions returns them to choose from (`path`, `line`).

| Parameter | Type | Required | Description |
|---|---|---|---|
| `hub_degree_limit` | integer | no | Stop the walk from expanding THROUGH any symbol with more than this many edges. Default 0 (off), which is what every result you have seen so far used. Set it (50 is a reasonable start) when a result is enormous because the walk crossed a logger, a util or a base class; pruned symbols are still listed, only their neighbours are not explored, and the response says how many were cut. |
| `limit` | integer | no | Most rows to return. Default 50, maximum 200. The answer says how many rows exist in total. |
| `line` | integer | no | Line the definition starts on (or any line inside it). Pass it with path to choose between several definitions in one file, such as overloads. A previous answer prints it as path:line. |
| `min_confidence` | string | no | Lowest confidence to follow: exact, high, medium or low. Default medium, so name-only (low) relationships are left out and counted; pass 'low' to walk them too, or 'high' when the result will gate an automated change. |
| `offset` | integer | no | Rows to skip, to read past the first page. Default 0. When more rows exist the answer names the offset to pass next. |
| `path` | string | no | File (or directory) the definition is in, to choose between several symbols that share the name. Matched exactly when it names a file. |
| `symbol` | string | yes | The symbol you are about to change, e.g. 'updateStatus' or 'TransactionService'. |

### `get_resolution_coverage`

Ask this when a find_callers, find_references or analyze_impact result came back shorter than you expected, BEFORE concluding anything from it. It reports how many of this index's references were actually bound to a definition, and names the two populations excluded from that rate: the ambiguous ones, where several definitions matched and RepoCortex refused to guess, and the ones that matched nothing here at all, which are mostly library calls. A low figure means every relationship answer from this index is a LOWER BOUND drawn from a minority of the evidence — an empty result then tells you about the index, not about the code — and the response names the command that raises it. Takes no arguments.

No parameters.

### `plan_rename`

Plan a symbol rename. Returns the edit SITES — a file and a line each, with no column — split into 'apply' (confidence 'exact': a compiler proved the binding) and 'review' (RepoCortex inferred the binding and may be wrong). You must locate the identifier on each line yourself; a line may contain it more than once or as part of a longer name. WRITES NOTHING. The plan is never complete: references RepoCortex could not resolve are absent and are not reported as absent, so after applying it, run search_code for the old name to find bindings that were missed entirely.

| Parameter | Type | Required | Description |
|---|---|---|---|
| `limit` | integer | no | Most rows to return. Default 50, maximum 200. The answer says how many rows exist in total. |
| `line` | integer | no | Line the definition starts on (or any line inside it). Pass it with path to choose between several definitions in one file, such as overloads. A previous answer prints it as path:line. |
| `min_confidence` | string | no | Lowest confidence to return: exact, high, medium or low. Default low. Recommended: 'high' for safe automated code changes and blast radius analysis to filter out heuristic false positives. |
| `name` | string | yes | Symbol to rename. |
| `new_name` | string | yes | The replacement identifier. Used only to render the plan; nothing is written. |
| `offset` | integer | no | Rows to skip, to read past the first page. Default 0. When more rows exist the answer names the offset to pass next. |
| `path` | string | no | File (or directory) the definition is in, to choose between several symbols that share the name. Matched exactly when it names a file. |

### `list_api_endpoints`

List the HTTP endpoints this service exposes, with the handler that serves each one and where it is declared. Routes are recognised from the framework's own syntax — a Spring annotation, a Go router registration, a FastAPI or Flask decorator, an Express member call — never from a string that merely looks like a path, which is what keeps a log message mentioning 'GET /users' out of the result. Pass method to filter to one HTTP verb. NOT covered: OpenAPI documents, GraphQL schemas, gRPC service definitions and API-gateway configuration — a service defined entirely in those reports no endpoints here, which is a coverage limit, not a finding about the code. Prefer this over grepping route decorators by hand.

| Parameter | Type | Required | Description |
|---|---|---|---|
| `method` | string | no | Filter to one HTTP method, e.g. GET. Case-insensitive. |

### `list_db_tables`

Report which code reads and which code writes each database table, with per-table read/write counts. Pass table to look up one table by name. READ and WRITE are counted separately: 'what touches this table' is a weaker question than 'what writes it', and the second is the one to check before changing a column. A table no migration here declares is still reported — that is the normal shape of a service reading a database another service owns. Some statements are assembled at run time; the response says how many, because whether that exact statement executes as written cannot be established by a static read. Prefer this over grepping for SQL string literals or ORM calls: those are exactly the queries this scan may also miss, and this tool says so instead of reporting a false all-clear.

| Parameter | Type | Required | Description |
|---|---|---|---|
| `table` | string | no | Filter to one table name. Case-insensitive. |

### `get_config_key`

Look up one configuration key by name: where it is declared and how many places read it, joined into a single mismatch check. The two findings worth acting on are a key READ but nowhere DECLARED — the way a service fails at start-up — and a key declared and never read, which is either dead configuration or a lookup this scan cannot see. VALUES ARE NEVER READ: no field in the response can hold one, so a wrong redaction rule cannot leak one either; a key whose name looks like a credential is labelled sensitive by name alone. Declarations are read from .env, YAML, JSON, TOML and .properties files — Kubernetes, Helm, Terraform and CI definitions are not, so a key set only there reports as undeclared even though it works in production, which is a coverage limit, not a finding. Prefer this over grepping for the key string across the repository.

| Parameter | Type | Required | Description |
|---|---|---|---|
| `key` | string | yes | The configuration key to look up, e.g. DATABASE_URL. |

### `get_file_outline`

Get a token-efficient structural outline of a source file by pruning implementation bodies while preserving all imports, types, interfaces, structs, function signatures, and docstrings. Use this instead of reading an entire large file (reducing 10,000 tokens to 300 tokens) when you only need to inspect declarations, API contracts, or symbol signatures.

| Parameter | Type | Required | Description |
|---|---|---|---|
| `path` | string | yes | File path to outline (relative to repo root or absolute). Prunes implementation bodies to reduce token size while preserving all types, structs, interfaces, signatures, and docstrings. |

### `minimize_diff`

Compress unified git diffs into structured, token-efficient representations. Strips redundant index hashes and file timestamps, extracts enclosing symbol headers, and collapses unchanged context runs to minimize context token usage. Use this when analyzing pull requests, patches, or git diffs to preserve context window space.

| Parameter | Type | Required | Description |
|---|---|---|---|
| `diff` | string | yes | Unified diff text (e.g. git diff output) to compress into a structured, token-efficient representation. |

### `plan_change`

Generate an ordered change plan for a symbol: the direct callers the index bound, a risk level, the test files found to cover it, and the steps in order. Use it before modifying or refactoring existing code. The callers are the ones RepoCortex could bind, not a guarantee of all of them: when they could not be determined the risk is 'unknown' and the first line says the plan is incomplete, so search for the name before relying on it.

| Parameter | Type | Required | Description |
|---|---|---|---|
| `hub_degree_limit` | integer | no | Stop the walk from expanding THROUGH any symbol with more than this many edges. Default 0 (off), which is what every result you have seen so far used. Set it (50 is a reasonable start) when a result is enormous because the walk crossed a logger, a util or a base class; pruned symbols are still listed, only their neighbours are not explored, and the response says how many were cut. |
| `limit` | integer | no | Most rows to return. Default 50, maximum 200. The answer says how many rows exist in total. |
| `line` | integer | no | Line the definition starts on (or any line inside it). Pass it with path to choose between several definitions in one file, such as overloads. A previous answer prints it as path:line. |
| `min_confidence` | string | no | Lowest confidence to return: exact, high, medium or low. Default low. Recommended: 'high' for safe automated code changes and blast radius analysis to filter out heuristic false positives. |
| `offset` | integer | no | Rows to skip, to read past the first page. Default 0. When more rows exist the answer names the offset to pass next. |
| `path` | string | no | File (or directory) the definition is in, to choose between several symbols that share the name. Matched exactly when it names a file. |
| `symbol` | string | yes | The symbol you are about to change, e.g. 'updateStatus' or 'TransactionService'. |

### `list_blindspots`

Audit the repository for code intelligence blindspots and hidden risks. Detects unresolved external and internal calls, untested critical functions with high fan-in, unmapped HTTP endpoints, orphan database tables, missing config declarations, and unparsed files. Use this to audit codebase health and find coverage gaps.

No parameters.

### `analyze_change`

Find out what a range of commits puts at risk: the symbols the change touched, everything those can reach, and the tests among them. Use it to review a branch or a pull request before merging, or to check your own work before you push — pass the base branch and, optionally, the head. Changed symbols are listed first, then the tests that reach them (run those first), then the other code within reach, each with its confidence tier. It reads the index as built: if the index is older than the head of the range, symbols added since are missing and the response says so. To ask about ONE symbol, use analyze_impact. Needs a git repository.

| Parameter | Type | Required | Description |
|---|---|---|---|
| `base` | string | yes | The revision the change starts from: a branch, tag or commit, e.g. 'main' or 'origin/main'. |
| `head` | string | no | The revision the change ends at. Default 'HEAD'. |
| `min_confidence` | string | no | Lowest confidence to follow: exact, high, medium or low. Default medium. Use 'high' when the result will gate a merge. |

### `find_owners`

Find who has recent context on the code a change would touch: the people who have lately changed the symbols a change to this one can reach, ranked by how much of that code they have worked on. Use it to suggest a reviewer for a pull request, or to find who to ask before changing unfamiliar code. It reports who CHANGED the code recently, read from git history, and not who owns it or must approve it: someone in CODEOWNERS may be absent. Names only; email addresses are never returned. Needs a git repository.

| Parameter | Type | Required | Description |
|---|---|---|---|
| `limit` | integer | no | Most people to list. Default 5. |
| `line` | integer | no | Line the definition starts on (or any line inside it). Pass it with path to choose between several definitions in one file, such as overloads. A previous answer prints it as path:line. |
| `path` | string | no | File (or directory) the definition is in, to choose between several symbols that share the name. Matched exactly when it names a file. |
| `symbol` | string | yes | The symbol you are changing, e.g. 'updateStatus' or 'TransactionService'. |

### `find_callees`

List what a symbol calls: the definitions inside this repository that its body invokes, each with the file and line of the call and how the binding was established (exact, high, medium or low). Use it to read a function's dependencies without opening the file, or to walk down from an entry point. It lists only calls the index could bind to a definition HERE: calls into libraries and the standard library never appear, so an empty or short result means 'nothing bound inside this repository', not 'calls nothing'. For the opposite direction use find_callers.

| Parameter | Type | Required | Description |
|---|---|---|---|
| `limit` | integer | no | Most rows to return. Default 50, maximum 200. The answer says how many rows exist in total. |
| `line` | integer | no | Line the definition starts on (or any line inside it). Pass it with path to choose between several definitions in one file, such as overloads. A previous answer prints it as path:line. |
| `min_confidence` | string | no | Lowest confidence to return: exact, high, medium or low. Default low. Recommended: 'high' for safe automated code changes and blast radius analysis to filter out heuristic false positives. |
| `name` | string | yes | Symbol name whose relationships you want, e.g. 'updateStatus' or 'TransactionService'. |
| `offset` | integer | no | Rows to skip, to read past the first page. Default 0. When more rows exist the answer names the offset to pass next. |
| `path` | string | no | File (or directory) the definition is in, to choose between several symbols that share the name. Matched exactly when it names a file. |

### `find_implementations`

List a type's hierarchy in both directions: what implements or extends it, and what it implements or extends itself. Use it before changing an interface, an abstract method or a base class, when the callers of the declaration are not the whole story because the work happens in the types built on it. Each row carries its confidence tier; exact means a compiler index proved the relationship. Only DECLARED relationships are visible from syntax, so a Go type that satisfies an interface without naming it is absent until a compiler index is imported — an empty result means none declared, not none exist.

| Parameter | Type | Required | Description |
|---|---|---|---|
| `limit` | integer | no | Most rows to return. Default 50, maximum 200. The answer says how many rows exist in total. |
| `line` | integer | no | Line the definition starts on (or any line inside it). Pass it with path to choose between several definitions in one file, such as overloads. A previous answer prints it as path:line. |
| `min_confidence` | string | no | Lowest confidence to return: exact, high, medium or low. Default low. Recommended: 'high' for safe automated code changes and blast radius analysis to filter out heuristic false positives. |
| `name` | string | yes | Symbol name whose relationships you want, e.g. 'updateStatus' or 'TransactionService'. |
| `offset` | integer | no | Rows to skip, to read past the first page. Default 0. When more rows exist the answer names the offset to pass next. |
| `path` | string | no | File (or directory) the definition is in, to choose between several symbols that share the name. Matched exactly when it names a file. |

### `trace_call_path`

Find how one symbol reaches another: the shortest chains of calls and references from 'from' to 'to', hop by hop, each hop with its file, line and confidence. Use it to answer 'how does this request end up writing that table', to confirm a suspected dependency before a refactor, or to explain a blast radius entry you did not expect. Only the SHORTEST routes are returned, at most three. No route found is not proof of independence: one unresolved call or interface dispatch along the way breaks the chain, and the response says so.

| Parameter | Type | Required | Description |
|---|---|---|---|
| `from` | string | yes | Symbol the path starts at, e.g. 'handleCallback'. |
| `from_line` | integer | no | Line the 'from' definition starts on, when several in one file share the name. |
| `from_path` | string | no | File (or directory) the 'from' definition is in, when several symbols share the name. |
| `max_depth` | integer | no | Longest route to look for, in hops. Default 6. Raise it only if the answer was 'no route found' and you expect a long one. |
| `to` | string | yes | Symbol the path should reach, e.g. 'saveTransaction'. |
| `to_line` | integer | no | Line the 'to' definition starts on, when several in one file share the name. |
| `to_path` | string | no | File (or directory) the 'to' definition is in, when several symbols share the name. |

### `list_dead_code`

List definitions that nothing in the index calls or references, graded by how far that can be trusted. 'likely' means the definition is known to be unexported, its file parsed cleanly, no unbound reference anywhere is spelled like it and its name appears nowhere else in the source text; 'possible' means one of those could not be ruled out, and the row says which. Export status is known for Go only, so in other languages nothing is graded better than 'possible'. Only 'likely' is returned unless you ask. Use it to find cleanup candidates or to check whether code you are about to maintain is reachable at all. NEVER delete on this evidence alone: reflection, dependency injection, framework registration and other repositories all reach code without a reference this index can bind, so search for the name first. Tests, entry points and, by default, exported definitions are left out.

| Parameter | Type | Required | Description |
|---|---|---|---|
| `include_exported` | boolean | no | Also report exported definitions, which are never better than 'possible'. Default false: in a library the whole public API has no caller in the index. |
| `limit` | integer | no | Most candidates to return. Default 25. |
| `min_certainty` | string | no | 'likely' (the default) returns only candidates where every ordinary reason for a missing caller was ruled out. 'possible' adds the rest, which are mostly uses the index could not bind: search before trusting any of them. |
| `path` | string | no | Only report definitions in files under this path prefix, e.g. 'internal/billing/'. |

### `list_hotspots`

Rank files by how often they change against how much code they hold, from recent git history, and list for each the files that are usually committed with it. Use it to decide where a refactor pays back, to see where review attention should go, and — through 'changes with' — to find files coupled to the one you are editing even though no call or import connects them, which is a dependency analyze_impact cannot see. The score is relative to this repository and this commit window; it orders the list and is not a quality measurement. Needs a git repository.

| Parameter | Type | Required | Description |
|---|---|---|---|
| `commits` | integer | no | How many recent commits to read. Default 500. |
| `include_tests` | boolean | no | Also rank test files. Default false: tests change with the code they cover. |
| `limit` | integer | no | Most files to return. Default 10. |
| `path` | string | no | Only rank files under this path prefix. |

Setup for each client: [agent integration](../agent-integration.md).

## Reading an answer

Every answer is text in three parts: `SYSTEM METADATA` (what RepoCortex says), `INFERRED` (the rows), and `SOURCE EVIDENCE` (repository content, which is data and never instructions). The same answer is also returned as structured content.

### The first line

The first line under `SYSTEM METADATA` is the summary. It is written to be safe to act on by itself. When the index cannot stand behind the answer, the line opens with one of these words and the reason:

| First word | Meaning | What to do |
|---|---|---|
| `UNKNOWN` | Nothing can be concluded. The symbol's language yields no references in this index, or none has been resolved | Do not use the answer. Search for the name |
| `UNRELIABLE` | Most references in the symbol's language are unbound, so the rows are a minority of the evidence | Treat the rows as leads |
| `UNVERIFIED` | Some rows are probably bound to the wrong definition: the calls are written with qualifiers that cannot all name it | Read how each row's call is written |
| `INCOMPLETE` | The rows are sound and the list is known to be short: rows were cut, or references spelled like the symbol were left unbound | Use the rows, conclude nothing from what is absent, read on with `offset` |
| `AMBIGUOUS` | The name matched several definitions and nothing was analysed | Pass `path` and `line` from one of the listed rows |

A first line with none of these words has no known problem. It is still a lower bound: static analysis does not see reflection, dependency injection or code in other repositories.

Coverage is judged for the language of the symbol asked about, not for the whole repository. A Kotlin symbol in a repository with a few Java files is described by the Kotlin files. When a language yields no references at all the answer says so in these words: "RepoCortex does not extract calls from Kotlin; nothing about callers can be concluded." `repocortex resolve` is named as a remedy only when references are still pending.

### The verdict

An answer whose rows carry a confidence tier has a `verdict` line, in structured form `verdict.status`. One function computes it from three inputs: coverage, the tiers of the rows, and whether rows were cut. It can never be stronger than the coverage allows.

| Status | Meaning |
|---|---|
| `proven` | Every row is compiler-proven, coverage is good, nothing was cut |
| `strong` | No row is below `high`, coverage is good, nothing was cut |
| `incomplete` | Rows are `high` or better, and the list is known to be short |
| `mixed` | Rows differ in strength, or some are probably bound to the wrong definition |
| `weak` | Every row is inferred, most of them weakly |
| `unreliable` | Coverage is low or unknown. The rows are not an answer, whatever their tiers |
| `empty` | Nothing was returned, which is not evidence of nothing |

`proven` and `strong` are the only two that say the rows may be acted on. Neither is reachable when the first line carries one of the words above.

### Other fields

| Field | Text line | Meaning |
|---|---|---|
| `reliability.level` | first word of the summary | `good`, `incomplete`, `unverified`, `low` or `none`. Present on relationship answers |
| `page` | `truncated:` | `offset`, `limit`, `total`, `returned`, `next_offset` for a paged list |
| `outcome` | `outcome:` | `ambiguous_name` when the answer is a list of definitions to choose from. Absent otherwise |
| `truncated`, `truncated_reason` | `truncated:` | Rows or traversal were cut, and why |
| `omitted` | `omitted:` | What the cut rows were, by kind and confidence |
| `notes` | `note:` | Caveats, most important first |

### Risk

`analyze_impact`, `plan_change` and `analyze_change` report a risk level: `low`, `moderate`, `high` or `unknown`.

`unknown` is reported in place of `low` when the callers could not be determined: references spelled like the symbol exist and were not bound to it, or its language yields no references. The answer then prints the counts and no score. A score computed without the callers is not a low score.

`moderate` and `high` stay as scored when callers are missing, because a floor survives missing evidence. The summary then reads "Risk moderate or higher".

`find_tests` follows the same rule. When no test is found and the calls to the symbol could not be bound, it says whether the symbol is tested is unknown. It does not say the symbol is untested.

## Choosing between definitions

A tool that takes `name` or `symbol` answers for one definition. When several share the name it does not pick one. It returns `outcome: ambiguous_name` and one row per definition, with the kind, the owning type, the parameter count, the lines and the first line of the declaration. Each row ends with the arguments that select it:

```text
- candidate fromJson  gson/src/main/java/com/google/gson/Gson.java:850  (method, in class Gson, 2 parameter(s), lines 850-852; select with path=gson/src/main/java/com/google/gson/Gson.java line=850)
```

This is an ordinary answer, not an error. `repocortex tool call` exits 0.

| Argument | On | Selects |
|---|---|---|
| `path` | every tool that takes `name` or `symbol` | The file. Matched as the exact file first, then as whole path segments (a directory or a file name), then as a substring |
| `line` | the same tools | The definition that starts on that line, or the innermost one that contains it |
| `from_path`, `from_line`, `to_path`, `to_line` | `trace_call_path` | The same, for each end |

`path` and `line` together select any definition. A qualified name does not: it is the file and the identifier, so overloads in one file share it.

A tool that is about one kind of thing prefers that kind when several definitions remain. `find_implementations` prefers a type over a method of the same name. `find_callers`, `find_callees` and `trace_call_path` prefer a function or method. The answer says when it chose this way. `get_symbol` shows every matching definition and `find_symbol` lists them all.

## Paging

`find_references`, `find_callers`, `find_callees`, `find_implementations`, `plan_rename`, `analyze_impact`, `find_tests` and `plan_change` take `limit` and `offset`.

| Argument | Default | Maximum |
|---|---|---|
| `limit` | 50 | 200 |
| `offset` | 0 | none |

An answer that does not carry every row says how many exist and what to pass next:

```text
truncated: showing rows 1-50 of 195 caller(s), so 145 are not shown here. Pass offset=50 to read on (limit accepts up to 200)
```

An answer is also limited to 24 KB, so a page can hold fewer rows than `limit`. `page.returned` and `page.next_offset` say how many it holds. `search_code` and `list_dead_code` take `limit` with their own defaults.

## What an answer will not state as fact

**Whether a definition is exported.** The index sets its exported flag from the first letter of the name. That is the language's own rule in the languages below and a guess everywhere else. Outside them an answer says the export status is unknown, `analyze_impact` adds no points for it either way, and `list_dead_code` grades the candidate `possible`.

| Language | Why the flag is trusted |
|---|---|
| `go` | An upper-case first letter is the language's own export rule |

The table is `exportedFlagTrusted` in `internal/capability/exported.go`. `TestExportedTableIsDocumented` keeps this copy in step with it.

**How often a low-confidence relationship is right.** It varies by language and by project. `analyze_impact` leaves `low` relationships out by default and counts what it left out, but quotes no precision figure. [Accuracy](../accuracy.md) says how precision is measured, and `repocortex accuracy` measures it on your repository.

**That a row is a call to the definition.** A call is bound by the name it calls. Each caller row says how the call is written when it carries a qualifier (`written as errors.New`), so a row qualified by another package or object can be discarded.

## Names

Every tool is `<verb>_<noun>`, with one name and no aliases; see
[naming conventions](naming-conventions.md).

## Prompts and instructions

The server sends under 900 bytes of session instructions and offers three
prompts: `plan_refactor` (argument `symbol`), `analyze_change` (argument
`symbols`) and `get_oriented`. See [protocols](../guides/protocols.md#mcp).
