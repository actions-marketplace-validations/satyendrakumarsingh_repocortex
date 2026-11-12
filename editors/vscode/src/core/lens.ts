// The pure half of the CodeLens: which definitions get a lens, what name to
// ask RepoCortex about, and what a find_callers answer says about one of them.

import type { ToolAnswer } from './result';
import { answeredLocation } from './target';

/** Most lenses placed in one file. Definitions past this get none. */
export const MAX_LENSES_PER_FILE = 60;

/** A definition as the editor's outline reports it. */
export interface OutlineSymbol {
  /** The outline's display name, which may carry a receiver or a parameter list. */
  readonly name: string;
  /** 'function' for functions, methods and constructors; 'type' for classes, interfaces, structs and enums. */
  readonly category: 'function' | 'type' | 'other';
  /** 0-based line of the name. */
  readonly line: number;
  readonly children?: readonly OutlineSymbol[];
}

export interface LensTarget {
  /** The bare name to ask RepoCortex about. */
  readonly name: string;
  /** 0-based line the lens sits on. */
  readonly line: number;
  readonly category: 'function' | 'type';
  /** True when another lens in the same file asks about the same name. */
  readonly sharedName: boolean;
}

/**
 * The bare identifier inside an outline name, or undefined when there is
 * none worth asking about.
 *
 * Language servers decorate names: gopls writes `(*Server).findCallers`,
 * Java writes `handle(Request, int)`, C++ writes `Parser::parse`, and
 * TypeScript writes `<function>` or `describe('x') callback` for anonymous
 * code. The index knows the bare name only.
 */
export function bareSymbolName(outlineName: string): string | undefined {
  let name = outlineName.trim();
  // Go receiver: "(*Server).findCallers" or "(Server).String".
  const receiver = /^\([^)]*\)\.(.+)$/u.exec(name);
  if (receiver !== null) {
    name = receiver[1] as string;
  }
  // Parameter list or generic parameters: "handle(Request, int)", "Box<T>".
  // The brackets must close the name: "describe('x') callback" is not a
  // symbol called describe.
  const head = /^([^(<[\s]+)\s*(?:[(<[].*[)>\]])?$/u.exec(name);
  if (head === null) {
    return undefined;
  }
  name = head[1] as string;
  // Qualified: "Parser::parse", "Outer.inner".
  const last = name.split(/::|\.|#/u).pop();
  name = (last ?? '').trim();
  if (!/^[\p{L}_$][\p{L}\p{N}_$]*$/u.test(name)) {
    return undefined;
  }
  return name;
}

/**
 * The definitions that get a lens: functions and types, at any nesting
 * depth, in document order, capped at `max`.
 */
export function lensTargets(symbols: readonly OutlineSymbol[], max: number = MAX_LENSES_PER_FILE): LensTarget[] {
  const found: { name: string; line: number; category: 'function' | 'type' }[] = [];
  const walk = (nodes: readonly OutlineSymbol[], depth: number): void => {
    for (const node of nodes) {
      if (node.category !== 'other') {
        const name = bareSymbolName(node.name);
        if (name !== undefined) {
          found.push({ name, line: node.line, category: node.category });
        }
      }
      // Functions nested in functions are local helpers and closures; the
      // index rarely has them, so only types are searched for members.
      if (node.children !== undefined && depth < 8 && node.category !== 'function') {
        walk(node.children, depth + 1);
      }
    }
  };
  walk(symbols, 0);
  found.sort((a, b) => a.line - b.line);

  // One lens per line: a second symbol on the same line would stack lenses.
  const byLine = new Map<number, { name: string; line: number; category: 'function' | 'type' }>();
  for (const f of found) {
    if (!byLine.has(f.line)) {
      byLine.set(f.line, f);
    }
  }
  const unique = [...byLine.values()].slice(0, Math.max(0, max));
  const counts = new Map<string, number>();
  for (const u of unique) {
    counts.set(u.name, (counts.get(u.name) ?? 0) + 1);
  }
  return unique.map((u) => ({ ...u, sharedName: (counts.get(u.name) ?? 0) > 1 }));
}

/** What a find_callers answer says about the definition a lens sits on. */
export type LensCount =
  /** The tool answered for a definition in this file. */
  | {
      readonly kind: 'count';
      /** Distinct calling symbols among the rows returned. */
      readonly callers: number;
      /** Rows returned: one per call site. */
      readonly sites: number;
      /** True when the tool cut the list, so both numbers are floors. */
      readonly atLeast: boolean;
    }
  /** The tool answered for a same-named definition in another file. */
  | { readonly kind: 'other-definition'; readonly path: string; readonly line: number }
  /** The index has no definition under this name. */
  | { readonly kind: 'not-indexed' };

/**
 * Reads a find_callers answer for a lens in `filePath` (repository-relative).
 *
 * The check on the answered location matters: the tool's path hint is a
 * preference, not a filter, so for a name shared by many definitions it can
 * answer for one in a different file. A count for another function must not
 * be shown above this one.
 */
export function lensCount(answer: ToolAnswer, filePath: string): LensCount {
  const answered = answeredLocation(answer.summary);
  if (answered === undefined) {
    return { kind: 'not-indexed' };
  }
  if (answered.path !== filePath) {
    return { kind: 'other-definition', path: answered.path, line: answered.line };
  }
  const distinct = new Set<string>();
  for (const item of answer.items) {
    distinct.add(`${item.path ?? ''}\u0000${item.name ?? ''}`);
  }
  return { kind: 'count', callers: distinct.size, sites: answer.items.length, atLeast: answer.truncated };
}

function plural(n: number, one: string, many: string): string {
  return n === 1 ? one : many;
}

/** The lens text. Zero is worded as "found", never as "unused". */
export function lensTitle(count: LensCount, target: Pick<LensTarget, 'sharedName'>): string {
  switch (count.kind) {
    case 'not-indexed':
      return 'RepoCortex: not in the index';
    case 'other-definition':
      return 'RepoCortex: callers unknown (name is shared)';
    case 'count': {
      const plus = count.atLeast ? '+' : '';
      let title: string;
      if (count.sites === 0) {
        title = 'RepoCortex: 0 callers found';
      } else {
        title = `RepoCortex: ${count.callers}${plus} ${plural(count.callers, 'caller', 'callers')}`;
        if (count.sites !== count.callers) {
          title += `, ${count.sites}${plus} call sites`;
        }
      }
      return target.sharedName ? `${title} (name shared in this file)` : title;
    }
  }
}

/** The lens tooltip: the tool's summary and every note it attached. */
export function lensTooltip(count: LensCount, answer: ToolAnswer, minConfidence: string): string {
  const lines: string[] = [];
  if (count.kind === 'other-definition') {
    lines.push(
      `RepoCortex answered for the definition at ${count.path}:${count.line}, not this one. ` +
        'Several definitions share this name and its path hint could not single this one out.',
    );
  }
  if (answer.summary !== '') {
    lines.push(answer.summary);
  }
  if (answer.truncated && answer.truncatedReason !== undefined) {
    lines.push(`Truncated: ${answer.truncatedReason}`);
  }
  lines.push(...answer.notes);
  if (minConfidence !== 'low') {
    lines.push(`Counting results at confidence "${minConfidence}" and above (repocortex.minConfidence).`);
  }
  lines.push('Click to list the callers.');
  return lines.join('\n\n');
}

/** Cache key for one lens lookup. Changes whenever the index is rebuilt. */
export function lensCacheKey(projectRoot: string, token: string, minConfidence: string, filePath: string, name: string): string {
  return [projectRoot, token, minConfidence, filePath, name].join('\u0000');
}
