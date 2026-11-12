import assert from 'node:assert/strict';
import { test } from 'node:test';

import { EnvelopeError, failureMessage, interpretToolRun, limitsOf, parseToolEnvelope, verdictLine } from '../src/core/result';
import { answer, fixture } from './helpers';

test('find_callers: every item keeps its tier, location and reason', () => {
  const a = answer('find_callers.json');
  assert.equal(a.tool, 'find_callers');
  assert.equal(a.isError, false);
  assert.equal(a.summary, '3 caller(s) of parseToolArg (internal/cli/tool.go:384).');
  assert.equal(a.indexStatus, 'ready');
  assert.equal(a.indexRevision, 1);
  assert.equal(a.items.length, 3);
  assert.deepEqual(a.items[2], {
    kind: 'calls',
    name: 'newToolCallCmd',
    path: 'internal/cli/tool.go',
    line: 323,
    confidence: 'high',
    source: 'source_code',
    why: 'defined in the same file',
  });
  assert.equal(a.truncated, false);
  assert.equal(a.verdict?.status, 'strong');
  assert.deepEqual(a.verdict?.counts, { high: 3 });
  assert.equal(a.provenance?.headMatchesBuild, true);
  assert.match(a.provenance?.caveat ?? '', /commits only/u);
  assert.match(a.text, /^=== SYSTEM METADATA ===/u);
});

test('analyze_impact: indirect rows have no line, and the tool says what it could not see', () => {
  const a = answer('analyze_impact.json');
  assert.deepEqual(
    a.items.map((i) => [i.kind, i.name, i.line]),
    [
      ['direct', 'TestParseToolArgTypes', 254],
      ['direct', 'newToolCallCmd', 323],
      ['indirect', 'newToolCmd', undefined],
      ['indirect', 'NewRoot', undefined],
    ],
  );
  assert.equal(a.truncated, true);
  assert.match(a.truncatedReason ?? '', /depth limit 3 reached/u);
  const limits = limitsOf(a);
  const texts = limits.map((l) => l.text);
  // Nothing the tool said is dropped: one truncation, every note, the caveat.
  assert.equal(limits.length, 1 + a.notes.length + 1);
  assert.ok(texts.some((t) => t.startsWith('NOT VISIBLE (unexpanded_frontier)')));
  assert.ok(texts.some((t) => t.startsWith('NOT VISIBLE (unresolved_references)')));
  assert.ok(texts.some((t) => t.startsWith('This is a LOWER BOUND.')));
  for (const note of a.notes) {
    assert.ok(texts.includes(note), note);
  }
  assert.equal(limits[0]?.kind, 'truncated');
  assert.equal(limits.at(-1)?.kind, 'caveat');
});

test('analyze_impact: results left out to fit the response are reported with their breakdown', () => {
  const a = answer('analyze_impact_omitted.json');
  assert.equal(a.items.length, 50);
  assert.deepEqual(a.omitted, {
    count: 51,
    byKind: [
      { label: 'indirect', count: 49 },
      { label: 'direct', count: 2 },
    ],
    byConfidence: [{ label: 'high', count: 51 }],
    weakestShown: 'high',
    strongerDropped: [],
  });
  const omitted = limitsOf(a).find((l) => l.kind === 'omitted');
  assert.ok(omitted);
  assert.match(omitted.text, /^51 result\(s\) were left out/u);
  assert.match(omitted.text, /indirect 49, direct 2/u);
});

test('find_tests and find_implementations parse their own item kinds', () => {
  const tests = answer('find_tests.json');
  assert.equal(tests.items.length, 10);
  assert.ok(tests.items.every((i) => i.kind === 'test' && i.line === undefined && i.confidence === 'high'));
  assert.equal(tests.truncated, true);

  const impl = answer('find_implementations.json');
  assert.equal(impl.items.length, 5);
  assert.ok(impl.items.every((i) => i.kind === 'subtype' && i.confidence === 'medium'));
  assert.equal(impl.verdict?.status, 'weak');
  assert.deepEqual(impl.verdict?.signals, ['all 5 results are inferred (5 medium, 0 low); none was compiler-proven']);
  assert.ok(limitsOf(impl).some((l) => l.kind === 'signal'));

  const both = answer('find_implementations_both.json');
  assert.deepEqual(
    both.items.map((i) => i.kind),
    ['subtype', 'supertype'],
  );
});

test('a stale index is the first limit, in the tool own words', () => {
  const a = answer('find_callers_stale.json');
  assert.equal(a.provenance?.headMatchesBuild, false);
  const limits = limitsOf(a);
  assert.equal(limits[0]?.kind, 'stale');
  assert.equal(limits[0]?.warning, true);
  assert.match(limits[0]?.text ?? '', /^The index was built from 0571f90c but HEAD is d28fecc3\./u);
  // The caveat is shown once, not again at the end.
  assert.equal(limits.filter((l) => l.text === a.provenance?.caveat).length, 1);
  assert.ok(limits.some((l) => l.kind === 'signal' && /does not match HEAD/u.test(l.text)));
});

test('a confidence floor is reported by the tool and passed through', () => {
  const a = answer('find_callers_floor.json');
  assert.equal(a.items.length, 0);
  assert.match(a.notes[0] ?? '', /^Filtered to confidence "exact" and above: 55 lower-confidence result\(s\) were omitted/u);
  assert.ok(limitsOf(a).some((l) => l.warning && /omitted/u.test(l.text)));
});

test('exit 0 with items is an answer', () => {
  const outcome = interpretToolRun({ code: 0, stdout: fixture('find_callers.json'), stderr: '' });
  assert.equal(outcome.kind, 'answer');
});

test('exit 0 with nothing found is still an answer', () => {
  const outcome = interpretToolRun({ code: 0, stdout: fixture('find_callers_missing.json'), stderr: '' });
  assert.equal(outcome.kind, 'answer');
  assert.ok(outcome.kind === 'answer' && outcome.answer.items.length === 0);
});

test('exit 1 with is_error is a tool failure carrying the tool message', () => {
  const bad = interpretToolRun({ code: 1, stdout: fixture('error_bad_confidence.json'), stderr: '' });
  assert.equal(bad.kind, 'tool-failure');
  assert.ok(bad.kind === 'tool-failure');
  assert.match(bad.message, /^unknown confidence "bogus": use exact, high, medium or low/u);
  assert.match(bad.message, /hint: /u);
  assert.doesNotMatch(bad.message, /SYSTEM METADATA|error_code/u);

  // A schema rejection has `result: null` and one bare line of text.
  const missing = interpretToolRun({ code: 1, stdout: fixture('error_missing_argument.json'), stderr: '' });
  assert.ok(missing.kind === 'tool-failure');
  assert.equal(missing.answer.hasResult, false);
  assert.match(missing.message, /missing properties: \["name"\]/u);

  const ambiguous = interpretToolRun({ code: 1, stdout: fixture('analyze_impact_ambiguous.json'), stderr: '' });
  assert.ok(ambiguous.kind === 'tool-failure');
  assert.match(ambiguous.message, /"New" matches 15 symbols; re-ask with a fully-qualified name/u);
});

test('no index: the CLI error on stderr, and the success-shaped "absent" answer', () => {
  const cli = interpretToolRun({ code: 1, stdout: '', stderr: fixture('stderr_no_index.txt') });
  assert.equal(cli.kind, 'no-index');
  assert.ok(cli.kind === 'no-index');
  assert.match(cli.message, /^no index at .*run 'repocortex init' and 'repocortex index' first$/u);

  const shaped = interpretToolRun({ code: 0, stdout: fixture('not_indexed.json'), stderr: '' });
  assert.equal(shaped.kind, 'no-index');
  assert.ok(shaped.kind === 'no-index');
  assert.match(shaped.message, /^No index has been built for this project yet/u);
  assert.match(shaped.message, /Run 'repocortex index'/u);
});

test('a CLI failure that is not about the index keeps its message', () => {
  const unknown = interpretToolRun({ code: 1, stdout: '', stderr: fixture('stderr_unknown_tool.txt') });
  assert.deepEqual(unknown, {
    kind: 'cli-error',
    message: 'no tool named "nope" on the "full" profile. Run \'repocortex tool list\' to see every tool.',
    exitCode: 1,
  });
  const silent = interpretToolRun({ code: 3, stdout: '', stderr: '' });
  assert.ok(silent.kind === 'cli-error');
  assert.match(silent.message, /status 3 and printed nothing/u);
  const garbage = interpretToolRun({ code: 0, stdout: 'not json', stderr: '' });
  assert.equal(garbage.kind, 'cli-error');
});

test('malformed envelopes are rejected, and wrong field types are ignored rather than trusted', () => {
  assert.throws(() => parseToolEnvelope('{'), EnvelopeError);
  assert.throws(() => parseToolEnvelope('[]'), EnvelopeError);
  assert.throws(() => parseToolEnvelope('{"tool":"x"}'), EnvelopeError);

  const odd = parseToolEnvelope(
    JSON.stringify({
      tool: 'find_callers',
      is_error: false,
      text: 7,
      result: {
        summary: ['not', 'a', 'string'],
        index_revision: '1',
        items: [null, 'x', { kind: 'calls', name: 5, line: -3, path: 'a.go', confidence: 'high' }, { kind: 'calls', line: 2.5 }],
        notes: ['kept', 9, ''],
        truncated: 'yes',
        provenance: { head_matches_build: 'true' },
        omitted: { count: 0 },
      },
    }),
  );
  assert.equal(odd.text, '');
  assert.equal(odd.summary, '');
  assert.equal(odd.indexRevision, 0);
  assert.deepEqual(odd.items, [{ kind: 'calls', path: 'a.go', confidence: 'high' }, { kind: 'calls' }]);
  assert.deepEqual(odd.notes, ['kept']);
  assert.equal(odd.truncated, false);
  // Anything but a literal true means "not known to match".
  assert.equal(odd.provenance?.headMatchesBuild, false);
  assert.equal(odd.omitted, undefined);
});

test('the failure message never comes back empty', () => {
  const blank = parseToolEnvelope('{"tool":"x","is_error":true,"text":"","result":null}');
  assert.equal(failureMessage(blank), 'The tool reported a failure and gave no message.');
});

test('the verdict line lists tiers strongest first', () => {
  assert.equal(verdictLine({ status: 'mixed', counts: { low: 2, exact: 1, high: 4 }, signals: [], nextStep: '' }), 'mixed (exact 1, high 4, low 2)');
  assert.equal(verdictLine({ status: 'empty', counts: {}, signals: [], nextStep: '' }), 'empty');
});
