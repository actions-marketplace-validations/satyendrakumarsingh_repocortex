// Picks which definition a question is about, and checks afterwards that the
// tool answered for that one.
//
// A name is not a symbol: fifteen functions can be called `New`. The tools
// take a name plus a path hint (find_callers, find_implementations) or a
// fully qualified name (analyze_impact, find_tests). Neither is airtight, so
// when the answer may be for a different definition than the one the user
// pointed at, the extension says so instead of presenting it as the answer.

import type { QuestionKind, QuestionTarget } from './args';
import type { ToolAnswer } from './result';

/** One definition, as find_symbol reports it. */
export interface Definition {
  readonly name: string;
  readonly kind: string;
  /** Repository-relative, forward slashes. */
  readonly path: string;
  /** 1-based. 0 when the tool reported none. */
  readonly line: number;
  readonly fqn?: string;
  readonly confidence?: string;
}

/** The definitions in a find_symbol answer that carry a location. */
export function definitionsFrom(answer: ToolAnswer): Definition[] {
  const out: Definition[] = [];
  for (const item of answer.items) {
    if (item.name === undefined || item.path === undefined) {
      continue;
    }
    const def: { name: string; kind: string; path: string; line: number; fqn?: string; confidence?: string } = {
      name: item.name,
      kind: item.kind,
      path: item.path,
      line: item.line ?? 0,
    };
    if (item.fqn !== undefined) def.fqn = item.fqn;
    if (item.confidence !== undefined) def.confidence = item.confidence;
    out.push(def);
  }
  return out;
}

/**
 * Where the editor believes the symbol is defined.
 *
 * `source` says how much the hint can be trusted. 'definition' comes from the
 * language server's go-to-definition and names the real definition. 'cursor'
 * is only where the cursor happens to be, which is the definition when the
 * user is on one and just a usage site otherwise.
 */
export interface DefinitionHint {
  readonly source: 'definition' | 'cursor' | 'none';
  /** Repository-relative path, forward slashes. Absent when unknown or outside the project. */
  readonly path?: string;
  /** 1-based line. */
  readonly line?: number;
  /** Set when go-to-definition leads outside the project, to a file the index cannot hold. */
  readonly outsideProject?: string;
}

export type Choice =
  /** The index has no definition under that name. */
  | { readonly kind: 'none' }
  /** One definition was chosen without asking. */
  | { readonly kind: 'chosen'; readonly definition: Definition; readonly basis: 'only-one' | 'matches-hint' }
  /**
   * The user must choose. 'ambiguous': several definitions share the name.
   * 'hint-mismatch': the language server points at a definition the index
   * does not have under this name, so every candidate may be a different
   * symbol that merely shares the name.
   */
  | { readonly kind: 'ask'; readonly candidates: readonly Definition[]; readonly reason: 'ambiguous' | 'hint-mismatch' };

/**
 * Chooses among the definitions the index holds under one name.
 *
 * A candidate in the hinted file wins; among several in that file, the one
 * at the hinted line, else the nearest one starting at or above it. When a
 * trusted hint matches no candidate, nothing is chosen silently, even if
 * there is only one candidate: it is probably a different symbol with the
 * same name. With only a cursor position to go on, a single candidate is
 * taken and several are put to the user.
 */
export function chooseDefinition(definitions: readonly Definition[], hint: DefinitionHint): Choice {
  if (definitions.length === 0) {
    return { kind: 'none' };
  }
  if (hint.path !== undefined) {
    const inFile = definitions.filter((d) => d.path === hint.path);
    if (inFile.length === 1) {
      return { kind: 'chosen', definition: inFile[0] as Definition, basis: 'matches-hint' };
    }
    if (inFile.length > 1) {
      if (hint.line !== undefined) {
        const line = hint.line;
        const exact = inFile.find((d) => d.line === line);
        if (exact !== undefined) {
          return { kind: 'chosen', definition: exact, basis: 'matches-hint' };
        }
        const above = inFile.filter((d) => d.line > 0 && d.line <= line).sort((a, b) => b.line - a.line)[0];
        if (above !== undefined) {
          return { kind: 'chosen', definition: above, basis: 'matches-hint' };
        }
      }
      return { kind: 'ask', candidates: inFile, reason: 'ambiguous' };
    }
  }
  if (hint.source === 'definition') {
    return { kind: 'ask', candidates: definitions, reason: 'hint-mismatch' };
  }
  if (definitions.length === 1) {
    return { kind: 'chosen', definition: definitions[0] as Definition, basis: 'only-one' };
  }
  return { kind: 'ask', candidates: definitions, reason: 'ambiguous' };
}

/**
 * The tool arguments that single out `chosen`.
 *
 * The fully qualified name is used only when the bare name is ambiguous:
 * analyze_impact and find_tests refuse an ambiguous bare name, and with a
 * unique one the bare name gives the more readable summary.
 */
export function targetFor(chosen: Definition, all: readonly Definition[]): QuestionTarget {
  const target: { name: string; path?: string; fqn?: string } = { name: chosen.name, path: chosen.path };
  if (all.length > 1 && chosen.fqn !== undefined) {
    target.fqn = chosen.fqn;
  }
  return target;
}

/**
 * The definition a summary says it answered for.
 *
 * find_callers and find_implementations name it in their summary line, as
 * `... of parseToolArg (internal/cli/tool.go:384).` This reads that location
 * back. It returns undefined when the summary carries none, which is the
 * case for analyze_impact and find_tests.
 */
export function answeredLocation(summary: string): { path: string; line: number } | undefined {
  const match = /\(([^()\s:][^()\s]*):(\d+)\)/u.exec(summary);
  if (match === null) {
    return undefined;
  }
  const line = Number.parseInt(match[2] as string, 10);
  if (!Number.isInteger(line) || line < 1) {
    return undefined;
  }
  return { path: match[1] as string, line };
}

/**
 * Warnings, written by the extension, about whether an answer is for the
 * definition the user chose. Empty when there is nothing to warn about.
 */
export function targetWarnings(
  kind: QuestionKind,
  chosen: Definition,
  all: readonly Definition[],
  answer: ToolAnswer,
): string[] {
  const warnings: string[] = [];
  if (all.length <= 1) {
    return warnings;
  }

  const sameFile = all.filter((d) => d.path === chosen.path);
  if (kind === 'callers' || kind === 'implementations') {
    const answered = answeredLocation(answer.summary);
    if (answered !== undefined && (answered.path !== chosen.path || (chosen.line > 0 && answered.line !== chosen.line))) {
      warnings.push(
        `This answer is for the definition at ${answered.path}:${answered.line}, NOT the one you asked about ` +
          `(${chosen.path}:${chosen.line}). ${all.length} definitions are named "${chosen.name}" and the tool's ` +
          'path hint could not single yours out. Do not read this as an answer about your symbol.',
      );
    } else if (answered === undefined) {
      warnings.push(
        `${all.length} definitions are named "${chosen.name}" and the tool did not say which one it answered for. ` +
          `The one asked about is ${chosen.path}:${chosen.line}.`,
      );
    }
  } else if (sameFile.length > 1) {
    warnings.push(
      `${sameFile.length} definitions named "${chosen.name}" are in ${chosen.path}. They share one fully qualified ` +
        'name, so this tool cannot tell them apart: the answer may be for any of them.',
    );
  }
  return warnings;
}

/** A one-line label for a definition in a pick list: `function  internal/cli/tool.go:384`. */
export function describeDefinition(def: Definition): string {
  const where = def.line > 0 ? `${def.path}:${def.line}` : def.path;
  return def.kind !== '' ? `${def.kind}  ${where}` : where;
}
