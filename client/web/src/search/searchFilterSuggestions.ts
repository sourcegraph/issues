import { Suggestion, FilterSuggestionTypes } from './input/Suggestion'
import { assign } from 'lodash/fp'
import { languageIcons } from '../../../shared/src/components/languageIcons'
import { NonFilterSuggestionType } from '../../../shared/src/search/suggestions/util'
import { FilterType } from '../../../shared/src/search/interactive/util'

export type SearchFilterSuggestions = Record<
    FilterSuggestionTypes,
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
                value: 'author:',
                description: 'username or git email of commit author',
            },
            {
                value: 'message:',
                description: 'commit message contents',
            },
            {
                value: 'committer:',
                description: 'git email of committer',
            },
            {
                value: 'content:',
                description: 'override the search pattern',
            },
            {
                value: 'visibility:',
                description: 'any | public | private',
            },
            {
                value: 'stable',
                description: 'yes | no',
            },
            {
                value: 'rev',
                description: 'repository revision (branch, commit hash, or tag), ',
            },
        ].map(
            assign({
                type: NonFilterSuggestionType.Filters,
            })
        ),
    },
    type: {
        values: [{ value: 'diff' }, { value: 'commit' }, { value: 'symbol' }, { value: 'file' }, { value: 'path' }].map(
            assign({
                type: FilterType.type,
            })
        ),
    },
    case: {
        default: 'no',
        values: [{ value: 'yes' }, { value: 'no' }].map(
            assign({
                type: FilterType.case,
            })
        ),
    },
    fork: {
        default: 'yes',
        values: [{ value: 'no' }, { value: 'only' }, { value: 'yes' }].map(
            assign({
                type: FilterType.fork,
            })
        ),
    },
    archived: {
        default: 'yes',
        values: [{ value: 'no' }, { value: 'only' }, { value: 'yes' }].map(
            assign({
                type: FilterType.archived,
            })
        ),
    },
    visibility: {
        default: 'any',
        values: [{ value: 'any' }, { value: 'private' }, { value: 'public' }].map(
            assign({
                type: FilterType.visibility,
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
            type: FilterType.file,
        })),
    },
    lang: {
        values: Object.keys(languageIcons).map(value => ({ type: FilterType.lang, value })),
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
                type: FilterType.repohascommitafter,
            })
        ),
    },
    count: {
        values: [{ value: '100' }, { value: '1000' }].map(
            assign({
                type: FilterType.count,
            })
        ),
    },
    timeout: {
        values: [{ value: '10s' }, { value: '30s' }].map(
            assign({
                type: FilterType.timeout,
            })
        ),
    },
    author: {
        values: [],
    },
    committer: {
        values: [],
    },
    message: {
        values: [],
    },
    before: {
        values: [{ value: '"1 week ago"' }, { value: '"1 day ago"' }, { value: '"last thursday"' }].map(
            assign({ type: FilterType.before })
        ),
    },
    after: {
        values: [{ value: '"1 week ago"' }, { value: '"1 day ago"' }, { value: '"last thursday"' }].map(
            assign({ type: FilterType.after })
        ),
    },
    content: {
        values: [],
    },
    patterntype: {
        values: [{ value: 'literal' }, { value: 'structural' }, { value: 'regexp' }].map(
            assign({ type: FilterType.patterntype })
        ),
    },
    index: {
        default: 'yes',
        values: [{ value: 'no' }, { value: 'only' }, { value: 'yes' }].map(
            assign({
                type: FilterType.index,
            })
        ),
    },
    stable: {
        values: [{ value: 'no' }, { value: 'yes' }].map(
            assign({
                type: FilterType.stable,
            })
        ),
    },
    rev: {
        values: [],
    },
}
