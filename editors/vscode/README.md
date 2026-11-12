# RepoCortex for VS Code

Shows what the local RepoCortex index knows about the symbol under your
cursor: who calls it, what a change to it could break, which tests reach it,
and what implements it. Every row carries the confidence tier RepoCortex
reports, and the view always shows what RepoCortex said it could not see.

The extension is a front end to the `repocortex` binary. It starts no server
and has no index of its own.

## Requirements

- The `repocortex` binary, on your `PATH` or at the path set in
  `repocortex.path`. See the [installation guide](https://github.com/satyendrakumarsingh/repocortex/blob/main/docs/getting-started/installation.md).
- An index for the workspace, in `.repocortex/`. Build one with the command
  **RepoCortex: Index this workspace in a terminal**, or yourself:

  ```bash
  repocortex init
  repocortex index
  repocortex resolve
  ```

- A trusted workspace. The extension runs a program against the folder, so
  it stays off in Restricted Mode.

## Commands

Put the cursor on a function, method or type, then run a command from the
Command Palette or the editor's context menu.

| Command | Asks RepoCortex | Tool |
|---|---|---|
| **RepoCortex: Show callers** | Which functions call this | `find_callers` |
| **RepoCortex: Show what this could break** | Direct and indirect dependents, with a risk score | `analyze_impact` |
| **RepoCortex: Show tests that reach this** | Tests that reach the symbol through the call graph | `find_tests` |
| **RepoCortex: Show implementations** | What implements or extends a type, and what it implements or extends | `find_implementations` |
| **RepoCortex: Index this workspace in a terminal** | Runs `repocortex index`, then `repocortex resolve` | |
| **RepoCortex: Show the full tool report** | Opens the exact text the tool returned | |
| **RepoCortex: Refresh index status** / **Show index status** | Re-reads the index state | |

Answers appear in the **RepoCortex** panel. Click a row to open the file at
the line the tool reported.

## Reading the results

The first group in the view is **What RepoCortex says about this answer**. It
holds, in the tool's own words:

- whether the index was built from the commit that is checked out;
- whether the answer was truncated, and why;
- how many results were left out to fit the response;
- every note the tool attached, such as how many references in the
  repository are unresolved;
- the tool's verdict on its own answer and its suggested next step.

Nothing in that group is filtered or shortened by the extension. Click any
line to read the full report.

Each result row shows its confidence tier:

| Tier | Meaning |
|---|---|
| `exact` | A compiler proved the binding when the index was built |
| `high` | Inferred from strong evidence. It may still be wrong |
| `medium` | Inferred from partial evidence. Treat it as a lead |
| `low` | A weak inference. Confirm it before acting on it |

An empty result is not proof. RepoCortex reports a lower bound: a caller it
could not resolve does not appear. The view says so each time.

### When a name is shared

Several definitions can share one name. The extension asks the language
server where the symbol under the cursor is defined, then asks RepoCortex
about that definition. If it cannot tell which one you mean, it asks you.

If the tool answers for a different definition than the one you chose, the
view says so in a line marked **Check**. Lines marked **Check** are written by
the extension, not by RepoCortex.

## CodeLens

Set `repocortex.codeLens.enabled` to show the caller count above function
and type definitions. It is off by default.

The lens does not run a process as you type:

- lens positions come from the editor's outline, computed once per document
  version and capped at 60 per file;
- a count is looked up only for a lens that is scrolled into view;
- a count is cached until the index is rebuilt;
- a lookup waits until the document has been quiet for about a second;
- at most two lookups run at a time.

A lens needs a language extension that provides an outline for the file. A
count of zero reads "0 callers found", never "unused".

## Status bar

The status bar item shows whether the workspace has an index and how old it
is, and warns when the index was built from a different commit than `HEAD`.
That comparison covers commits only: uncommitted changes are never in the
index. Click the item to index the workspace or refresh the status.

## Settings

| Setting | Default | Meaning |
|---|---|---|
| `repocortex.path` | `repocortex` | Path to the binary, or a command name on `PATH`. A user setting only: a workspace cannot change which program runs |
| `repocortex.codeLens.enabled` | `false` | Show the caller count above definitions |
| `repocortex.minConfidence` | `low` | Lowest tier shown: `low`, `medium`, `high` or `exact`. Sent to the tool as `min_confidence`, so the tool reports how many results it left out |

## Privacy

Everything stays on your machine.

- The extension runs the local `repocortex` binary and reads what it prints.
  It opens no network connection and has no telemetry.
- The binary is started directly with an argument list, never through a
  shell. Text from the editor reaches it only as a JSON value.
- Every run has a timeout and stops when you cancel it.

## Troubleshooting

| Message | Cause | Fix |
|---|---|---|
| The repocortex binary was not found | The binary is not on the `PATH` VS Code sees | Set `repocortex.path` to the full path, for example `/usr/local/bin/repocortex` |
| This workspace has no index to answer from | No `.repocortex/` index | Run **RepoCortex: Index this workspace in a terminal** |
| The tool reported a failure | The tool rejected the question | Open the report from the message to read the tool's own text |
| No tool named ... | The installed binary is older than this extension | Update `repocortex` |
| No CodeLens appears | The file has no outline, or the index is absent | Install the language's extension; check the status bar |

## Build from source

Needs Node.js 22 or later.

```bash
cd editors/vscode
npm install
npm run compile
npm test
npx @vscode/vsce package --no-dependencies
code --install-extension repocortex-vscode-0.1.0.vsix
```

To run the tests against a real binary and index as well:

```bash
REPOCORTEX_BIN=/path/to/repocortex REPOCORTEX_PROJECT=/path/to/indexed/repo npm test
```

## Licence

Apache-2.0, the same as RepoCortex. See [LICENSE](LICENSE) and [NOTICE](NOTICE).
