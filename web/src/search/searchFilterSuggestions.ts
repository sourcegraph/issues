import { Suggestion, FiltersSuggestionTypes } from './input/Suggestion'
import { assign } from 'lodash/fp'
import { languageIcons } from '../../../shared/src/components/languageIcons'
import { NonFilterSuggestionTypes } from '../../../shared/src/search/suggestions/util'
import { FilterTypes } from '../../../shared/src/search/interactive/util'

export type SearchFilterSuggestions = Record<
    FiltersSuggestionTypes,
    {
        default?: string
        values: Suggestion[]
    }
>

export const searchFilterSuggestions: SearchFilterSuggestions = {
    filters: {
        values: [
            {
                value: 'repo:',
                description: 'regex-pattern (include results whose repository path matches)',
            },
            {
                value: '-repo:',
                description: 'regex-pattern (exclude results whose repository path matches)',
            },
            {
                value: 'repogroup:',
                description: 'group-name (include results from the named group)',
            },
            {
                value: 'repohasfile:',
                description: 'regex-pattern (include results from repos that contain a matching file)',
            },
            {
                value: '-repohasfile:',
                description: 'regex-pattern (exclude results from repositories that contain a matching file)',
            },
            {
                value: 'repohascommitafter:',
                description: '"string specifying time frame" (filter out stale repositories without recent commits)',
            },
            {
                value: 'file:',
                description: 'regex-pattern (include results whose file path matches)',
            },
            {
                value: '-file:',
                description: 'regex-pattern (exclude results whose file path matches)',
            },
            {
                value: 'type:',
                description: 'code | diff | commit | symbol',
            },
            {
                value: 'case:',
                description: 'yes | no (default)',
            },
            {
                value: 'lang:',
                description: 'lang-name (include results from the named language)',
            },
            {
                value: '-lang:',
                description: 'lang-name (exclude results from the named language)',
            },
            {
                value: 'fork:',
                description: 'no | only | yes (default)',
            },
            {
                value: 'archived:',
                description: 'no | only | yes (default)',
            },
            {
                value: 'count:',
                description: 'integer (number of results to fetch)',
            },
            {
                value: 'timeout:',
                description: '"string specifying time duration" (duration before timeout)',
            },
            {
                value: 'after:',
                description: '"string specifying time frame" (time frame to match commits after)',
            },
            {
                value: 'before:',
                description: '"string specifying time frame" (time frame to match commits before)',
            },
            {
                value: 'message:',
                description: 'commit message contents',
            },
            {
                value: 'content:',
                description: 'override the search pattern',
            },
        ].map(
            assign({
                type: NonFilterSuggestionTypes.filters,
            })
        ),
    },
    type: {
        default: 'code',
        values: [{ value: 'code' }, { value: 'diff' }, { value: 'commit' }, { value: 'symbol' }].map(
            assign({
                type: FilterTypes.type,
            })
        ),
    },
    case: {
        default: 'no',
        values: [{ value: 'yes' }, { value: 'no' }].map(
            assign({
                type: FilterTypes.case,
            })
        ),
    },
    fork: {
        default: 'yes',
        values: [{ value: 'no' }, { value: 'only' }, { value: 'yes' }].map(
            assign({
                type: FilterTypes.fork,
            })
        ),
    },
    archived: {
        default: 'yes',
        values: [{ value: 'no' }, { value: 'only' }, { value: 'yes' }].map(
            assign({
                type: FilterTypes.archived,
            })
        ),
    },
    file: {
        values: [
            { value: '(test|spec)', displayValue: 'Test files' },
            { value: '.(txt|md)', displayValue: 'Text files' },
        ].map(suggestion => ({
            ...suggestion,
            description: suggestion.value,
            type: FilterTypes.file,
        })),
    },
    lang: {
        values: Object.keys(languageIcons).map(value => ({ type: FilterTypes.lang, value })),
    },
    repogroup: {
        values: [],
    },
    repo: {
        values: [],
    },
    repohasfile: {
        values: [],
    },
    repohascommitafter: {
        values: [{ value: "'1 week ago'" }, { value: "'1 month ago'" }].map(
            assign({
                type: FilterTypes.repohascommitafter,
            })
        ),
    },
    count: {
        values: [{ value: '100' }, { value: '1000' }].map(
            assign({
                type: FilterTypes.count,
            })
        ),
    },
    timeout: {
        values: [{ value: '10s' }, { value: '30s' }].map(
            assign({
                type: FilterTypes.timeout,
            })
        ),
    },
    author: {
        values: [],
    },
    message: {
        values: [],
    },
    before: {
        values: [{ value: '"1 week ago"' }, { value: '"1 day ago"' }, { value: '"last thursday"' }].map(
            assign({ type: FilterTypes.before })
        ),
    },
    after: {
        values: [{ value: '"1 week ago"' }, { value: '"1 day ago"' }, { value: '"last thursday"' }].map(
            assign({ type: FilterTypes.after })
        ),
    },
    content: {
        values: [],
    },
    patterntype: {
        values: [{ value: 'literal' }, { value: 'structural' }, { value: 'regexp' }].map(
            assign({ type: FilterTypes.patterntype })
        ),
    },
}
