import { editor, Position } from 'monaco-editor'

import { SearchPatternType } from '../../graphql-operations'

import { getHoverResult } from './hover'
import { scanSearchQuery, ScanSuccess, ScanResult } from './scanner'
import { Token } from './token'

/* eslint-disable ban/ban */

/**
 * Needed for mocking methods not implemented in JSDom, for tests dependent on Monaco editor model.
 */
Object.defineProperty(window, 'matchMedia', {
    writable: true,
    value: jest.fn().mockImplementation(query => ({
        matches: false,
        media: query,
        onchange: null,
        addListener: jest.fn(), // Deprecated
        removeListener: jest.fn(), // Deprecated
        addEventListener: jest.fn(),
        removeEventListener: jest.fn(),
        dispatchEvent: jest.fn(),
    })),
})

expect.addSnapshotSerializer({
    serialize: value => JSON.stringify(value, null, 2),
    test: () => true,
})

const toSuccess = (result: ScanResult<Token[]>): Token[] => (result as ScanSuccess<Token[]>).term

describe('getHoverResult()', () => {
    test('returns hover contents for filters', () => {
        const input = 'repo:sourcegraph file:code_intelligence'
        const scannedQuery = toSuccess(scanSearchQuery(input))
        expect(getHoverResult(scannedQuery, new Position(1, 4), editor.createModel(input))).toMatchInlineSnapshot(`
            {
              "contents": [
                {
                  "value": "Include only results from repositories matching the given search pattern."
                }
              ],
              "range": {
                "startLineNumber": 1,
                "endLineNumber": 1,
                "startColumn": 1,
                "endColumn": 6
              }
            }
        `)
        expect(getHoverResult(scannedQuery, new Position(1, 18), editor.createModel(input))).toMatchInlineSnapshot(`
            {
              "contents": [
                {
                  "value": "Include only results from files matching the given search pattern."
                }
              ],
              "range": {
                "startLineNumber": 1,
                "endLineNumber": 1,
                "startColumn": 18,
                "endColumn": 23
              }
            }
        `)
        expect(getHoverResult(scannedQuery, new Position(1, 30), editor.createModel(input))).toMatchInlineSnapshot(`
            {
              "contents": [
                {
                  "value": "Matches the string \`code_intelligence\`."
                }
              ],
              "range": {
                "startLineNumber": 1,
                "endLineNumber": 1,
                "startColumn": 23,
                "endColumn": 40
              }
            }
        `)
    })

    test('returns hover contents for fields and regexp values', () => {
        const input = 'repo:^hey$'
        const scannedQuery = toSuccess(scanSearchQuery(input))
        expect(getHoverResult(scannedQuery, new Position(1, 3), editor.createModel(input))).toMatchInlineSnapshot(`
            {
              "contents": [
                {
                  "value": "Include only results from repositories matching the given search pattern."
                }
              ],
              "range": {
                "startLineNumber": 1,
                "endLineNumber": 1,
                "startColumn": 1,
                "endColumn": 6
              }
            }
        `)
        expect(getHoverResult(scannedQuery, new Position(1, 6), editor.createModel(input))).toMatchInlineSnapshot(`
            {
              "contents": [
                {
                  "value": "**Start anchor**. Match the beginning of a string. Typically used to match a string prefix, as in \`^prefix\`. Also often used with the end anchor \`$\` to match an exact string, as in \`^exact$\`."
                }
              ],
              "range": {
                "startLineNumber": 1,
                "endLineNumber": 1,
                "startColumn": 6,
                "endColumn": 7
              }
            }
        `)
    })

    test('returns hover contents regexp patterns', () => {
        const input = '\\b.*?'
        const scannedQuery = toSuccess(scanSearchQuery(input, false, SearchPatternType.regexp))
        expect(getHoverResult(scannedQuery, new Position(1, 1), editor.createModel(input))).toMatchInlineSnapshot(`
            {
              "contents": [
                {
                  "value": "**Word boundary**. Match a position where a word character comes after a non-word character, or vice versa. Typically used to match whole words, as in \`\\\\bword\\\\b\`."
                }
              ],
              "range": {
                "startLineNumber": 1,
                "endLineNumber": 1,
                "startColumn": 1,
                "endColumn": 3
              }
            }
        `)
        expect(getHoverResult(scannedQuery, new Position(1, 2), editor.createModel(input))).toMatchInlineSnapshot(`
            {
              "contents": [
                {
                  "value": "**Word boundary**. Match a position where a word character comes after a non-word character, or vice versa. Typically used to match whole words, as in \`\\\\bword\\\\b\`."
                }
              ],
              "range": {
                "startLineNumber": 1,
                "endLineNumber": 1,
                "startColumn": 1,
                "endColumn": 3
              }
            }
        `)
        expect(getHoverResult(scannedQuery, new Position(1, 3), editor.createModel(input))).toMatchInlineSnapshot(`
            {
              "contents": [
                {
                  "value": "**Dot**. Match any character except a line break."
                }
              ],
              "range": {
                "startLineNumber": 1,
                "endLineNumber": 1,
                "startColumn": 3,
                "endColumn": 4
              }
            }
        `)
        expect(getHoverResult(scannedQuery, new Position(1, 4), editor.createModel(input))).toMatchInlineSnapshot(`
            {
              "contents": [
                {
                  "value": "**Zero or more**. Match zero or more of the previous expression."
                }
              ],
              "range": {
                "startLineNumber": 1,
                "endLineNumber": 1,
                "startColumn": 4,
                "endColumn": 5
              }
            }
        `)
        expect(getHoverResult(scannedQuery, new Position(1, 5), editor.createModel(input))).toMatchInlineSnapshot(`
            {
              "contents": [
                {
                  "value": "**Lazy**. Match as few as characters as possible that match the previous expression."
                }
              ],
              "range": {
                "startLineNumber": 1,
                "endLineNumber": 1,
                "startColumn": 5,
                "endColumn": 6
              }
            }
        `)
    })

    test('regexp group range encloses pattern', () => {
        const input = '(abcd){1,3}'
        const scannedQuery = toSuccess(scanSearchQuery(input, false, SearchPatternType.regexp))
        expect(getHoverResult(scannedQuery, new Position(1, 1), editor.createModel(input))).toMatchInlineSnapshot(`
            {
              "contents": [
                {
                  "value": "**Group**. Groups together multiple expressions to match."
                }
              ],
              "range": {
                "startLineNumber": 1,
                "endLineNumber": 1,
                "startColumn": 1,
                "endColumn": 7
              }
            }
        `)
        expect(getHoverResult(scannedQuery, new Position(1, 2), editor.createModel(input))).toMatchInlineSnapshot(`
            {
              "contents": [
                {
                  "value": "Matches the string \`abcd\`."
                }
              ],
              "range": {
                "startLineNumber": 1,
                "endLineNumber": 1,
                "startColumn": 2,
                "endColumn": 6
              }
            }
        `)
        expect(getHoverResult(scannedQuery, new Position(1, 8), editor.createModel(input))).toMatchInlineSnapshot(`
            {
              "contents": [
                {
                  "value": "**Range**. Match between 1 and 3 of the previous expression."
                }
              ],
              "range": {
                "startLineNumber": 1,
                "endLineNumber": 1,
                "startColumn": 7,
                "endColumn": 12
              }
            }
        `)
    })

    test('regexp escape characters', () => {
        const input = '\\q\\r\\n\\.\\\\'
        const scannedQuery = toSuccess(scanSearchQuery(input, false, SearchPatternType.regexp))
        expect(getHoverResult(scannedQuery, new Position(1, 1), editor.createModel(input))).toMatchInlineSnapshot(`
            {
              "contents": [
                {
                  "value": "**Escaped Character**. The character \`q\` is escaped."
                }
              ],
              "range": {
                "startLineNumber": 1,
                "endLineNumber": 1,
                "startColumn": 1,
                "endColumn": 3
              }
            }
        `)
        expect(getHoverResult(scannedQuery, new Position(1, 3), editor.createModel(input))).toMatchInlineSnapshot(`
            {
              "contents": [
                {
                  "value": "**Escaped Character**. Match a carriage return."
                }
              ],
              "range": {
                "startLineNumber": 1,
                "endLineNumber": 1,
                "startColumn": 3,
                "endColumn": 5
              }
            }
        `)
        expect(getHoverResult(scannedQuery, new Position(1, 5), editor.createModel(input))).toMatchInlineSnapshot(`
            {
              "contents": [
                {
                  "value": "**Escaped Character**. Match a new line."
                }
              ],
              "range": {
                "startLineNumber": 1,
                "endLineNumber": 1,
                "startColumn": 5,
                "endColumn": 7
              }
            }
        `)
        expect(getHoverResult(scannedQuery, new Position(1, 7), editor.createModel(input))).toMatchInlineSnapshot(`
            {
              "contents": [
                {
                  "value": "**Escaped Character**. Match the character \`.\`."
                }
              ],
              "range": {
                "startLineNumber": 1,
                "endLineNumber": 1,
                "startColumn": 7,
                "endColumn": 9
              }
            }
        `)
        expect(getHoverResult(scannedQuery, new Position(1, 9), editor.createModel(input))).toMatchInlineSnapshot(`
            {
              "contents": [
                {
                  "value": "**Escaped Character**. Match the character \`\\\\\`."
                }
              ],
              "range": {
                "startLineNumber": 1,
                "endLineNumber": 1,
                "startColumn": 9,
                "endColumn": 11
              }
            }
        `)
    })

    test('ordinary and negated character class', () => {
        const input = '[^a-z][0-9]'
        const scannedQuery = toSuccess(scanSearchQuery(input, false, SearchPatternType.regexp))
        expect(getHoverResult(scannedQuery, new Position(1, 2), editor.createModel(input))).toMatchInlineSnapshot(`
            {
              "contents": [
                {
                  "value": "**Negated character class**. Match any character _not_ inside the square brackets."
                }
              ],
              "range": {
                "startLineNumber": 1,
                "endLineNumber": 1,
                "startColumn": 1,
                "endColumn": 7
              }
            }
        `)

        expect(getHoverResult(scannedQuery, new Position(1, 7), editor.createModel(input))).toMatchInlineSnapshot(`
            {
              "contents": [
                {
                  "value": "**Character class**. Match any character inside the square brackets."
                }
              ],
              "range": {
                "startLineNumber": 1,
                "endLineNumber": 1,
                "startColumn": 7,
                "endColumn": 12
              }
            }
        `)
    })

    test('literal search interprets parentheses as patterns', () => {
        const input = '(abcd)'
        const scannedQuery = toSuccess(scanSearchQuery(input, false, SearchPatternType.literal))
        expect(getHoverResult(scannedQuery, new Position(1, 1), editor.createModel(input))).toMatchInlineSnapshot(`
            {
              "contents": [
                {
                  "value": "Matches the string \`(abcd)\`."
                }
              ],
              "range": {
                "startLineNumber": 1,
                "endLineNumber": 1,
                "startColumn": 1,
                "endColumn": 7
              }
            }
        `)
        expect(getHoverResult(scannedQuery, new Position(1, 2), editor.createModel(input))).toMatchInlineSnapshot(`
            {
              "contents": [
                {
                  "value": "Matches the string \`(abcd)\`."
                }
              ],
              "range": {
                "startLineNumber": 1,
                "endLineNumber": 1,
                "startColumn": 1,
                "endColumn": 7
              }
            }
        `)
    })

    test('returns hover contents for revision syntax', () => {
        const input = 'repo:^foo$@head:v1.3 rev:*refs/heads/*:*!refs/heads/release*'
        const scannedQuery = toSuccess(scanSearchQuery(input, false, SearchPatternType.literal))

        expect(getHoverResult(scannedQuery, new Position(1, 11), editor.createModel(input))).toMatchInlineSnapshot(`
            {
              "contents": [
                {
                  "value": "**Search at revision**. Separates a repository pattern and the revisions to search, like commits or branches. The part before the \`@\` specifies the repositories to search, the part after the \`@\` specifies which revisions to search."
                }
              ],
              "range": {
                "startLineNumber": 1,
                "endLineNumber": 1,
                "startColumn": 11,
                "endColumn": 12
              }
            }
        `)

        expect(getHoverResult(scannedQuery, new Position(1, 12), editor.createModel(input))).toMatchInlineSnapshot(`
            {
              "contents": [
                {
                  "value": "**Revision HEAD**. Search the repository at the latest HEAD commit of the default branch."
                }
              ],
              "range": {
                "startLineNumber": 1,
                "endLineNumber": 1,
                "startColumn": 12,
                "endColumn": 16
              }
            }
        `)

        expect(getHoverResult(scannedQuery, new Position(1, 16), editor.createModel(input))).toMatchInlineSnapshot(`
            {
              "contents": [
                {
                  "value": "**Revision separator**. Separates multiple revisions to search across. For example, \`1a35d48:feature:3.15\` searches the repository for matches at commit \`1a35d48\`, or a branch named \`feature\`, or a tag \`3.15\`."
                }
              ],
              "range": {
                "startLineNumber": 1,
                "endLineNumber": 1,
                "startColumn": 16,
                "endColumn": 17
              }
            }
        `)

        expect(getHoverResult(scannedQuery, new Position(1, 17), editor.createModel(input))).toMatchInlineSnapshot(`
            {
              "contents": [
                {
                  "value": "**Revision branch name or tag**. Search the branch name or tag at the head commit."
                }
              ],
              "range": {
                "startLineNumber": 1,
                "endLineNumber": 1,
                "startColumn": 17,
                "endColumn": 21
              }
            }
        `)

        expect(getHoverResult(scannedQuery, new Position(1, 26), editor.createModel(input))).toMatchInlineSnapshot(`
            {
              "contents": [
                {
                  "value": "**Revision glob pattern to include**. A prefixing indicating that a glob pattern follows. Git references matching the glob pattern are included in the search. Typically used where a pattern like \`*refs/heads/*\` searches across all repository branches at the head commit."
                }
              ],
              "range": {
                "startLineNumber": 1,
                "endLineNumber": 1,
                "startColumn": 26,
                "endColumn": 27
              }
            }
        `)

        expect(getHoverResult(scannedQuery, new Position(1, 27), editor.createModel(input))).toMatchInlineSnapshot(`
            {
              "contents": [
                {
                  "value": "**Revision using git reference path**. Search the branch name or tag at the head commit. Search across git objects, like commits or branches, that match this git reference path. Typically used in conjunction with glob patterns, where a pattern like \`*refs/heads/*\` searches across all repository branches at the head commit."
                }
              ],
              "range": {
                "startLineNumber": 1,
                "endLineNumber": 1,
                "startColumn": 27,
                "endColumn": 38
              }
            }
        `)

        expect(getHoverResult(scannedQuery, new Position(1, 41), editor.createModel(input))).toMatchInlineSnapshot(`
            {
              "contents": [
                {
                  "value": "**Revision glob pattern to exclude**. A prefix indicating that git references, like a commit or branch name, should be **excluded** from search based on the glob pattern that follows. Used in conjunction with a glob pattern that matches a set of commits or branches, followed by a a pattern to exclude from the set. For example, \`*refs/heads/*:*!refs/heads/release*\` searches all branches at the head commit, excluding branches matching \`release*\`."
                }
              ],
              "range": {
                "startLineNumber": 1,
                "endLineNumber": 1,
                "startColumn": 40,
                "endColumn": 42
              }
            }
        `)
    })

    test('returns hover contents for structural syntax', () => {
        const input = ':[var~\\w+] ...'
        const scannedQuery = toSuccess(scanSearchQuery(input, false, SearchPatternType.structural))

        expect(getHoverResult(scannedQuery, new Position(1, 1), editor.createModel(input))).toMatchInlineSnapshot(`
            {
              "contents": [
                {
                  "value": "**Regular expression hole**. Match the regular expression defined inside this hole."
                }
              ],
              "range": {
                "startLineNumber": 1,
                "endLineNumber": 1,
                "startColumn": 1,
                "endColumn": 11
              }
            }
        `)

        expect(getHoverResult(scannedQuery, new Position(1, 3), editor.createModel(input))).toMatchInlineSnapshot(`
            {
              "contents": [
                {
                  "value": "**Hole variable**. A descriptive name for the syntax matched by this hole."
                }
              ],
              "range": {
                "startLineNumber": 1,
                "endLineNumber": 1,
                "startColumn": 3,
                "endColumn": 6
              }
            }
        `)

        expect(getHoverResult(scannedQuery, new Position(1, 6), editor.createModel(input))).toMatchInlineSnapshot(`
            {
              "contents": [
                {
                  "value": "**Regular expression separator**. Indicates the start of a regular expression that this hole should match."
                }
              ],
              "range": {
                "startLineNumber": 1,
                "endLineNumber": 1,
                "startColumn": 6,
                "endColumn": 7
              }
            }
        `)

        expect(getHoverResult(scannedQuery, new Position(1, 9), editor.createModel(input))).toMatchInlineSnapshot(`
            {
              "contents": [
                {
                  "value": "**One or more**. Match one or more of the previous expression."
                }
              ],
              "range": {
                "startLineNumber": 1,
                "endLineNumber": 1,
                "startColumn": 9,
                "endColumn": 10
              }
            }
        `)
        expect(getHoverResult(scannedQuery, new Position(1, 10), editor.createModel(input))).toMatchInlineSnapshot(`
            {
              "contents": [
                {
                  "value": "**Regular expression hole**. Match the regular expression defined inside this hole."
                }
              ],
              "range": {
                "startLineNumber": 1,
                "endLineNumber": 1,
                "startColumn": 1,
                "endColumn": 11
              }
            }
        `)

        expect(getHoverResult(scannedQuery, new Position(1, 12), editor.createModel(input))).toMatchInlineSnapshot(`
            {
              "contents": [
                {
                  "value": "**Structural hole**. Matches code structures contextually. See the [syntax reference](https://docs.sourcegraph.com/code_search/reference/structural#syntax-reference) for a complete description."
                }
              ],
              "range": {
                "startLineNumber": 1,
                "endLineNumber": 1,
                "startColumn": 12,
                "endColumn": 15
              }
            }
        `)
    })
})

test('returns hover contents for select', () => {
    const input = 'select:repo repo:foo'
    const scannedQuery = toSuccess(scanSearchQuery(input, false, SearchPatternType.literal))

    expect(getHoverResult(scannedQuery, new Position(1, 8), editor.createModel(input))).toMatchInlineSnapshot(`
        {
          "contents": [
            {
              "value": "Select and display distinct repository paths from search results."
            }
          ],
          "range": {
            "startLineNumber": 1,
            "endLineNumber": 1,
            "startColumn": 8,
            "endColumn": 12
          }
        }
    `)
})

test('returns repo:contains hovers', () => {
    const input = 'repo:contains.file(foo)'
    const scannedQuery = toSuccess(scanSearchQuery(input, false, SearchPatternType.literal))

    expect(getHoverResult(scannedQuery, new Position(1, 8), editor.createModel(input))).toMatchInlineSnapshot(`
        {
          "contents": [
            {
              "value": "**Built-in predicate**. Search only inside repositories that contain a **file path** matching the regular expression \`foo\`."
            }
          ],
          "range": {
            "startLineNumber": 1,
            "endLineNumber": 1,
            "startColumn": 6,
            "endColumn": 24
          }
        }
    `)
})

test('returns multiline hovers', () => {
    const input = `repo:contains(
      file:foo
      content:bar
)`
    const scannedQuery = toSuccess(scanSearchQuery(input, false, SearchPatternType.literal))

    expect(getHoverResult(scannedQuery, new Position(4, 1), editor.createModel(input))).toMatchInlineSnapshot(`
        {
          "contents": [
            {
              "value": "**Built-in predicate**. Search only inside repositories that satisfy the specified \`file:\` and \`content:\` filters. \`file:\` and \`content:\` filters should be regular expressions."
            }
          ],
          "range": {
            "startLineNumber": 1,
            "endLineNumber": 4,
            "startColumn": 6,
            "endColumn": 2
          }
        }
    `)
})
