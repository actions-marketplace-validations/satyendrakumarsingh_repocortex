import * as fs from 'node:fs';
import * as path from 'node:path';

import type { ToolAnswer } from '../src/core/result';
import { parseToolEnvelope } from '../src/core/result';

// The files in test/fixtures are real output, captured from a `repocortex`
// built at commit 6fadd51 and run against an index of this repository
// (init, index, resolve). Only two things were changed afterwards: absolute
// index paths became /work/project/.repocortex/index.db, and the project
// name became "repocortex".
//
//   <tool>.json            repocortex --json tool call <tool> ...   (exit 0)
//   find_callers.json            name=parseToolArg
//   find_callers_truncated.json  name=emitJSON                (item cap reached)
//   find_callers_floor.json      name=emitJSON min_confidence=exact
//   find_callers_missing.json    name=NoSuchSymbolAnywhere
//   find_callers_wrong_target.json  name=New path=internal/web/web.go
//                                (answers for ownership.go: the path hint lost)
//   find_callers_stale.json      fixtures/payment-platform, indexed one commit behind HEAD
//   find_symbol_many.json        name=New                     (15 definitions)
//   analyze_impact_omitted.json  symbol=emitJSON              (51 results left out)
//   analyze_impact_ambiguous.json  symbol=New                 (exit 1, is_error)
//   find_implementations.json    name=Symbol;  _both: name=Impacted
//   error_missing_argument.json  find_callers with no name    (exit 1, result null)
//   error_bad_confidence.json    min_confidence=bogus         (exit 1)
//   not_indexed.json             after `init` only            (exit 0, status absent)
//   status_*.json                repocortex --json status
//   stderr_no_index.txt          tool call in a folder with no .repocortex (exit 1)
//   stderr_unknown_tool.txt      tool call nope               (exit 1)

/** The extension folder. Compiled tests live in out/test, two levels below it. */
export const EXTENSION_DIR = path.resolve(__dirname, '..', '..');

export const FIXTURE_DIR = path.join(EXTENSION_DIR, 'test', 'fixtures');

/** A captured sample, exactly as `repocortex` printed it. */
export function fixture(name: string): string {
  return fs.readFileSync(path.join(FIXTURE_DIR, name), 'utf8');
}

export function answer(name: string): ToolAnswer {
  return parseToolEnvelope(fixture(name));
}
