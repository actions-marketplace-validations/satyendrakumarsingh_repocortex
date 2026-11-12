import assert from 'node:assert/strict';
import { test } from 'node:test';

import type { OutlineSymbol } from '../src/core/lens';
import { bareSymbolName, lensCacheKey, lensCount, lensTargets, lensTitle, lensTooltip, MAX_LENSES_PER_FILE } from '../src/core/lens';
import { answer } from './helpers';

test('outline names are reduced to the bare identifier the index knows', () => {
  assert.equal(bareSymbolName('parseToolArg'), 'parseToolArg');
  assert.equal(bareSymbolName('(*Server).findCallers'), 'findCallers'); // gopls
  assert.equal(bareSymbolName('(Confidence).String'), 'String');
  assert.equal(bareSymbolName('handle(Request, int)'), 'handle'); // Java
  assert.equal(bareSymbolName('Box<T>'), 'Box');
  assert.equal(bareSymbolName('Parser::parse'), 'parse'); // C++
  assert.equal(bareSymbolName('Outer.inner'), 'inner');
  assert.equal(bareSymbolName('größe'), 'größe');
  assert.equal(bareSymbolName('<function>'), undefined); // TypeScript anonymous
  assert.equal(bareSymbolName("describe('x') callback"), undefined);
  assert.equal(bareSymbolName(''), undefined);
  assert.equal(bareSymbolName('operator=='), undefined);
});

const outline: OutlineSymbol[] = [
  { name: 'maxItems', category: 'other', line: 3 },
  {
    name: 'Server',
    category: 'type',
    line: 10,
    children: [
      { name: 'port', category: 'other', line: 11 },
      { name: 'start', category: 'function', line: 14, children: [{ name: 'helper', category: 'function', line: 16 }] },
      { name: 'String', category: 'function', line: 30 },
    ],
  },
  { name: '(*Server).stop', category: 'function', line: 40 },
  { name: '(Mode).String', category: 'function', line: 50 },
  { name: '<function>', category: 'function', line: 60 },
];

test('lenses go on functions and types, in document order, not on locals', () => {
  const targets = lensTargets(outline);
  assert.deepEqual(
    targets.map((t) => [t.name, t.line, t.category, t.sharedName]),
    [
      ['Server', 10, 'type', false],
      ['start', 14, 'function', false],
      ['String', 30, 'function', true],
      ['stop', 40, 'function', false],
      ['String', 50, 'function', true],
    ],
  );
});

test('the number of lenses per file is capped', () => {
  const big: OutlineSymbol[] = Array.from({ length: 500 }, (_, i) => ({ name: `f${i}`, category: 'function' as const, line: i * 3 }));
  assert.equal(lensTargets(big).length, MAX_LENSES_PER_FILE);
  assert.equal(lensTargets(big, 5).length, 5);
  assert.deepEqual(lensTargets(big, 0), []);
  // Two symbols on one line give one lens.
  const sameLine: OutlineSymbol[] = [
    { name: 'a', category: 'function', line: 1 },
    { name: 'b', category: 'function', line: 1 },
  ];
  assert.equal(lensTargets(sameLine).length, 1);
});

test('count: call sites and distinct callers are both reported', () => {
  const a = answer('find_callers.json');
  const count = lensCount(a, 'internal/cli/tool.go');
  assert.deepEqual(count, { kind: 'count', callers: 2, sites: 3, atLeast: false });
  assert.equal(lensTitle(count, { sharedName: false }), 'RepoCortex: 2 callers, 3 call sites');
  assert.equal(lensTitle({ kind: 'count', callers: 1, sites: 1, atLeast: false }, { sharedName: false }), 'RepoCortex: 1 caller');
  assert.equal(lensTitle(count, { sharedName: true }), 'RepoCortex: 2 callers, 3 call sites (name shared in this file)');
});

test('count: a list the tool cut short is shown as a floor', () => {
  const a = answer('find_callers_truncated.json');
  const count = lensCount(a, 'internal/cli/commands.go');
  assert.ok(count.kind === 'count');
  assert.equal(count.sites, 50);
  assert.equal(count.atLeast, true);
  assert.match(lensTitle(count, { sharedName: false }), /^RepoCortex: \d+\+ callers, 50\+ call sites$/u);
  assert.match(lensTooltip(count, a, 'low'), /Truncated: the item cap \(50\) was reached/u);
});

test('count: zero is "found", and the tooltip carries the tool warning that it is a lower bound', () => {
  const a = answer('find_callers_floor.json');
  const count = lensCount(a, 'internal/cli/commands.go');
  assert.deepEqual(count, { kind: 'count', callers: 0, sites: 0, atLeast: false });
  assert.equal(lensTitle(count, { sharedName: false }), 'RepoCortex: 0 callers found');
  const tooltip = lensTooltip(count, a, 'exact');
  assert.match(tooltip, /55 lower-confidence result\(s\) were omitted/u);
  assert.match(tooltip, /LOWER BOUND, not proof there are none/u);
  assert.match(tooltip, /Counting results at confidence "exact" and above/u);
});

test('a count for a same-named definition in another file is never shown as this one', () => {
  // Captured from `find_callers name=New path=internal/web/web.go`.
  const a = answer('find_callers_wrong_target.json');
  const count = lensCount(a, 'internal/web/web.go');
  assert.deepEqual(count, { kind: 'other-definition', path: 'internal/capability/ownership/ownership.go', line: 136 });
  assert.equal(lensTitle(count, { sharedName: false }), 'RepoCortex: callers unknown (name is shared)');
  assert.match(lensTooltip(count, a, 'low'), /answered for the definition at internal\/capability\/ownership\/ownership\.go:136, not this one/u);
  // The very same answer is a real count for the file it is about.
  assert.equal(lensCount(a, 'internal/capability/ownership/ownership.go').kind, 'count');
});

test('a symbol the index does not have gets no number', () => {
  const a = answer('find_callers_missing.json');
  const count = lensCount(a, 'internal/cli/tool.go');
  assert.deepEqual(count, { kind: 'not-indexed' });
  assert.equal(lensTitle(count, { sharedName: false }), 'RepoCortex: not in the index');
  assert.match(lensTooltip(count, a, 'low'), /No symbol named "NoSuchSymbolAnywhere" in the index\./u);
});

test('the cache key changes with the index, the floor, the file and the name', () => {
  const base = lensCacheKey('/r', 'ready|1|t', 'low', 'a.go', 'f');
  assert.equal(base, lensCacheKey('/r', 'ready|1|t', 'low', 'a.go', 'f'));
  assert.notEqual(base, lensCacheKey('/r', 'ready|2|t2', 'low', 'a.go', 'f'));
  assert.notEqual(base, lensCacheKey('/r', 'ready|1|t', 'high', 'a.go', 'f'));
  assert.notEqual(base, lensCacheKey('/r', 'ready|1|t', 'low', 'b.go', 'f'));
  assert.notEqual(base, lensCacheKey('/r', 'ready|1|t', 'low', 'a.go', 'g'));
  assert.notEqual(base, lensCacheKey('/other', 'ready|1|t', 'low', 'a.go', 'f'));
});
