/**
 * Groups highlights that have overlapping or adjacent context. The input must be sorted.
 */
export const mergeContext = <T extends { line: number }>(context: number, highlights: T[]): T[][] => {
    const groupsOfHighlights: T[][] = []

    for (let index = 0; index < highlights.length; index++) {
        const current = highlights[index]
        const previous = highlights[index - 1]
        if (!previous || current.line - previous.line - 2 * context > 1) {
            // Either this is the beginning of the file, or there is at
            // least one line between the end of the previous context
            // and the beginning of this context, so start a new group.
            groupsOfHighlights.push([current])
        } else {
            // This context either overlaps with or is adjacent to the
            // previous context, so add this highlight to the previous
            // group.
            groupsOfHighlights[groupsOfHighlights.length - 1].push(current)
        }
    }

    return groupsOfHighlights
}
