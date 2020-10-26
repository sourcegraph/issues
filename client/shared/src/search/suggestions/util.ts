import { FilterType } from '../interactive/util'

export type SuggestionType = FilterType | NonFilterSuggestionType

/**
 * NonFilterSuggestionType represents the types of suggestion results that do not match a filter.
 *
 * For example, there is no `symbol:` filter, but there are symbol suggestion results.
 */
export enum NonFilterSuggestionType {
    Filters = 'filters',
    Directory = 'dir',
    Symbol = 'symbol',
}
