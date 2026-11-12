# Changelog

All notable changes to the RepoCortex VS Code extension are recorded here.

## Unreleased

Follows the answer format of the `repocortex` tools as it stands after
release 1.0.1. That release refuses the `path` and `line` arguments this
version sends, so the binary must be updated with it.

### Changed

- Every question names one definition by its bare name, its file and the
  line it starts on. The fully qualified name is no longer sent: overloads in
  one file share it.
- A name the tool matches to several definitions is offered as a list to
  choose from, and the question is asked again about the choice. The list is
  never shown as results.
- The results view shows the tool's summary unchanged, including an opening
  `UNKNOWN`, `UNRELIABLE`, `UNVERIFIED` or `INCOMPLETE`, and shows the same
  word beside the row count. The reliability the tool reports comes first in
  the view when it is less than good.
- The verdicts `unreliable`, `incomplete` and `mixed` are shown as warnings,
  as `weak` and `empty` already were.
- Row counts come from the tool's own total. A list the tool returned in
  part reads `showing 50 of 195`, and the CodeLens shows the counted total
  with no `+`.
- The CodeLens marks a count the index cannot stand behind, as in
  `1 caller (unreliable)`, and reads `callers unknown` when the tool says
  nothing can be concluded or lists definitions instead of answering.
- A tool failure that names unexpected arguments says the binary may be
  older than the extension expects.

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
