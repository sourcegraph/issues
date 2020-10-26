import { Filter } from './parser'
import { SearchSuggestion } from '../suggestions'
import {
    FilterType,
    isNegatedFilter,
    resolveNegatedFilter,
    NegatableFilter,
    isNegatableFilter,
    isFilterType,
    isAliasedFilterType,
    AliasedFilterType,
} from '../interactive/util'
import { Omit } from 'utility-types'

interface BaseFilterDefinition {
    alias?: string
    description: string
    discreteValues?: string[]
    suggestions?: SearchSuggestion['__typename'] | string[]
    default?: string
    /** Whether the filter may only be used 0 or 1 times in a query. */
    singular?: boolean
}

interface NegatableFilterDefinition extends Omit<BaseFilterDefinition, 'description'> {
    negatable: true
    description: (negated: boolean) => string
}

export type FilterDefinition = BaseFilterDefinition | NegatableFilterDefinition

export const LANGUAGES: string[] = [
    'c',
    'cpp',
    'csharp',
    'css',
    'go',
    'graphql',
    'haskell',
    'html',
    'java',
    'javascript',
    'json',
    'lua',
    'markdown',
    'php',
    'powershell',
    'python',
    'r',
    'ruby',
    'rust',
    'sass',
    'swift',
    'typescript',
]

export const FILTERS: Record<NegatableFilter, NegatableFilterDefinition> &
    Record<Exclude<FilterType, NegatableFilter>, BaseFilterDefinition> = {
    [FilterType.after]: {
        description: 'Commits made after a certain date',
    },
    [FilterType.archived]: {
        description: 'Include results from archived repositories.',
        singular: true,
    },
    [FilterType.author]: {
        negatable: true,
        description: negated => `${negated ? 'Exclude' : 'Include only'} commits or diffs authored by a user.`,
    },
    [FilterType.before]: {
        description: 'Commits made before a certain date',
    },
    [FilterType.case]: {
        description: 'Treat the search pattern as case-sensitive.',
        discreteValues: ['yes', 'no'],
        default: 'no',
        singular: true,
    },
    [FilterType.committer]: {
        description: (negated: boolean): string =>
            `${negated ? 'Exclude' : 'Include only'} commits and diffs committed by a user.`,
        negatable: true,
        singular: true,
    },
    [FilterType.content]: {
        description: (negated: boolean): string =>
            `${negated ? 'Exclude' : 'Include only'} results from files if their content matches the search pattern.`,
        negatable: true,
        singular: true,
    },
    [FilterType.count]: {
        description: 'Number of results to fetch (integer)',
        singular: true,
    },
    [FilterType.file]: {
        alias: 'f',
        negatable: true,
        description: negated =>
            `${negated ? 'Exclude' : 'Include only'} results from files matching the given search pattern.`,
        suggestions: 'File',
    },
    [FilterType.fork]: {
        discreteValues: ['yes', 'no', 'only'],
        description: 'Include results from forked repositories.',
        singular: true,
    },
    [FilterType.index]: {
        discreteValues: ['yes', 'no', 'only'],
        description: 'Include results from indexed repositories',
        singular: true,
    },
    [FilterType.lang]: {
        negatable: true,
        description: negated => `${negated ? 'Exclude' : 'Include only'} results from the given language`,
        suggestions: LANGUAGES,
    },
    [FilterType.message]: {
        negatable: true,
        description: negated =>
            `${negated ? 'Exclude' : 'Include only'} Commits with messages matching a certain string`,
    },
    [FilterType.patterntype]: {
        discreteValues: ['regexp', 'literal', 'structural'],
        description: 'The pattern type (regexp, literal, structural) in use',
        singular: true,
    },
    [FilterType.repo]: {
        alias: 'r',
        negatable: true,
        description: negated =>
            `${negated ? 'Exclude' : 'Include only'} results from repositories matching the given search pattern.`,
        suggestions: 'Repository',
    },
    [FilterType.repogroup]: {
        description: 'group-name (include results from the named group)',
        singular: true,
        suggestions: 'RepoGroup',
    },
    [FilterType.repohascommitafter]: {
        description: '"string specifying time frame" (filter out stale repositories without recent commits)',
        singular: true,
    },
    [FilterType.repohasfile]: {
        negatable: true,
        description: negated =>
            `${negated ? 'Exclude' : 'Include only'} results from repos that contain a matching file`,
    },
    [FilterType.rev]: {
        description: 'Search a revision (branch, commit hash, or tag) instead of the default branch.',
        singular: true,
    },
    [FilterType.stable]: {
        discreteValues: ['yes', 'no'],
        default: 'no',
        description: 'Forces search to return a stable result ordering (currently limited to file content matches).',
        singular: true,
    },
    [FilterType.timeout]: {
        description: 'Duration before timeout',
        singular: true,
    },
    [FilterType.type]: {
        description: 'Limit results to the specified type.',
        discreteValues: ['diff', 'commit', 'symbol', 'repo', 'path', 'file'],
    },
    [FilterType.visibility]: {
        discreteValues: ['any', 'private', 'public'],
        description: 'Include results from repositories with the matching visibility (private, public, any).',
        singular: true,
    },
}

export const discreteValueAliases: { [key: string]: string[] } = {
    yes: ['yes', 'y', 'Y', 'YES', 'Yes', '1', 't', 'T', 'true', 'TRUE', 'True'],
    no: ['n', 'N', 'no', 'NO', 'No', '0', 'f', 'F', 'false', 'FALSE', 'False'],
    only: ['o', 'only', 'ONLY', 'Only'],
}

/**
 * Returns the {@link FilterDefinition} for the given filterType if it exists, or `undefined` otherwise.
 */
export const resolveFilter = (
    filterType: string
):
    | { type: NegatableFilter; negated: boolean; definition: NegatableFilterDefinition }
    | { type: Exclude<FilterType, NegatableFilter>; definition: BaseFilterDefinition }
    | undefined => {
    filterType = filterType.toLowerCase()

    if (isAliasedFilterType(filterType)) {
        const aliasKey = filterType as keyof typeof AliasedFilterType
        filterType = AliasedFilterType[aliasKey]
    }

    if (isNegatedFilter(filterType)) {
        const type = resolveNegatedFilter(filterType)
        return {
            type,
            definition: FILTERS[type],
            negated: true,
        }
    }
    if (isFilterType(filterType)) {
        if (isNegatableFilter(filterType)) {
            return {
                type: filterType,
                definition: FILTERS[filterType],
                negated: false,
            }
        }
        if (FILTERS[filterType]) {
            return { type: filterType, definition: FILTERS[filterType] }
        }
    }
    for (const [type, definition] of Object.entries(FILTERS as Record<FilterType, FilterDefinition>)) {
        if (definition.alias && filterType === definition.alias) {
            return {
                type: type as Exclude<FilterType, NegatableFilter>,
                definition: definition as BaseFilterDefinition,
            }
        }
    }
    return undefined
}

/**
 * Checks whether a discrete value is valid for a given filter, accounting for valid aliases.
 */
const isValidDiscreteValue = (definition: NegatableFilterDefinition | BaseFilterDefinition, value: string): boolean => {
    if (!definition.discreteValues || definition.discreteValues.includes(value)) {
        return true
    }

    const validDiscreteValuesForDefinition = Object.keys(discreteValueAliases).filter(key =>
        definition.discreteValues?.includes(key)
    )

    for (const discreteValue of validDiscreteValuesForDefinition) {
        if (discreteValueAliases[discreteValue].includes(value)) {
            return true
        }
    }
    return false
}

/**
 * Validates a filter given its type and value.
 */
export const validateFilter = (
    filterType: string,
    filterValue: Filter['filterValue']
): { valid: true } | { valid: false; reason: string } => {
    const typeAndDefinition = resolveFilter(filterType)
    if (!typeAndDefinition) {
        return { valid: false, reason: 'Invalid filter type.' }
    }
    const { definition } = typeAndDefinition
    if (
        definition.discreteValues &&
        (!filterValue ||
            (filterValue.token.type !== 'literal' && filterValue.token.type !== 'quoted') ||
            (filterValue.token.type === 'literal' && !isValidDiscreteValue(definition, filterValue.token.value)) ||
            (filterValue.token.type === 'quoted' && !isValidDiscreteValue(definition, filterValue.token.quotedValue)))
    ) {
        return {
            valid: false,
            reason: `Invalid filter value, expected one of: ${definition.discreteValues.join(', ')}.`,
        }
    }
    return { valid: true }
}

/** Whether a given filter type may only be used 0 or 1 times in a query. */
export const isSingularFilter = (filter: string): boolean =>
    Object.keys(FILTERS)
        .filter(key => FILTERS[key as FilterType].singular)
        .includes(filter)
