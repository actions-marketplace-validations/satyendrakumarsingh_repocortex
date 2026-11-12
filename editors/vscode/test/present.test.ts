import assert from 'node:assert/strict';
import { test } from 'node:test';

import type { LimitRow, ResultRow } from '../src/core/present';
import { buildResultModel, fullReport, limitLine } from '../src/core/present';
import { answer } from './helpers';

function results(model: ReturnType<typeof buildResultModel>): ResultRow[] {
  return model.sections.flatMap((s) => s.rows).filter((r): r is ResultRow => r.type === 'result');
}

function limits(model: ReturnType<typeof buildResultModel>): LimitRow[] {
  return model.sections.flatMap((s) => s.rows).filter((r): r is LimitRow => r.type === 'limit');
}

test('callers: every row states its tier and location', () => {
  const a = answer('find_callers.json');
  const model = buildResultModel(a, { kind: 'callers', symbol: 'parseToolArg', askedAbout: 'function  internal/cli/tool.go:384' });
  assert.equal(model.title, 'Callers of parseToolArg (function  internal/cli/tool.go:384)');
  assert.equal(model.summary, '3 caller(s) of parseToolArg (internal/cli/tool.go:384).');
  assert.equal(model.resultCount, 3);
  const rows = results(model);
  assert.deepEqual(
    rows.map((r) => [r.label, r.description, r.confidence, r.path, r.line]),
    [
      ['TestParseToolArgTypes', 'high · internal/cli/structure_test.go:254', 'high', 'internal/cli/structure_test.go', 254],
      ['TestParseToolArgTypes', 'high · internal/cli/structure_test.go:264', 'high', 'internal/cli/structure_test.go', 264],
      ['newToolCallCmd', 'high · internal/cli/tool.go:323', 'high', 'internal/cli/tool.go', 323],
    ],
  );
  assert.match(rows[2]?.tooltip ?? '', /Why: defined in the same file/u);
  assert.match(rows[2]?.tooltip ?? '', /^high: inferred by RepoCortex/u);
  assert.equal(model.sections[1]?.title, 'Call sites (3)');
});

test('the limits section comes first and carries every statement the tool made', () => {
  const a = answer('analyze_impact.json');
  const model = buildResultModel(a, { kind: 'impact', symbol: 'parseToolArg' });
  const first = model.sections[0];
  assert.ok(first);
  assert.match(first.title, /^What RepoCortex says about this answer \(\d+\)$/u);
  assert.equal(first.warning, true);
  const texts = limits(model).map((l) => l.text);
  assert.ok(texts.includes(a.truncatedReason as string));
  for (const note of a.notes) {
    assert.ok(texts.includes(note), note);
  }
  assert.ok(texts.includes(a.provenance?.caveat as string));
  assert.ok(texts.some((t) => t.startsWith('strong (high 4). No result fell below high confidence.')));
  assert.ok(limits(model).every((l) => l.author === 'tool'));
});

test('impact: rows are grouped by kind in the order the tool returned them', () => {
  const model = buildResultModel(answer('analyze_impact.json'), { kind: 'impact', symbol: 'parseToolArg' });
  assert.deepEqual(
    model.sections.slice(1).map((s) => s.title),
    ['Direct: proven call edge to this symbol (2)', 'Indirect: reaches this symbol through other calls (2)'],
  );
  const indirect = results(model).filter((r) => r.kind === 'indirect');
  assert.equal(indirect[0]?.description, 'high · internal/cli/tool.go');
  assert.equal(indirect[0]?.line, undefined);
  assert.match(indirect[0]?.tooltip ?? '', /RepoCortex reported no line for this row/u);
});

test('implementations: subtypes and supertypes are separate groups, medium tier shown as medium', () => {
  const model = buildResultModel(answer('find_implementations_both.json'), { kind: 'implementations', symbol: 'Impacted' });
  assert.deepEqual(
    model.sections.slice(1).map((s) => s.title),
    ['Implements or extends this (1)', 'This implements or extends (1)'],
  );
  assert.deepEqual(
    results(model).map((r) => r.confidence),
    ['high', 'medium'],
  );
  assert.equal(model.title, 'Implementations of Impacted');
});

test('tests: ten rows, none dropped', () => {
  const model = buildResultModel(answer('find_tests.json'), { kind: 'tests', symbol: 'parseToolArg' });
  assert.equal(results(model).length, 10);
  assert.equal(model.sections[1]?.title, 'Tests (10)');
  assert.equal(model.title, 'Tests that reach parseToolArg');
});

test('an empty answer still shows why it may be empty', () => {
  const a = answer('find_callers_wrong_target.json');
  const model = buildResultModel(a, { kind: 'callers', symbol: 'New' });
  assert.equal(results(model).length, 0);
  assert.equal(model.sections.length, 1);
  assert.ok(limits(model).some((l) => /LOWER BOUND, not proof there are none/u.test(l.text) && l.warning));
});

test('extension warnings come first and are marked as the extension own words', () => {
  const model = buildResultModel(answer('find_callers_wrong_target.json'), {
    kind: 'callers',
    symbol: 'New',
    extensionWarnings: ['This answer is for another definition.'],
  });
  const all = limits(model);
  assert.deepEqual(all[0], { type: 'limit', label: 'Check', text: 'This answer is for another definition.', warning: true, author: 'extension' });
  assert.ok(all.slice(1).every((l) => l.author === 'tool'));
});

test('a row with no confidence says so instead of inventing one', () => {
  const a = answer('find_callers.json');
  const stripped = { ...a, items: [{ kind: 'calls', name: 'x', path: 'a.go', line: 2 }] };
  const row = results(buildResultModel(stripped, { kind: 'callers', symbol: 'y' }))[0];
  assert.equal(row?.description, 'no tier · a.go:2');
  assert.equal(row?.confidence, '');
  assert.match(row?.tooltip ?? '', /reported no confidence tier/u);
});

test('the full report holds the exact command and the tool text verbatim', () => {
  const a = answer('find_callers.json');
  const model = buildResultModel(a, { kind: 'callers', symbol: 'parseToolArg', extensionWarnings: ['Extension note.'] });
  const report = fullReport(model, a, ['--project=/work/repo', '--json', 'tool', 'call', 'find_callers', '--args={"name":"parseToolArg"}']);
  assert.ok(report.includes('  --project=/work/repo\n  --json\n  tool\n  call\n  find_callers\n  --args={"name":"parseToolArg"}\n'));
  assert.ok(report.includes('Notes from the extension (not from RepoCortex):\n  - Extension note.'));
  assert.ok(report.includes(a.text.trimEnd()));
});

test('a limit is one labelled line, and a note the tool already labelled is not labelled twice', () => {
  assert.equal(limitLine({ label: 'Truncated', text: 'depth limit 3 reached' }), 'Truncated: depth limit 3 reached');
  const model = buildResultModel(answer('find_implementations.json'), { kind: 'implementations', symbol: 'Symbol' });
  const lines = limits(model).map(limitLine);
  assert.ok(lines.includes("Note: 5 result(s) are medium or low confidence (heuristic inference with higher error rates). Pass min_confidence='high' to restrict to compiler-verified and high-precision edges."));
  assert.ok(!lines.some((line) => line.startsWith('Note: Note:')));
  // Indented risk factors keep their label.
  const impact = limits(buildResultModel(answer('analyze_impact.json'), { kind: 'impact', symbol: 'parseToolArg' })).map(limitLine);
  assert.ok(impact.some((line) => line.startsWith('Note: public_api (+0):')));
});
