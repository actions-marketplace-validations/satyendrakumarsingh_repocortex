import assert from 'node:assert/strict';
import { test } from 'node:test';

import { questionArguments } from '../src/core/args';
import type { Definition } from '../src/core/target';
import { answeredLocation, chooseDefinition, definitionsFrom, describeDefinition, targetFor, targetWarnings } from '../src/core/target';
import { answer } from './helpers';

const one = definitionsFrom(answer('find_symbol.json'));
const many = definitionsFrom(answer('find_symbol_many.json'));

function def(path: string, line: number, name = 'String'): Definition {
  return { name, kind: 'method', path, line, fqn: `${path}:${name}` };
}

test('definitions are read from a find_symbol answer', () => {
  assert.deepEqual(one, [
    { name: 'parseToolArg', kind: 'function', path: 'internal/cli/tool.go', line: 384, fqn: 'internal/cli/tool.go:parseToolArg', confidence: 'high' },
  ]);
  assert.equal(many.length, 15);
  assert.ok(many.every((d) => d.name === 'New'));
  assert.deepEqual(definitionsFrom(answer('find_symbol_missing.json')), []);
});

test('nothing in the index means nothing is chosen', () => {
  assert.deepEqual(chooseDefinition([], { source: 'cursor', path: 'a.go', line: 1 }), { kind: 'none' });
});

test('a single definition is taken when only the cursor position is known', () => {
  const choice = chooseDefinition(one, { source: 'cursor', path: 'internal/cli/structure_test.go', line: 254 });
  assert.deepEqual(choice, { kind: 'chosen', definition: one[0], basis: 'only-one' });
  assert.equal(chooseDefinition(one, { source: 'none' }).kind, 'chosen');
});

test('go-to-definition picks the right one of many', () => {
  const choice = chooseDefinition(many, { source: 'definition', path: 'internal/web/web.go', line: 309 });
  assert.ok(choice.kind === 'chosen');
  assert.equal(choice.basis, 'matches-hint');
  assert.equal(choice.definition.path, 'internal/web/web.go');
});

test('many definitions and no usable hint: the user chooses', () => {
  const choice = chooseDefinition(many, { source: 'cursor', path: 'cmd/repocortex/main.go', line: 10 });
  assert.ok(choice.kind === 'ask');
  assert.equal(choice.reason, 'ambiguous');
  assert.equal(choice.candidates.length, 15);
});

test('a definition the index does not have is never silently swapped for a namesake', () => {
  // The language server says the symbol lives in the standard library. The
  // index has one unrelated function with the same name.
  const outside = chooseDefinition(one, { source: 'definition', outsideProject: '/usr/local/go/src/strings/strings.go' });
  assert.ok(outside.kind === 'ask');
  assert.equal(outside.reason, 'hint-mismatch');

  // Inside the project, but in a file where the index has no such definition.
  const unindexed = chooseDefinition(one, { source: 'definition', path: 'internal/gen/zz_generated.go', line: 12 });
  assert.ok(unindexed.kind === 'ask');
  assert.equal(unindexed.reason, 'hint-mismatch');
});

test('several definitions in one file: the hinted line decides', () => {
  const defs = [def('data.go', 156), def('data.go', 201), def('other.go', 5)];
  const exact = chooseDefinition(defs, { source: 'definition', path: 'data.go', line: 201 });
  assert.ok(exact.kind === 'chosen' && exact.definition.line === 201);
  // The buffer has drifted a few lines from the index: nearest one above.
  const drifted = chooseDefinition(defs, { source: 'definition', path: 'data.go', line: 204 });
  assert.ok(drifted.kind === 'chosen' && drifted.definition.line === 201);
  const between = chooseDefinition(defs, { source: 'cursor', path: 'data.go', line: 170 });
  assert.ok(between.kind === 'chosen' && between.definition.line === 156);
  // Above every candidate, or no line at all: ask, but only about that file.
  const above = chooseDefinition(defs, { source: 'cursor', path: 'data.go', line: 3 });
  assert.ok(above.kind === 'ask' && above.candidates.length === 2);
  const noLine = chooseDefinition(defs, { source: 'cursor', path: 'data.go' });
  assert.ok(noLine.kind === 'ask' && noLine.candidates.length === 2);
});

test('the fully qualified name is sent only when the bare name is ambiguous', () => {
  const single = targetFor(one[0] as Definition, one);
  assert.deepEqual(single, { name: 'parseToolArg', path: 'internal/cli/tool.go' });
  assert.deepEqual(questionArguments('impact', single, 'low'), { symbol: 'parseToolArg', min_confidence: 'low' });

  const web = many.find((d) => d.path === 'internal/web/web.go') as Definition;
  const target = targetFor(web, many);
  assert.deepEqual(target, { name: 'New', path: 'internal/web/web.go', fqn: 'internal/web/web.go:New' });
  assert.deepEqual(questionArguments('tests', target, 'high'), { symbol: 'internal/web/web.go:New', min_confidence: 'high' });
  assert.deepEqual(questionArguments('callers', target, 'high'), { name: 'New', path: 'internal/web/web.go', min_confidence: 'high' });
});

test('the answered definition is read back from the summary', () => {
  assert.deepEqual(answeredLocation(answer('find_callers.json').summary), { path: 'internal/cli/tool.go', line: 384 });
  assert.deepEqual(answeredLocation(answer('find_implementations.json').summary), { path: 'internal/capability/symbols.go', line: 19 });
  assert.deepEqual(answeredLocation(answer('find_callers_stale.json').summary), {
    path: 'src/main/java/com/pay/model/Transaction.java',
    line: 10,
  });
  assert.equal(answeredLocation(answer('analyze_impact.json').summary), undefined);
  assert.equal(answeredLocation(answer('find_tests.json').summary), undefined);
  assert.equal(answeredLocation(answer('find_callers_missing.json').summary), undefined);
  assert.equal(answeredLocation(''), undefined);
});

test('an answer for a different definition than the one asked about is flagged', () => {
  // Captured from `find_callers name=New path=internal/web/web.go`: the tool
  // answered for internal/capability/ownership/ownership.go:136 instead.
  const wrong = answer('find_callers_wrong_target.json');
  const web = many.find((d) => d.path === 'internal/web/web.go') as Definition;
  const warnings = targetWarnings('callers', web, many, wrong);
  assert.equal(warnings.length, 1);
  assert.match(warnings[0] as string, /is for the definition at internal\/capability\/ownership\/ownership\.go:136, NOT the one you asked about \(internal\/web\/web\.go:309\)/u);

  // The same answer is correct when ownership.go was the one asked about.
  const ownership = many.find((d) => d.path === 'internal/capability/ownership/ownership.go') as Definition;
  assert.deepEqual(targetWarnings('callers', ownership, many, wrong), []);
});

test('no warning when the name is unique', () => {
  assert.deepEqual(targetWarnings('callers', one[0] as Definition, one, answer('find_callers.json')), []);
  assert.deepEqual(targetWarnings('impact', one[0] as Definition, one, answer('analyze_impact.json')), []);
});

test('same-file namesakes share a fully qualified name, and the impact tools cannot tell them apart', () => {
  const defs = [def('data.go', 156), def('data.go', 201), def('other.go', 5)];
  const warnings = targetWarnings('impact', defs[1] as Definition, defs, answer('analyze_impact.json'));
  assert.equal(warnings.length, 1);
  assert.match(warnings[0] as string, /2 definitions named "String" are in data\.go/u);
  // A namesake in another file is told apart by the qualified name: no warning.
  assert.deepEqual(targetWarnings('tests', defs[2] as Definition, defs, answer('find_tests.json')), []);
});

test('a shared name with no answered location is stated, not hidden', () => {
  const web = many.find((d) => d.path === 'internal/web/web.go') as Definition;
  const warnings = targetWarnings('callers', web, many, { ...answer('find_callers_wrong_target.json'), summary: 'something new' });
  assert.equal(warnings.length, 1);
  assert.match(warnings[0] as string, /did not say which one it answered for/u);
});

test('a definition is described with its kind and location', () => {
  assert.equal(describeDefinition(one[0] as Definition), 'function  internal/cli/tool.go:384');
  assert.equal(describeDefinition({ name: 'x', kind: '', path: 'a.go', line: 0 }), 'a.go');
});
