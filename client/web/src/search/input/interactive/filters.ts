import { Suggestion } from '../Suggestion'
import { assign } from 'lodash/fp'
import { FilterType } from '../../../../../shared/src/search/interactive/util'
import { resolveFilter } from '../../../../../shared/src/search/parser/filters'

/** FilterTypes which have a finite number of valid options. */
export type FiniteFilterType =
    | FilterType.archived
    | FilterType.fork
    | FilterType.type
    | FilterType.index
    | FilterType.visibility
    | FilterType.stable

const withBooleanSuggestions = (type: FiniteFilterType): Suggestion[] =>
    ['no', 'only', 'yes'].map(value => ({ type, value }))

export const finiteFilters: Record<
    FiniteFilterType,
    {
        default: string
        values: Suggestion[]
    }
> = {
    archived: {
        default: 'yes',
        values: withBooleanSuggestions(FilterType.archived),
    },
    fork: {
        default: 'yes',
        values: withBooleanSuggestions(FilterType.fork),
    },
    index: {
        default: 'yes',
        values: withBooleanSuggestions(FilterType.index),
    },
    type: {
        default: '',
        values: [
            { displayValue: 'code', value: '' },
            { value: 'commit' },
            { value: 'diff' },
            { value: 'repo' },
            { value: 'path' },
            { value: 'symbols' },
        ].map(
            assign({
                type: FilterType.type,
            })
        ),
    },
    visibility: {
        default: 'any',
        values: [{ value: 'any' }, { value: 'public' }, { value: 'private' }].map(
            assign({
                type: FilterType.visibility,
            })
        ),
    },
    stable: {
        default: 'no',
        values: [{ value: 'yes' }, { value: 'no' }].map(
            assign({
                type: FilterType.stable,
            })
        ),
    },
}

export const isFiniteFilter = (filter: FilterType): filter is FiniteFilterType =>
    !!resolveFilter(filter) && ['fork', 'archived', 'type', 'index', 'visibility'].includes(filter)

export function isTextFilter(filter: FilterType): boolean {
    return !!resolveFilter(filter) && !isFiniteFilter(filter)
}

/**
 * Some filter types should have their suggestions searched without influence
 * from the rest of the query, as they will then influence the scope of other filters.
 *
 * Same as {@link isolatedFuzzySearchFilters} but using FilterTypes rather than SuggestionTypes.
 */
export const isolatedFuzzySearchFiltersFilterType = [FilterType.repo, FilterType.repogroup]

export const FilterTypeToProseNames: Record<FilterType, string> = {
    repo: 'Repository',
    repogroup: 'Repository group',
    repohasfile: 'Repo has file',
    repohascommitafter: 'Repo has commit after',
    file: 'File',
    lang: 'Language',
    count: 'Count',
    timeout: 'Timeout',
    fork: 'Forks',
    archived: 'Archived repos',
    case: 'Case sensitive',
    after: 'Committed after',
    before: 'Committed before',
    message: 'Commit message contains',
    author: 'Commit author',
    committer: 'Committer',
    type: 'Type',
    content: 'Content',
    patterntype: 'Pattern type',
    index: 'Indexed repos',
    visibility: 'Repository visiblity',
    stable: 'Stable result ordering',
    rev: 'Revision',
}
