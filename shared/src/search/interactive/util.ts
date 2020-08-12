/**
 * The data structure that holds the filters in a query.
 *
 */
export interface FiltersToTypeAndValue {
    /**
     * Key is a unique string, generated by uniqueId appended to `filterType` .
     * */
    [key: string]: {
        // `type` is the field type of the filter (repo, file, etc.)
        type: FilterType
        // `value` is the current value for that particular filter,
        value: string
        // `editable` is whether the corresponding filter input is currently editable in the UI.
        editable: boolean
        // `negated` is whether the filter is negated. Optional because some filters are non-negatable.
        negated?: boolean
    }
}

export enum FilterType {
    repo = 'repo',
    repogroup = 'repogroup',
    repohasfile = 'repohasfile',
    repohascommitafter = 'repohascommitafter',
    file = 'file',
    type = 'type',
    case = 'case',
    lang = 'lang',
    fork = 'fork',
    archived = 'archived',
    visibility = 'visibility',
    count = 'count',
    timeout = 'timeout',
    before = 'before',
    after = 'after',
    author = 'author',
    message = 'message',
    content = 'content',
    patterntype = 'patterntype',
    index = 'index',
}

export const isFilterType = (filter: string): filter is FilterType => filter in FilterType

export const filterTypeKeys: FilterType[] = Object.keys(FilterType) as FilterType[]

export enum NegatedFilters {
    repo = '-repo',
    file = '-file',
    lang = '-lang',
    r = '-r',
    f = '-f',
    l = '-l',
    repohasfile = '-repohasfile',
    content = '-content',
}

/** The list of filters that are able to be negated. */
export type NegatableFilter =
    | FilterType.repo
    | FilterType.file
    | FilterType.repohasfile
    | FilterType.lang
    | FilterType.content

export const isNegatableFilter = (filter: FilterType): filter is NegatableFilter =>
    Object.keys(NegatedFilters).includes(filter)

/** The list of all negated filters. i.e. all valid filters that have `-` as a suffix. */
export const negatedFilters = Object.values(NegatedFilters)

export const isNegatedFilter = (filter: string): filter is NegatedFilters =>
    negatedFilters.includes(filter as NegatedFilters)

const negatedFilterToNegatableFilter: { [key: string]: NegatableFilter } = {
    '-repo': FilterType.repo,
    '-file': FilterType.file,
    '-lang': FilterType.lang,
    '-r': FilterType.repo,
    '-f': FilterType.file,
    '-l': FilterType.lang,
    '-repohasfile': FilterType.repohasfile,
    '-content': FilterType.content,
}

export const resolveNegatedFilter = (filter: NegatedFilters): NegatableFilter => negatedFilterToNegatableFilter[filter]
