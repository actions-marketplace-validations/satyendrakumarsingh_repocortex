import assert from 'node:assert/strict';
import { test } from 'node:test';

import {
  checkSymbolName,
  expandBinaryPath,
  findSymbolArguments,
  indexStepArgv,
  indexStepsFor,
  MAX_SYMBOL_LENGTH,
  questionArguments,
  statusArgv,
  toolCallArgv,
} from '../src/core/args';
import { normalizeMinConfidence } from '../src/core/confidence';

test('a tool call is one argv array with the arguments as a single JSON value', () => {
  const argv = toolCallArgv('/work/repo', 'find_callers', { name: 'parseToolArg', path: 'internal/cli/tool.go', min_confidence: 'high' });
  assert.deepEqual(argv, [
    '--project=/work/repo',
    '--json',
    'tool',
    'call',
    'find_callers',
    '--args={"name":"parseToolArg","path":"internal/cli/tool.go","min_confidence":"high"}',
  ]);
});

test('editor text never becomes its own argument, whatever it contains', () => {
  // Each of these would be dangerous or wrong in the CLI's key=value form or
  // on a shell command line. As JSON inside one argv element they are inert.
  const hostile = ['@/etc/passwd', 'true', 'null', '42', '--project=/tmp', 'a"; rm -rf ~; "', '$(id)', '`id`', 'x y', "it's"];
  for (const name of hostile) {
    const argv = toolCallArgv('/work/repo', 'find_symbol', findSymbolArguments(name));
    assert.equal(argv.length, 6, name);
    assert.equal(argv[4], 'find_symbol');
    const last = argv[5] as string;
    assert.ok(last.startsWith('--args='), name);
    assert.deepEqual(JSON.parse(last.slice('--args='.length)), { name }, name);
  }
});

test('a project path that looks like a flag stays the value of --project', () => {
  assert.equal(toolCallArgv('--json', 'find_symbol', { name: 'x' })[0], '--project=--json');
  assert.deepEqual(statusArgv('/work/my repo'), ['--project=/work/my repo', '--json', 'status']);
});

test('only the tools the extension knows can be called', () => {
  assert.throws(() => toolCallArgv('/r', 'minimize_diff' as never, {}), /unknown tool/u);
});

test('callers and implementations send name, a path hint and the floor', () => {
  assert.deepEqual(questionArguments('callers', { name: 'New', path: 'internal/web/web.go', fqn: 'internal/web/web.go:New' }, 'low'), {
    name: 'New',
    path: 'internal/web/web.go',
    min_confidence: 'low',
  });
  assert.deepEqual(questionArguments('implementations', { name: 'Symbol' }, 'high'), { name: 'Symbol', min_confidence: 'high' });
});

test('impact and tests send the fully qualified name when there is one', () => {
  assert.deepEqual(questionArguments('impact', { name: 'New', path: 'internal/web/web.go', fqn: 'internal/web/web.go:New' }, 'medium'), {
    symbol: 'internal/web/web.go:New',
    min_confidence: 'medium',
  });
  assert.deepEqual(questionArguments('tests', { name: 'parseToolArg', path: 'internal/cli/tool.go' }, 'low'), {
    symbol: 'parseToolArg',
    min_confidence: 'low',
  });
});

test('symbol names are checked before use', () => {
  assert.deepEqual(checkSymbolName('  parseToolArg '), { ok: true, name: 'parseToolArg' });
  assert.equal(checkSymbolName('').ok, false);
  assert.equal(checkSymbolName('   ').ok, false);
  assert.equal(checkSymbolName('two words').ok, false);
  assert.equal(checkSymbolName('line\nbreak').ok, false);
  assert.equal(checkSymbolName('nul\u0000byte').ok, false);
  assert.equal(checkSymbolName('=>').ok, false);
  assert.equal(checkSymbolName('x'.repeat(MAX_SYMBOL_LENGTH + 1)).ok, false);
  assert.equal(checkSymbolName('x'.repeat(MAX_SYMBOL_LENGTH)).ok, true);
  // Identifiers outside ASCII are real identifiers.
  assert.equal(checkSymbolName('größe').ok, true);
  assert.equal(checkSymbolName('$scope').ok, true);
});

test('the index steps depend on whether a database exists', () => {
  assert.deepEqual(indexStepsFor('absent'), ['init', 'index', 'resolve']);
  assert.deepEqual(indexStepsFor(undefined), ['init', 'index', 'resolve']);
  assert.deepEqual(indexStepsFor('empty'), ['index', 'resolve']);
  assert.deepEqual(indexStepsFor('ready'), ['index', 'resolve']);
  assert.deepEqual(indexStepArgv('/work/repo', 'resolve'), ['--project=/work/repo', 'resolve']);
});

test('the binary path expands a leading tilde and nothing else', () => {
  assert.equal(expandBinaryPath('', '/home/u'), 'repocortex');
  assert.equal(expandBinaryPath('repocortex', '/home/u'), 'repocortex');
  assert.equal(expandBinaryPath('~/go/bin/repocortex', '/home/u'), '/home/u/go/bin/repocortex');
  assert.equal(expandBinaryPath('~/go/bin/repocortex', '/home/u/'), '/home/u/go/bin/repocortex');
  assert.equal(expandBinaryPath('/opt/$HOME/repocortex', '/home/u'), '/opt/$HOME/repocortex');
  assert.equal(expandBinaryPath('~other/bin/repocortex', '/home/u'), '~other/bin/repocortex');
});

test('the minimum confidence setting falls back to low', () => {
  assert.equal(normalizeMinConfidence('HIGH'), 'high');
  assert.equal(normalizeMinConfidence(' exact '), 'exact');
  assert.equal(normalizeMinConfidence('bogus'), 'low');
  assert.equal(normalizeMinConfidence(undefined), 'low');
  assert.equal(normalizeMinConfidence(3), 'low');
});
