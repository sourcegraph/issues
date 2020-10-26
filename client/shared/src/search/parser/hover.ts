import * as Monaco from 'monaco-editor'
import { Sequence, toMonacoRange } from './parser'
import { resolveFilter } from './filters'

/**
 * Returns the hover result for a hovered search token in the Monaco query input.
 */
export const getHoverResult = (
    { members }: Pick<Sequence, 'members'>,
    { column }: Pick<Monaco.Position, 'column'>
): Monaco.languages.Hover | null => {
    const tokenAtColumn = members.find(({ range }) => range.start + 1 <= column && range.end + 1 >= column)
    if (!tokenAtColumn || tokenAtColumn.token.type !== 'filter') {
        return null
    }
    const { filterType } = tokenAtColumn.token
    const resolvedFilter = resolveFilter(filterType.token.value)
    if (!resolvedFilter) {
        return null
    }
    return {
        contents: [
            {
                value:
                    'negated' in resolvedFilter
                        ? resolvedFilter.definition.description(resolvedFilter.negated)
                        : resolvedFilter.definition.description,
            },
        ],
        range: toMonacoRange(tokenAtColumn.range),
    }
}
