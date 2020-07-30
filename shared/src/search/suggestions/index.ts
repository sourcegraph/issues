import { RepoGroup, SearchSuggestion as DynamicSearchSuggestion } from '../../graphql/schema'

export type SearchSuggestion = DynamicSearchSuggestion | RepoGroup
