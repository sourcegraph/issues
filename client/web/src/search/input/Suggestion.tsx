import * as React from 'react'
import { LanguageIcon } from '../../../../shared/src/components/languageIcons'
import { dirname, basename } from '../../util/path'
import FilterIcon from 'mdi-react/FilterIcon'
import FileIcon from 'mdi-react/FileIcon'
import SourceRepositoryIcon from 'mdi-react/SourceRepositoryIcon'
import { SymbolIcon } from '../../../../shared/src/symbols/SymbolIcon'
import { SuggestionType, NonFilterSuggestionType } from '../../../../shared/src/search/suggestions/util'
import { escapeRegExp } from 'lodash'
import { FilterType } from '../../../../shared/src/search/interactive/util'
import { SearchSuggestion } from '../../../../shared/src/search/suggestions'
import { appendSubtreeQueryParameter } from '../../../../shared/src/util/url'
import { SymbolKind } from '../../../../shared/src/graphql-operations'

export const filterAliases: Record<string, FilterSuggestionTypes> = {
    r: FilterType.repo,
    g: FilterType.repogroup,
    f: FilterType.file,
    l: FilterType.lang,
    language: FilterType.lang,
}

/**
 * Filters which use fuzzy-search for their suggestion values
 */
export const fuzzySearchFilters: FilterSuggestionTypes[] = [
    FilterType.repo,
    FilterType.repogroup,
    FilterType.file,
    FilterType.repohasfile,
]

/**
 * Some filter types should have their suggestions searched without influence
 * from the rest of the query, as they will then influence the scope of other filters.
 */
export const isolatedFuzzySearchFilters: FilterSuggestionTypes[] = [FilterType.repo, FilterType.repogroup]

/**
 * dir and symbol are fetched/suggested by the fuzzy-search
 * but are not filter types: /web/src/search/searchFilterSuggestions.ts
 */
export type FilterSuggestionTypes = FilterType | 'filters'

export interface Suggestion {
    type: SuggestionType
    /** The value to be suggested and that will be added to queries */
    value: string
    /**
     * Optional value to use when suggestion is displayed to the user.
     * Useful for displaying a "human-readable" suggestion value
     */
    displayValue?: string
    /** Description that will be displayed together with suggestion value */
    description?: string
    /** Fuzzy-search suggestions may have a url for redirect when selected */
    url?: string
    /** Label informing what will happen when suggestion is selected */
    label?: string
    /** For suggestions of type `symbol` */
    symbolKind?: SymbolKind
    /** If the suggestion was loaded from the fuzzy-search */
    fromFuzzySearch?: true
}

interface SuggestionIconProps {
    suggestion: Suggestion
    className?: string
}

/**
 * @returns The given string with escaped special characters and wrapped with regex boundaries
 */
const formatRegExp = (value: string): string => '^' + escapeRegExp(value) + '$'

export function createSuggestion(item: SearchSuggestion, globbing: boolean): Suggestion | undefined {
    switch (item.__typename) {
        case 'Repository': {
            return {
                type: FilterType.repo,
                // Add "regex start and end boundaries" to
                // correctly scope additional suggestions
                value: globbing ? item.name : formatRegExp(item.name),
                displayValue: item.name,
                url: `/${item.name}`,
                label: 'go to repository',
            }
        }
        case 'File': {
            const descriptionParts = []
            const directory = dirname(item.path)
            if (directory !== '.') {
                descriptionParts.push(`${directory}/`)
            }
            descriptionParts.push(basename(item.repository.name))
            if (item.isDirectory) {
                return {
                    type: NonFilterSuggestionType.Directory,
                    value: globbing ? item.path : '^' + escapeRegExp(item.path),
                    description: descriptionParts.join(' — '),
                    url: appendSubtreeQueryParameter(item.url),
                    label: 'go to dir',
                }
            }
            return {
                type: FilterType.file,
                value: globbing ? item.path : formatRegExp(item.path),
                displayValue: item.name,
                description: descriptionParts.join(' — '),
                url: appendSubtreeQueryParameter(item.url),
                label: 'go to file',
            }
        }
        case 'Symbol': {
            return {
                type: NonFilterSuggestionType.Symbol,
                symbolKind: item.kind,
                value: item.name,
                description: `${item.containerName || item.location.resource.path} — ${basename(
                    item.location.resource.repository.name
                )}`,
                url: item.url,
                label: 'go to definition',
            }
        }
        case 'RepoGroup':
            return {
                type: FilterType.repogroup,
                value: item.name,
            }
        default:
            return undefined
    }
}

const SuggestionIcon: React.FunctionComponent<SuggestionIconProps> = ({ suggestion, children, ...props }) => {
    switch (suggestion.type) {
        case NonFilterSuggestionType.Filters:
            return <FilterIcon {...props} />
        case FilterType.repo:
        case FilterType.repogroup:
            return <SourceRepositoryIcon {...props} />
        case FilterType.file:
            return <FileIcon {...props} />
        case FilterType.lang:
            return <LanguageIcon {...props} language={suggestion.value} {...props} />
        case NonFilterSuggestionType.Symbol:
            if (!suggestion.symbolKind) {
                return null
            }
            return <SymbolIcon kind={suggestion.symbolKind} {...props} />
        default:
            return null
    }
}

interface SuggestionProps {
    suggestion: Suggestion
    isSelected?: boolean
    onClick?: () => void
    /** If suggestion.label should be shown, else show defaultLabel  */
    showUrlLabel: boolean
    /** Suggestion label to show if(!showUrlLabel || !suggestion.label) */
    defaultLabel?: string
}

export const SuggestionItem: React.FunctionComponent<SuggestionProps> = ({
    suggestion,
    isSelected,
    showUrlLabel,
    defaultLabel,
    ...props
}) => (
    <li className={`suggestion ${isSelected ? ' suggestion--selected' : ''} test-suggestion-item`} {...props}>
        <SuggestionIcon className="icon-inline suggestion__icon" suggestion={suggestion} />
        <div className="suggestion__title">{suggestion.displayValue ?? suggestion.value}</div>
        <div className="suggestion__description">{suggestion.description}</div>
        {(showUrlLabel || defaultLabel) && (
            <div className="suggestion__action" hidden={!isSelected}>
                <kbd>enter</kbd> {(showUrlLabel && suggestion?.label) || defaultLabel}
            </div>
        )}
    </li>
)
