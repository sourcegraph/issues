/**
 * Returns the sum of the number of matches of the patterns in the string.
 *
 * @param patterns Patterns to match in the string.
 */
export function count(string: string, ...patterns: RegExp[]): number {
    let count = 0
    for (const pattern of patterns) {
        if (!pattern.global) {
            throw new Error('expected RegExp to be global (or else count is inaccurate)')
        }
        const match = string.match(pattern)
        if (match) {
            count += match.length
        }
    }
    return count
}

export function numberWithCommas(number: string | number): string {
    return number.toString().replace(/\B(?=(\d{3})+(?!\d))/g, ',')
}

export function pluralize(string: string, count: number, plural = string + 's'): string {
    return count === 1 ? string : plural
}

/**
 * Replaces all non alphabetic characters with `-` and lowercases the result.
 */
export function sanitizeClass(value: string): string {
    return value.replace(/[^A-Za-z]/g, '-').toLowerCase()
}

/**
 * In the given string, deduplicate whitespace.
 * E.g: " a  b  c  " => " a b c "
 */
export function dedupeWhitespace(value: string): string {
    return value.replace(/\s+/g, ' ')
}

/**
 * Checkes whether a given string is quoted.
 *
 * @param value string to check against
 */
export function isQuoted(value: string): boolean {
    return value.startsWith('"') && value.endsWith('"') && value !== '"'
}

/**
 * Replaces a substring within a string.
 *
 * @param string Original string
 * @param range The range in of the substring to be replaced
 * @param replacement an optional replacement string
 */
export function replaceRange(string: string, { start, end }: { start: number; end: number }, replacement = ''): string {
    return string.slice(0, start) + replacement + string.slice(end)
}
