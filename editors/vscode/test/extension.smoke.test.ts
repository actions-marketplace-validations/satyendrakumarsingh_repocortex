// Runs the extension's real code — activation, commands, results tree,
// CodeLens, status bar, index task — against a stand-in for the `vscode`
// module and a stub `repocortex` executable that replays captured output.
//
// What this proves: the extension starts the binary with the argument arrays
// it should, reads the real JSON shapes, and drives the editor API as
// modelled in fakeVscode.ts. What it does not prove: anything about how the
// real VS Code renders, schedules or lays out. That needs a person and a
// window.

import assert from 'node:assert/strict';
import * as fs from 'node:fs';
import Module = require('node:module');
import * as os from 'node:os';
import * as path from 'node:path';
import { after, before, test } from 'node:test';

import type { Control } from './fakeVscode';
import { createFakeVscode } from './fakeVscode';
import { EXTENSION_DIR, FIXTURE_DIR } from './helpers';

const posixOnly = process.platform === 'win32' ? 'the stub binary is a shebang script' : false;

// The stub stands in for the repocortex binary. It appends the argv it was
// started with to a log, then prints a captured sample.
const STUB = `#!/usr/bin/env node
const fs = require('fs');
const path = require('path');
const fixtures = process.env.RC_STUB_FIXTURES;
const argv = process.argv.slice(2);
fs.appendFileSync(process.env.RC_STUB_LOG, JSON.stringify(argv) + '\\n');
const project = (argv.find((a) => a.startsWith('--project=')) || '').slice('--project='.length);
const noIndex = fs.existsSync(path.join(project, '.stub-no-index'));
const read = (f) => fs.readFileSync(path.join(fixtures, f), 'utf8');
const rest = argv.filter((a) => !a.startsWith('--project=') && a !== '--json');
if (rest[0] === 'status') {
  process.stdout.write(read(noIndex ? 'status_absent.json' : 'status_ready.json'));
  process.exit(0);
}
if (rest[0] === 'init' || rest[0] === 'index' || rest[0] === 'resolve') process.exit(0);
if (rest[0] === 'tool' && rest[1] === 'call') {
  if (noIndex) { process.stderr.write(read('stderr_no_index.txt')); process.exit(1); }
  const tool = rest[2];
  const args = JSON.parse(rest[3].slice('--args='.length));
  const name = args.name || args.symbol || '';
  const table = {
    'get_project_overview|': ['get_project_overview.json', 0],
    'find_symbol|parseToolArg': ['find_symbol.json', 0],
    'find_callers|parseToolArg': ['find_callers.json', 0],
    'analyze_impact|parseToolArg': ['analyze_impact.json', 0],
    'find_tests|parseToolArg': ['find_tests.json', 0],
    'find_symbol|Symbol': ['find_symbol.json', 0, 'Symbol'],
    'find_implementations|Symbol': ['find_implementations.json', 0],
    'find_symbol|New': ['find_symbol_many.json', 0],
    'find_callers|New': ['find_callers_wrong_target.json', 0],
    'find_symbol|Boom': ['find_symbol.json', 0, 'Boom'],
    'analyze_impact|Boom': ['error_bad_confidence.json', 1],
    'find_symbol|Ghost': ['find_symbol_missing.json', 0],
  };
  const hit = table[tool + '|' + name];
  if (!hit) { process.stderr.write(read('stderr_unknown_tool.txt')); process.exit(1); }
  let text = read(hit[0]);
  if (hit[2]) text = text.split('parseToolArg').join(hit[2]);
  process.stdout.write(text);
  process.exit(hit[1]);
}
process.stderr.write('error: unknown command\\n');
process.exit(1);
`;

const RESULTS = 'repocortex.results';

let work: string;
let stub: string;
let log: string;
let control: Control;
let restoreLoad: (() => void) | undefined;
const subscriptions: { dispose(): unknown }[] = [];

function calls(): string[][] {
  return fs.existsSync(log)
    ? fs
        .readFileSync(log, 'utf8')
        .split('\n')
        .filter((l) => l !== '')
        .map((l) => JSON.parse(l) as string[])
    : [];
}

function toolCalls(): { tool: string; args: Record<string, unknown> }[] {
  return calls()
    .filter((argv) => argv[2] === 'tool' && argv[3] === 'call')
    .map((argv) => ({ tool: argv[4] as string, args: JSON.parse((argv[5] as string).slice('--args='.length)) as Record<string, unknown> }));
}

/** Tool calls made to answer a question, leaving out the status bar's own HEAD check. */
function questionCalls(): { tool: string; args: Record<string, unknown> }[] {
  return toolCalls().filter((c) => c.tool !== 'get_project_overview');
}

/** Waits until the status bar has finished any refresh that is running or queued. */
async function settle(): Promise<void> {
  await control.run('repocortex.refreshStatus');
}

function clearLog(): void {
  fs.rmSync(log, { force: true });
}

function write(relative: string, text: string): string {
  const file = path.join(work, relative);
  fs.mkdirSync(path.dirname(file), { recursive: true });
  fs.writeFileSync(file, text);
  return file;
}

before(() => {
  if (posixOnly !== false) return;
  work = fs.realpathSync(fs.mkdtempSync(path.join(os.tmpdir(), 'rc-vscode-smoke-')));
  fs.mkdirSync(path.join(work, 'project', '.repocortex'), { recursive: true });
  stub = path.join(work, 'repocortex-stub');
  fs.writeFileSync(stub, STUB, { mode: 0o755 });
  log = path.join(work, 'calls.log');
  process.env['RC_STUB_FIXTURES'] = FIXTURE_DIR;
  process.env['RC_STUB_LOG'] = log;

  // 30 lines of padding so line numbers resemble a real file.
  const pad = (n: number): string => Array.from({ length: n }, (_, i) => `// line ${i + 1}`).join('\n');
  write('project/internal/cli/tool.go', `${pad(8)}\n\tk, v, err := parseToolArg(pair, stdin)\nfunc parseToolArg(pair string) {\n}\nvar s Symbol\nvar x = New()\nvar b = Boom()\nvar g = Ghost()\n`);
  write('project/internal/cli/structure_test.go', `${pad(300)}\n`);
  write('project/internal/web/web.go', `${pad(5)}\nfunc New() {}\n`);

  const fake = createFakeVscode();
  control = fake.control;
  control.workspaceFolders = [{ uri: (fake.api['Uri'] as { file(p: string): never }).file(path.join(work, 'project')), name: 'project', index: 0 }];
  control.settings['repocortex.path'] = stub;

  const loader = Module as unknown as { _load: (request: string, ...rest: unknown[]) => unknown };
  const original = loader._load;
  loader._load = function (this: unknown, request: string, ...rest: unknown[]): unknown {
    return request === 'vscode' ? fake.api : original.call(this, request, ...rest);
  };
  restoreLoad = () => void (loader._load = original);

  // eslint-disable-next-line @typescript-eslint/no-require-imports
  const extension = require(path.join(EXTENSION_DIR, 'out', 'src', 'extension.js')) as { activate(context: unknown): void };
  extension.activate({ subscriptions });
});

after(() => {
  for (const s of subscriptions.splice(0)) s.dispose();
  restoreLoad?.();
  if (work !== undefined) fs.rmSync(work, { recursive: true, force: true });
});

const toolFile = (): string => path.join(work, 'project', 'internal', 'cli', 'tool.go');

test('activation registers every contributed command and shows the index state', { skip: posixOnly }, async () => {
  const manifest = JSON.parse(fs.readFileSync(path.join(EXTENSION_DIR, 'package.json'), 'utf8')) as {
    contributes: { commands: { command: string }[] };
  };
  for (const { command } of manifest.contributes.commands) {
    assert.ok(control.commands.has(command), `${command} is contributed but not registered`);
  }
  assert.equal(control.codeLensProviders.length, 1);
  assert.ok(control.treeViews.has(RESULTS));

  await control.run('repocortex.refreshStatus');
  const item = control.statusBarItems[0];
  assert.equal(item.command, 'repocortex.showStatus');
  assert.equal(item.visible, true);
  assert.match(item.text, /^\$\(database\) RepoCortex: (just now|unknown age|\d+[mhd] old)$/u);
  assert.match(item.tooltip, /724 files, 10743 symbols, 20640 relationships indexed\./u);
  assert.match(item.tooltip, /commits only/u);
  // Status is read with `status`, and the HEAD comparison with one tool call.
  const seen = new Set(calls().map((argv) => argv.slice(1).join(' ')));
  assert.deepEqual([...seen].sort(), ['--json status', '--json tool call get_project_overview --args={}']);
});

test('Show callers: looks the definition up, asks, and fills the view', { skip: posixOnly }, async () => {
  await settle();
  clearLog();
  control.openEditor(toolFile(), 8, 20); // on "parseToolArg" in the call
  await control.run('repocortex.showCallers');

  const project = path.join(work, 'project');
  assert.deepEqual(calls(), [
    [`--project=${project}`, '--json', 'tool', 'call', 'find_symbol', '--args={"name":"parseToolArg"}'],
    [
      `--project=${project}`,
      '--json',
      'tool',
      'call',
      'find_callers',
      '--args={"name":"parseToolArg","path":"internal/cli/tool.go","min_confidence":"low"}',
    ],
  ]);

  const tree = control.tree(RESULTS);
  assert.equal(tree.title, 'Callers of parseToolArg (function  internal/cli/tool.go:384)');
  assert.equal(tree.message, '3 caller(s) of parseToolArg (internal/cli/tool.go:384).');
  assert.equal(tree.description, '3 rows');
  const labels = tree.rows.map((r) => `${'  '.repeat(r.depth)}${r.item.label}${r.item.description !== undefined ? ` | ${r.item.description}` : ''}`);
  assert.deepEqual(labels.slice(-4), [
    'Call sites (3)',
    '  TestParseToolArgTypes | high · internal/cli/structure_test.go:254',
    '  TestParseToolArgTypes | high · internal/cli/structure_test.go:264',
    '  newToolCallCmd | high · internal/cli/tool.go:323',
  ]);
  // What the tool could not see is in the view, first, in its own words.
  assert.match(labels[0] as string, /^What RepoCortex says about this answer \(3\)$/u);
  assert.equal(labels[1], '  Note: 32615 references repository-wide are unresolved or ambiguous, so this list may be incomplete.');
  assert.match(labels[2] as string, /^ {2}Index: This compares commits only\./u);
  assert.match(labels[3] as string, /^ {2}Verdict: strong \(high 3\)\./u);
  assert.deepEqual(control.messages, []);
});

test('a result row opens its file at the reported line', { skip: posixOnly }, async () => {
  const row = control.tree(RESULTS).rows.find((r) => r.item.description === 'high · internal/cli/structure_test.go:254');
  assert.ok(row);
  assert.equal(row.item.command?.command, 'repocortex.openResult');
  await control.run('repocortex.openResult', ...(row.item.command?.arguments ?? []));
  const shown = control.shown.at(-1);
  assert.equal(shown?.document.uri.fsPath, path.join(work, 'project', 'internal', 'cli', 'structure_test.go'));
  assert.equal(shown?.options?.selection?.start.line, 253);

  // A path that leaves the project is refused, not opened.
  const before = control.shown.length;
  await control.run('repocortex.openResult', { type: 'result', label: 'x', path: '../../etc/passwd', line: 1 });
  assert.equal(control.shown.length, before);
  assert.match(control.messages.at(-1)?.text ?? '', /is not a path inside this project/u);
  control.messages.length = 0;
});

test('Show the full tool report: the verbatim tool text', { skip: posixOnly }, async () => {
  await control.run('repocortex.showFullReport');
  const text = control.shown.at(-1)?.document.getText() ?? '';
  assert.match(text, /=== SYSTEM METADATA ===\n3 caller\(s\) of parseToolArg/u);
  assert.match(text, /=== INFERRED ===/u);
  assert.match(text, /--args=\{"name":"parseToolArg","path":"internal\/cli\/tool\.go","min_confidence":"low"\}/u);
});

test('Show what this could break, Show tests, Show implementations', { skip: posixOnly }, async () => {
  control.openEditor(toolFile(), 9, 6); // on the definition
  control.setSetting('repocortex.minConfidence', 'high');
  await settle();
  clearLog();
  await control.run('repocortex.showImpact');
  assert.deepEqual(questionCalls().at(-1), { tool: 'analyze_impact', args: { symbol: 'parseToolArg', min_confidence: 'high' } });
  let tree = control.tree(RESULTS);
  assert.equal(tree.message, 'parseToolArg: 2 direct, 2 indirect. Risk low (score 10).');
  const all = tree.rows.map((r) => String(r.item.label));
  assert.ok(all.some((l) => l.startsWith('Truncated: depth limit 3 reached')));
  assert.ok(all.some((l) => l.startsWith('Note: NOT VISIBLE (unresolved_references)')));
  assert.ok(all.includes('Direct: proven call edge to this symbol (2)'));
  assert.ok(all.includes('Indirect: reaches this symbol through other calls (2)'));

  await control.run('repocortex.showTests');
  assert.deepEqual(questionCalls().at(-1), { tool: 'find_tests', args: { symbol: 'parseToolArg', min_confidence: 'high' } });
  tree = control.tree(RESULTS);
  assert.equal(tree.description, '10 rows');

  // A test row has no line: opening it searches the file for the name.
  control.setSetting('repocortex.minConfidence', 'low');
  await settle();
  control.openEditor(toolFile(), 11, 7); // on "Symbol"
  await control.run('repocortex.showImplementations');
  assert.deepEqual(questionCalls().at(-1), {
    tool: 'find_implementations',
    args: { name: 'Symbol', path: 'internal/cli/tool.go', min_confidence: 'low' },
  });
  tree = control.tree(RESULTS);
  assert.ok(tree.rows.some((r) => r.item.label === 'Implements or extends this (5)'));
  assert.ok(tree.rows.filter((r) => r.item.description?.startsWith('medium · ')).length === 5);
  assert.deepEqual(control.messages, []);
});

test('a shared name: the user picks, and a wrong-definition answer is flagged', { skip: posixOnly }, async () => {
  clearLog();
  control.openEditor(toolFile(), 12, 9); // on "New", which this file does not define
  control.answerQuickPick = (items) => items.find((i: { label: string }) => i.label === 'internal/web/web.go:309');
  await control.run('repocortex.showCallers');

  const pick = control.quickPicks.at(-1);
  assert.equal(pick?.items.length, 15);
  assert.equal(pick?.options.title, '15 indexed definitions are named "New".');
  assert.deepEqual(questionCalls().at(-1), { tool: 'find_callers', args: { name: 'New', path: 'internal/web/web.go', min_confidence: 'low' } });

  const tree = control.tree(RESULTS);
  const first = tree.rows[1]?.item;
  assert.match(String(first?.label), /^Check: This answer is for the definition at internal\/capability\/ownership\/ownership\.go:136, NOT the one you asked about \(internal\/web\/web\.go:309\)/u);
  assert.match(first?.tooltip ?? '', /Written by the RepoCortex extension, not by the repocortex tool\./u);

  // Dismissing the pick asks nothing further.
  clearLog();
  control.answerQuickPick = () => undefined;
  await control.run('repocortex.showCallers');
  assert.deepEqual(
    questionCalls().map((c) => c.tool),
    ['find_symbol'],
  );
});

test('go-to-definition outside the project: a namesake in the index is not silently used', { skip: posixOnly }, async () => {
  clearLog();
  const quickPicksBefore = control.quickPicks.length;
  control.openEditor(toolFile(), 8, 20);
  control.definitionProvider = () => [{ uri: { scheme: 'file', fsPath: '/usr/local/go/src/strings/strings.go' }, range: { start: { line: 4, character: 0 } } }];
  control.answerQuickPick = () => undefined;
  await control.run('repocortex.showCallers');
  assert.equal(control.quickPicks.length, quickPicksBefore + 1);
  assert.match(control.quickPicks.at(-1)?.options.title, /The editor resolves "parseToolArg" to \/usr\/local\/go\/src\/strings\/strings\.go which the index does not list/u);
  assert.deepEqual(
    questionCalls().map((c) => c.tool),
    ['find_symbol'],
  );
  control.definitionProvider = () => [];
});

test('a symbol the index does not have: the tool own "not found" is shown, with no failure', { skip: posixOnly }, async () => {
  control.openEditor(toolFile(), 14, 9); // on "Ghost"
  await control.run('repocortex.showCallers');
  const tree = control.tree(RESULTS);
  assert.equal(tree.message, '0 definition(s) named "NoSuchSymbolAnywhere".');
  assert.ok(tree.rows.some((r) => String(r.item.label).startsWith('Note: Not found. The name must match exactly')));
  assert.deepEqual(control.messages, []);
});

test('a tool failure (exit 1, is_error) is reported with the tool message', { skip: posixOnly }, async () => {
  control.openEditor(toolFile(), 13, 9); // on "Boom"
  await control.run('repocortex.showImpact');
  const message = control.messages.at(-1);
  assert.equal(message?.severity, 'error');
  assert.match(message?.text ?? '', /the tool find_callers reported a failure\. unknown confidence "bogus"/u);
  assert.deepEqual(message?.actions, ['Show Report']);
  const tree = control.tree(RESULTS);
  assert.equal(tree.rows.length, 0);
  assert.match(tree.message, /reported a failure/u);
  control.messages.length = 0;
});

test('CodeLens: off by default; when on, one lookup per symbol, cached', { skip: posixOnly }, async () => {
  const provider = control.codeLensProviders[0];
  const document = control.openEditor(toolFile(), 0, 0);
  await control.run('repocortex.refreshStatus');
  const token = { isCancellationRequested: false, onCancellationRequested: () => ({ dispose() {} }) };
  const range = { start: { line: 9, character: 5 } };
  control.symbolProvider = () => [{ name: 'parseToolArg', kind: 11, range, selectionRange: range, children: [] }];

  clearLog();
  assert.deepEqual(await provider.provideCodeLenses(document, token), []);
  assert.deepEqual(calls(), []);

  control.setSetting('repocortex.codeLens.enabled', true);
  await settle();
  clearLog();
  const lenses = await provider.provideCodeLenses(document, token);
  assert.equal(lenses.length, 1);
  assert.equal(lenses[0].range.start.line, 9);
  // Placing lenses starts no process.
  assert.deepEqual(calls(), []);

  const resolved = await provider.resolveCodeLens(lenses[0], token);
  assert.equal(resolved.command.title, 'RepoCortex: 2 callers, 3 call sites');
  assert.equal(resolved.command.command, 'repocortex.showCallers');
  assert.match(resolved.command.tooltip, /32615 references repository-wide are unresolved or ambiguous/u);
  assert.deepEqual(questionCalls(), [
    { tool: 'find_callers', args: { name: 'parseToolArg', path: 'internal/cli/tool.go', min_confidence: 'low' } },
  ]);

  // Asking again, as VS Code does after every edit, starts nothing.
  const again = await provider.provideCodeLenses(document, token);
  await provider.resolveCodeLens(again[0], token);
  await provider.resolveCodeLens(again[0], token);
  assert.equal(questionCalls().length, 1);

  // Clicking the lens asks about that definition.
  clearLog();
  await control.run('repocortex.showCallers', ...resolved.command.arguments);
  assert.deepEqual(
    questionCalls().map((c) => c.tool),
    ['find_symbol', 'find_callers'],
  );
  control.setSetting('repocortex.codeLens.enabled', false);
  await settle();
});

test('no index: a clear message, an offer to build one, and the index command runs in a terminal', { skip: posixOnly }, async () => {
  const marker = path.join(work, 'project', '.stub-no-index');
  fs.writeFileSync(marker, '');
  try {
    control.openEditor(toolFile(), 8, 20);
    await control.run('repocortex.showCallers');
    const message = control.messages.at(-1);
    assert.equal(message?.severity, 'warning');
    assert.match(message?.text ?? '', /this workspace has no index to answer from\. no index at /u);
    assert.deepEqual(message?.actions, ['Index Workspace']);

    await control.run('repocortex.refreshStatus');
    assert.equal(control.statusBarItems[0].text, '$(circle-slash) RepoCortex: no index');
    // With no index the lens provider places nothing and starts nothing.
    control.setSetting('repocortex.codeLens.enabled', true);
    await control.run('repocortex.refreshStatus');
    const token = { isCancellationRequested: false, onCancellationRequested: () => ({ dispose() {} }) };
    assert.deepEqual(await control.codeLensProviders[0].provideCodeLenses(control.activeTextEditor?.document, token), []);
    control.setSetting('repocortex.codeLens.enabled', false);

    control.tasks.length = 0;
    await control.run('repocortex.indexWorkspace');
    const project = path.join(work, 'project');
    assert.deepEqual(
      control.tasks.map((t) => [t.execution.process, t.execution.args, t.execution.options.cwd]),
      [
        [stub, [`--project=${project}`, 'init'], project],
        [stub, [`--project=${project}`, 'index'], project],
        [stub, [`--project=${project}`, 'resolve'], project],
      ],
    );
  } finally {
    fs.rmSync(marker, { force: true });
  }
  control.messages.length = 0;
});

test('the index command stops at the first step that fails', { skip: posixOnly }, async () => {
  control.tasks.length = 0;
  control.taskExitCode = (task) => (task.definition['step'] === 'index' ? 2 : 0);
  await control.run('repocortex.indexWorkspace');
  assert.deepEqual(
    control.tasks.map((t) => t.definition['step']),
    ['index'],
  );
  assert.match(control.messages.at(-1)?.text ?? '', /"repocortex index" exited with status 2\. The later steps were not run\./u);
  control.taskExitCode = () => 0;
  control.messages.length = 0;
});

test('binary not found: a clear message that points at the setting', { skip: posixOnly }, async () => {
  const missing = path.join(work, 'no-such-repocortex');
  control.setSetting('repocortex.path', missing);
  try {
    control.openEditor(toolFile(), 8, 20);
    await control.run('repocortex.showCallers');
    const message = control.messages.find((m) => m.severity === 'error');
    assert.ok(message);
    assert.match(message.text, /the repocortex binary was not found \(tried ".*no-such-repocortex"\)/u);
    assert.match(message.text, /repocortex\.path/u);
    assert.deepEqual(message.actions, ['Open Settings']);
    await control.run('repocortex.refreshStatus');
    assert.equal(control.statusBarItems[0].text, '$(error) RepoCortex: binary not found');

    // The index command does not open a terminal for a binary that is not there.
    control.tasks.length = 0;
    await control.run('repocortex.indexWorkspace');
    assert.equal(control.tasks.length, 0);
  } finally {
    control.setSetting('repocortex.path', stub);
    await settle();
  }
  control.messages.length = 0;
});

test('no symbol under the cursor: says so and starts nothing', { skip: posixOnly }, async () => {
  await settle();
  clearLog();
  const document = control.openEditor(toolFile(), 10, 0); // on "}"
  assert.equal(document.lineAt(10).text, '}');
  await control.run('repocortex.showCallers');
  assert.equal(control.messages.at(-1)?.text, 'RepoCortex: There is no symbol under the cursor.');
  assert.deepEqual(calls(), []);
});
