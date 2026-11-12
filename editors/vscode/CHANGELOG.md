# Changelog

All notable changes to the RepoCortex VS Code extension are recorded here.

## 0.1.0

First version.

### Added

- Commands for the symbol under the cursor: **Show callers**, **Show what
  this could break**, **Show tests that reach this** and **Show
  implementations**. They call `find_callers`, `analyze_impact`, `find_tests`
  and `find_implementations` through `repocortex tool call`.
- A results view in the panel. Each row shows its confidence tier and opens
  the file at the reported line. The tool's own notes, truncation reason,
  omitted counts, verdict and index caveat are shown first and never hidden.
- A check that the tool answered for the definition that was asked about,
  shown as a **Check** line when it did not.
- A CodeLens with the caller count above function and type definitions. Off
  by default, resolved only for visible lenses, cached until the index is
  rebuilt, and held back while you type.
- A status bar item showing whether the workspace has an index, how old it
  is, and whether it was built from the checked-out commit.
- **Index this workspace in a terminal**, which runs `repocortex index` and
  `repocortex resolve` as visible tasks, with `repocortex init` first when
  there is no index.
- Settings: `repocortex.path`, `repocortex.codeLens.enabled` and
  `repocortex.minConfidence`.
- Messages for a missing binary, a missing index and a tool failure, each
  with one action that fixes it.
