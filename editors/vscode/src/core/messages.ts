// What the user is told when a question cannot be answered, and the one
// action that would fix it. Kept here, away from the VS Code API, so the
// wording for each failure can be tested.

import { AbortedError, QueueOverflowError } from './cache';
import type { ToolAnswer, ToolOutcome } from './result';
import { EnvelopeError } from './result';
import {
  BinaryNotFoundError,
  BinaryNotRunnableError,
  OutputTooLargeError,
  RunCancelledError,
  RunTimeoutError,
} from './runner';
import { StatusError } from './status';

/** The single follow-up action offered next to a message. */
export type Remedy = 'open-settings' | 'index' | 'show-report' | 'none';

export interface UserMessage {
  readonly text: string;
  readonly severity: 'error' | 'warning' | 'info';
  readonly remedy: Remedy;
}

/** True for errors that mean "the user or the editor stopped waiting". */
export function isCancellation(err: unknown): boolean {
  return err instanceof RunCancelledError || err instanceof AbortedError;
}

/**
 * The message for an error thrown while running the binary, or undefined
 * for a cancellation, which needs no message.
 */
export function messageForError(err: unknown): UserMessage | undefined {
  if (isCancellation(err)) {
    return undefined;
  }
  if (err instanceof BinaryNotFoundError) {
    return {
      text:
        `RepoCortex: the repocortex binary was not found (tried "${err.binary}"). ` +
        'Install RepoCortex, or set "repocortex.path" in your user settings to the full path of the binary.',
      severity: 'error',
      remedy: 'open-settings',
    };
  }
  if (err instanceof BinaryNotRunnableError) {
    return {
      text: `RepoCortex: ${err.message}. Check "repocortex.path" in your user settings.`,
      severity: 'error',
      remedy: 'open-settings',
    };
  }
  if (err instanceof RunTimeoutError) {
    return {
      text: `RepoCortex: ${err.message} Nothing was changed. Try again, or run the same question with "repocortex tool call" in a terminal to see what it is doing.`,
      severity: 'warning',
      remedy: 'none',
    };
  }
  if (err instanceof OutputTooLargeError || err instanceof EnvelopeError || err instanceof StatusError) {
    return {
      text: `RepoCortex: ${err.message} The installed repocortex may not match what this extension expects.`,
      severity: 'error',
      remedy: 'none',
    };
  }
  if (err instanceof QueueOverflowError) {
    return { text: 'RepoCortex: too many lookups were waiting, so this one was dropped.', severity: 'info', remedy: 'none' };
  }
  return {
    text: `RepoCortex: ${err instanceof Error ? err.message : String(err)}`,
    severity: 'error',
    remedy: 'none',
  };
}

/** The message for a run that finished without a usable answer. */
export function messageForOutcome(outcome: Exclude<ToolOutcome, { kind: 'answer' }>): UserMessage {
  switch (outcome.kind) {
    case 'no-index':
      return {
        text: `RepoCortex: this workspace has no index to answer from. ${outcome.message}`,
        severity: 'warning',
        remedy: 'index',
      };
    case 'tool-failure': {
      // A binary from before `path` and `line` selected a definition rejects
      // them as arguments its schema does not have.
      const older = /\bunexpected additional properties\b/iu.test(outcome.message)
        ? ' The installed repocortex may be older than this extension expects; update it.'
        : '';
      return {
        text: `RepoCortex: the tool ${outcome.answer.tool !== '' ? outcome.answer.tool + ' ' : ''}reported a failure. ${outcome.message}${older}`,
        severity: 'error',
        remedy: 'show-report',
      };
    }
    case 'cli-error': {
      const older = /\bno tool named\b|\bunknown (?:command|flag)\b/iu.test(outcome.message)
        ? ' The installed repocortex may be older than this extension expects; update it.'
        : '';
      return {
        text: `RepoCortex: repocortex failed. ${outcome.message}${older}`,
        severity: 'error',
        remedy: 'none',
      };
    }
  }
}

/**
 * The message for a name the tool matched to several definitions and
 * answered for none of. It is not a failure: the tool's summary says what
 * to choose between, and the report lists the candidates.
 */
export function messageForAmbiguous(name: string, answer: ToolAnswer): UserMessage {
  const summary = answer.summary !== '' ? ` ${answer.summary}` : '';
  return {
    text: `RepoCortex: "${name}" matches several definitions, so nothing was analysed.${summary}`,
    severity: 'warning',
    remedy: 'show-report',
  };
}
