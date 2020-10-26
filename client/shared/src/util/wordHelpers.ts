// Adapted from
// https://raw.githubusercontent.com/Microsoft/vscode/2937cb0baa742db4094f810e4a59cbaf7ad02db7/src/vs/editor/common/model/wordHelper.ts.
// Copyright (c) Microsoft Corporation. All rights reserved. Licensed under the MIT License.

/**
 * Matches words.
 *
 * @todo It is convenient for this to also match tokens that begin with common completion trigger
 * characters, such as `@` (for username completion). That is not useful in general. When we have a
 * need for this to be stricter, support custom regexps and make this not match `@` and the other
 * non-standard word symbols..
 *
 * Users of this value *must* reset it before using it as follows: `WORD_REGEXP.lastIndex = 0`.
 */
const WORD_REGEXP = /(-?\d*\.\d\w*)|([^\s"'(),;<>?[\\\]`{}]+)/g

/**
 * A word that was found in a model surrounding a position.
 */
export interface WordAtPosition {
    /** The word. */
    readonly word: string

    /** The column where the word starts. */
    readonly startColumn: number

    /** The column where the word ends. */
    readonly endColumn: number
}

/**
 * Finds the word that surrounds a position in text.
 *
 * @param column The column at which to find the word (in {@link text}).
 * @param text A single line of text.
 * {@link WordAtPosition#endColumn}. Use this when {@link text} is a line suffix, not a whole line.
 */
export function getWordAtText(column: number, text: string): WordAtPosition | null {
    WORD_REGEXP.lastIndex = 0
    const match = WORD_REGEXP.exec(text)
    if (!match) {
        return null // no words
    }

    // Find whitespace-enclosed text around column and match from there.
    const start = text.lastIndexOf(' ', column - 1) + 1

    WORD_REGEXP.lastIndex = start
    while (true) {
        const match = WORD_REGEXP.exec(text)
        if (!match) {
            return null
        }
        const matchIndex = match.index || 0
        if (matchIndex <= column && WORD_REGEXP.lastIndex >= column) {
            return {
                word: match[0],
                startColumn: matchIndex,
                endColumn: WORD_REGEXP.lastIndex,
            }
        }
    }
}
