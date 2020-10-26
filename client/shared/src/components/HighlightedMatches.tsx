import * as React from 'react'

/** Wraps matches of pattern in text with <strong>. */
export const HighlightedMatches: React.FunctionComponent<{
    /* The text to display matches in. */
    text: string

    /* The pattern to highlight in the text. */
    pattern: string

    /** The class name for the <strong> element for matches. */
    className?: string
}> = ({ text, pattern, className }) => (
    <span>
        {fuzzyMatches(text.toLowerCase(), pattern.toLowerCase()).map((span, index) =>
            span.match ? (
                <strong key={index} className={className}>
                    {text.slice(span.start, span.end)}
                </strong>
            ) : (
                text.slice(span.start, span.end)
            )
        )}
    </span>
)

export interface Span {
    start: number
    end: number
    match: boolean
}

export function fuzzyMatches(text: string, pattern: string): Span[] {
    const matches: Span[] = []
    let index = 0
    let last: Span | undefined
    for (let position = 0; position < pattern.length; position++) {
        const textIndex = text.indexOf(pattern.charAt(position), index)
        if (textIndex === -1) {
            break
        }
        if (last?.match && textIndex === index) {
            last.end = textIndex + 1
        } else if ((last && !last.match) || textIndex > index) {
            matches.push({ start: index, end: textIndex, match: false })
            last = undefined
        }
        if (!last) {
            last = { start: textIndex, end: textIndex + 1, match: true }
            matches.push(last)
        }
        index = textIndex + 1
    }
    if (text && index < text.length) {
        matches.push({ start: index, end: text.length, match: false })
    }
    return matches
}
