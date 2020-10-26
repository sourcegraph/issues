import { IRange } from 'monaco-editor'
import { filterTypeKeysWithAliases } from '../interactive/util'

/**
 * Represents a zero-indexed character range in a single-line search query.
 */
export interface CharacterRange {
    /** Zero-based character on the line */
    start: number
    /** Zero-based character on the line */
    end: number
}

/**
 * Converts a zero-indexed, single-line {@link CharacterRange} to a Monaco {@link IRange}.
 */
export const toMonacoRange = ({ start, end }: CharacterRange): IRange => ({
    startLineNumber: 1,
    endLineNumber: 1,
    startColumn: start + 1,
    endColumn: end + 1,
})

/**
 * Represents a literal in a search query.
 *
 * Example: `Conn`.
 */
export interface Literal {
    type: 'literal'
    value: string
}

/**
 * Represents a filter in a search query.
 *
 * Example: `repo:^github\.com\/sourcegraph\/sourcegraph$`.
 */
export interface Filter {
    type: 'filter'
    filterType: Pick<ParseSuccess<Literal>, 'range' | 'token'>
    filterValue: Pick<ParseSuccess<Literal | Quoted>, 'range' | 'token'> | undefined
}

/**
 * Represents an operator in a search query.
 *
 * Example: AND, OR, NOT.
 */
export interface Operator {
    type: 'operator'
    value: string
}

/**
 * Represents a sequence of tokens in a search query.
 */
export interface Sequence {
    type: 'sequence'
    members: Pick<ParseSuccess<Exclude<Token, Sequence>>, 'range' | 'token'>[]
}

/**
 * Represents a quoted string in a search query.
 *
 * Example: "Conn".
 */
export interface Quoted {
    type: 'quoted'
    quotedValue: string
}

/**
 * Represents a C-style comment, terminated by a newline.
 *
 * Example: `// Oh hai`
 */
export interface Comment {
    type: 'comment'
    value: string
}

export type Token =
    | { type: 'whitespace' }
    | { type: 'openingParen' }
    | { type: 'closingParen' }
    | { type: 'operator' }
    | Comment
    | Literal
    | Filter
    | Sequence
    | Quoted

/**
 * Represents the failed result of running a {@link Parser} on a search query.
 */
interface ParseError {
    type: 'error'

    /**
     * A string representing the token that would have been expected
     * for successful parsing at {@link ParseError#at}.
     */
    expected: string

    /**
     * The index in the search query string where parsing failed.
     */
    at: number
}

/**
 * Represents the successful result of running a {@link Parser} on a search query.
 */
export interface ParseSuccess<T = Token> {
    type: 'success'

    /**
     * The parsed token.
     */
    token: T

    /**
     * The character range that was successfully parsed.
     */
    range: CharacterRange
}

/**
 * Represents the result of running a {@link Parser} on a search query.
 */
export type ParserResult<T = Token> = ParseError | ParseSuccess<T>

type Parser<T = Token> = (input: string, start: number) => ParserResult<T>

/**
 * Returns a {@link Parser} that succeeds if zero or more tokens parsed
 * by the given `parseToken` parsers are found in a search query.
 */
const zeroOrMore = (parseToken: Parser): Parser<Sequence> => (input, start) => {
    const members: Pick<ParseSuccess<Exclude<Token, Sequence>>, 'range' | 'token'>[] = []
    let adjustedStart = start
    let end = start + 1
    while (input[adjustedStart] !== undefined) {
        const result = parseToken(input, adjustedStart)
        if (result.type === 'error') {
            return result
        }
        if (result.token.type === 'sequence') {
            for (const member of result.token.members) {
                members.push(member)
            }
        } else {
            const { range, token } = result
            members.push({ range, token })
        }
        end = result.range.end
        adjustedStart = end
    }
    return {
        type: 'success',
        range: { start, end },
        token: { type: 'sequence', members },
    }
}

/**
 * Returns a {@link Parser} that succeeds if any of the given parsers succeeds.
 */
const oneOf = <T>(...parsers: Parser<T>[]): Parser<T> => (input, start) => {
    const expected: string[] = []
    for (const parser of parsers) {
        const result = parser(input, start)
        if (result.type === 'success') {
            return result
        }
        expected.push(result.expected)
    }
    return {
        type: 'error',
        expected: `One of: ${expected.join(', ')}`,
        at: start,
    }
}

/**
 * A {@link Parser} that will attempt to parse quoted strings in a search query.
 */
const quoted: Parser<Quoted> = (input, start) => {
    if (input[start] !== '"') {
        return { type: 'error', expected: '"', at: start }
    }
    let end = start + 1
    while (input[end] && (input[end] !== '"' || input[end - 1] === '\\')) {
        end = end + 1
    }
    if (!input[end]) {
        return { type: 'error', expected: '"', at: end }
    }
    return {
        type: 'success',
        // end + 1 as `end` is currently the index of the quote in the string.
        range: { start, end: end + 1 },
        token: { type: 'quoted', quotedValue: input.slice(start + 1, end) },
    }
}

/**
 * Returns a {@link Parser} that will attempt to parse tokens matching
 * the given character in a search query.
 */
const character = (character: string): Parser<Literal> => (input, start) => {
    if (input[start] !== character) {
        return { type: 'error', expected: character, at: start }
    }
    return {
        type: 'success',
        range: { start, end: start + 1 },
        token: { type: 'literal', value: character },
    }
}

/**
 * Returns a {@link Parser} that will attempt to parse
 * tokens matching the given RegExp pattern in a search query.
 */
const pattern = <T extends Token = Literal>(
    regexp: RegExp,
    output?: T | ((input: string, range: CharacterRange) => T),
    expected?: string
): Parser<T> => {
    if (!regexp.source.startsWith('^')) {
        regexp = new RegExp(`^${regexp.source}`, regexp.flags)
    }
    return (input, start) => {
        const matchTarget = input.slice(Math.max(0, start))
        if (!matchTarget) {
            return { type: 'error', expected: expected || `/${regexp.source}/`, at: start }
        }
        const match = matchTarget.match(regexp)
        if (!match) {
            return { type: 'error', expected: expected || `/${regexp.source}/`, at: start }
        }
        const range = { start, end: start + match[0].length }
        return {
            type: 'success',
            range,
            token: output
                ? typeof output === 'function'
                    ? output(input, range)
                    : output
                : ({ type: 'literal', value: match[0] } as T),
        }
    }
}

const whitespace = pattern(/\s+/, { type: 'whitespace' as const }, 'whitespace')

const literal = pattern(/[^\s)]+/)

const operator = pattern(
    /(and|AND|or|OR|not|NOT)/,
    (input, { start, end }): Operator => ({ type: 'operator', value: input.slice(start, end) })
)

const comment = pattern(
    /\/\/.*/,
    (input, { start, end }): Comment => ({ type: 'comment', value: input.slice(start, end) })
)

const filterKeyword = pattern(new RegExp(`-?(${filterTypeKeysWithAliases.join('|')})+(?=:)`, 'i'))

const filterDelimiter = character(':')

const filterValue = oneOf<Quoted | Literal>(quoted, literal)

const openingParen = pattern(/\(/, { type: 'openingParen' as const })

const closingParen = pattern(/\)/, { type: 'closingParen' as const })

/**
 * Returns a {@link Parser} that succeeds if a token parsed by `parseToken`,
 * followed by whitespace or EOF, is found in the search query.
 */
const followedBy = (
    parseToken: Parser<Exclude<Token, Sequence>>,
    parseNext: Parser<Exclude<Token, Sequence>>
): Parser<Sequence> => (input, start) => {
    const members: Pick<ParseSuccess<Exclude<Token, Sequence>>, 'range' | 'token'>[] = []
    const tokenResult = parseToken(input, start)
    if (tokenResult.type === 'error') {
        return tokenResult
    }
    members.push({ token: tokenResult.token, range: tokenResult.range })
    let { end } = tokenResult.range
    if (input[end] !== undefined) {
        const separatorResult = parseNext(input, end)
        if (separatorResult.type === 'error') {
            return separatorResult
        }
        members.push({ token: separatorResult.token, range: separatorResult.range })
        end = separatorResult.range.end
    }
    return {
        type: 'success',
        range: { start, end },
        token: { type: 'sequence', members },
    }
}

/**
 * A {@link Parser} that will attempt to parse {@link Filter} tokens
 * (consisting a of a filter type and a filter value, separated by a colon)
 * in a search query.
 */
const filter: Parser<Filter> = (input, start) => {
    const parsedKeyword = filterKeyword(input, start)
    if (parsedKeyword.type === 'error') {
        return parsedKeyword
    }
    const parsedDelimiter = filterDelimiter(input, parsedKeyword.range.end)
    if (parsedDelimiter.type === 'error') {
        return parsedDelimiter
    }
    const parsedValue =
        input[parsedDelimiter.range.end] === undefined ? undefined : filterValue(input, parsedDelimiter.range.end)
    if (parsedValue && parsedValue.type === 'error') {
        return parsedValue
    }
    return {
        type: 'success',
        range: { start, end: parsedValue ? parsedValue.range.end : parsedDelimiter.range.end },
        token: {
            type: 'filter',
            filterType: parsedKeyword,
            filterValue: parsedValue,
        },
    }
}

const baseTerms = [operator, filter, quoted, literal]

const createParser = (terms: Parser<Exclude<Token, Sequence>>[]): Parser<Sequence> =>
    zeroOrMore(
        oneOf<Token>(
            whitespace,
            openingParen,
            closingParen,
            ...terms.map(token =>
                followedBy(token, oneOf<{ type: 'whitespace' } | { type: 'closingParen' }>(whitespace, closingParen))
            )
        )
    )

/**
 * A {@link Parser} for a Sourcegraph search query.
 */
const searchQuery = createParser(baseTerms)

/**
 * A {@link Parser} for a Sourcegraph search query containing comments.
 */
const searchQueryWithComments = createParser([comment, ...baseTerms])

/**
 * Parses a search query string.
 */
export const parseSearchQuery = (query: string, interpretComments?: boolean): ParserResult<Sequence> =>
    interpretComments ? searchQueryWithComments(query, 0) : searchQuery(query, 0)
