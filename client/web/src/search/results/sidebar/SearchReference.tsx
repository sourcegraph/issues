import { Tab, TabList, TabPanel, TabPanels, Tabs } from '@reach/tabs'
import classNames from 'classnames'
import { escapeRegExp } from 'lodash'
import ChevronDownIcon from 'mdi-react/ChevronDownIcon'
import ChevronLeftIcon from 'mdi-react/ChevronLeftIcon'
import ExternalLinkIcon from 'mdi-react/ExternalLinkIcon'
import { Range, Selection } from 'monaco-editor'
import React, { ReactElement, useCallback, useMemo, useState } from 'react'
import { Collapse } from 'reactstrap'

import { Link } from '@sourcegraph/shared/src/components/Link'
import { Markdown } from '@sourcegraph/shared/src/components/Markdown'
import { SearchPatternType } from '@sourcegraph/shared/src/graphql-operations'
import { FILTERS, FilterType, isNegatableFilter } from '@sourcegraph/shared/src/search/query/filters'
import { scanSearchQuery } from '@sourcegraph/shared/src/search/query/scanner'
import { appendFilter, updateFilter } from '@sourcegraph/shared/src/search/query/transformer'
import { findFilter, FilterKind } from '@sourcegraph/shared/src/search/query/validate'
import { VersionContextProps } from '@sourcegraph/shared/src/search/util'
import { TelemetryProps } from '@sourcegraph/shared/src/telemetry/telemetryService'
import { renderMarkdown } from '@sourcegraph/shared/src/util/markdown'
import { useLocalStorage } from '@sourcegraph/shared/src/util/useLocalStorage'

import { CaseSensitivityProps, PatternTypeProps, SearchContextProps } from '../..'
import { QueryChangeSource, QueryState } from '../../helpers'

import styles from './SearchReference.module.scss'
import sidebarStyles from './SearchSidebarSection.module.scss'

const SEARCH_REFERENCE_TAB_KEY = 'SearchProduct.SearchReference.Tab'

export interface FilterInfo {
    type: FilterType
    placeholder: Placeholder
    description: string
    /**
     * Force showing or not showing suggestions for this fileter.
     */
    showSuggestions?: boolean
    /**
     * Used to indicate whether this filter/example should be listed in the
     * "Common" filters section and at which position
     */
    commonRank?: number
    alias?: string
    examples?: string[]
}

interface OperatorInfo {
    operator: string
    placeholder: Placeholder
    description: string
    alias?: string
    examples?: string[]
}

type SearchReferenceInfo = FilterInfo | OperatorInfo

/**
 * Adds additional search reference information from the existing filters list.
 */
function augmentFilterInfo(searchReference: FilterInfo): void {
    const filter = FILTERS[searchReference.type]
    if (filter?.alias) {
        searchReference.alias = filter.alias
    }
}

const filterInfos: FilterInfo[] = [
    {
        type: FilterType.after,
        placeholder: parsePlaceholder('"{last week}"'),
        description:
            'Only include results from diffs or commits which have a commit date after the specified time frame. To use this filter, the search query must contain `type:diff` or `type:commit`.',
        commonRank: 100,
        examples: ['after:"6 weeks ago"', 'after:"november 1 2019"'],
    },
    {
        type: FilterType.archived,
        placeholder: parsePlaceholder('{yes/only}'),
        description:
            'The "yes" option includes archived repositories. The "only" option filters results to only archived repositories. Results in archived repositories are excluded by default.',
        examples: ['repo:sourcegraph/ archived:only'],
    },
    {
        type: FilterType.author,
        placeholder: parsePlaceholder('{name}'),
        description: `Only include results from diffs or commits authored by the user. Regexps are supported. Note that they match the whole author string of the form \`Full Name <user@example.com>\`, so to include only authors from a specific domain, use \`author:example.com>$\`.

You can also search by \`committer:git-email\`. *Note: there is a committer only when they are a different user than the author.*

To use this filter, the search query must contain \`type:diff\` or \`type:commit\`.`,
        examples: ['type:diff author:nick'],
    },
    {
        type: FilterType.before,
        placeholder: parsePlaceholder('"{last thursday}"'),
        description:
            'Only include results from diffs or commits which have a commit date before the specified time frame. To use this filter, the search query must contain `type:diff` or `type:commit`.',
        commonRank: 100,
        examples: ['before:"last thursday"', 'before:"november 1 2019"'],
    },
    {
        type: FilterType.case,
        placeholder: parsePlaceholder('{yes}'),
        description: 'Perform a case sensitive query. Without this, everything is matched case insensitively.',
        examples: ['OPEN_FILE case:yes'],
    },
    {
        type: FilterType.content,
        placeholder: parsePlaceholder('"{pattern}"'),
        description:
            'Set the search pattern with a dedicated parameter. Useful when searching literally for a string that may conflict with the search pattern syntax. In between the quotes, the `\\` character will need to be escaped (`\\\\` to evaluate for `\\`).',
        commonRank: 70,
        examples: ['repo:sourcegraph content:"repo:sourcegraph"', 'file:Dockerfile alpine -content:alpine:latest'],
    },
    {
        type: FilterType.count,
        placeholder: parsePlaceholder('{N/all}'),
        description:
            'Retrieve *N* results. By default, Sourcegraph stops searching early and returns if it finds a full page of results. This is desirable for most interactive searches. To wait for all results, use **count:all**.',
        commonRank: 60,
        examples: ['count:1000 function', 'count:all err'],
    },
    {
        type: FilterType.file,
        placeholder: parsePlaceholder('{regexp-pattern}'),
        commonRank: 30,
        description: 'Only include results in files whose full path matches the regexp.',
        examples: ['file:.js$ httptest', 'file:internal/ httptest', 'file:.js$ -file:test http'],
    },
    {
        type: FilterType.file,
        placeholder: parsePlaceholder('contains.content({regexp-pattern})'),
        description: 'Search only inside files that contain content matching the provided regexp pattern.',
        examples: ['file:contains.content(github.com/sourcegraph/sourcegraph)'],
    },
    {
        type: FilterType.fork,
        placeholder: parsePlaceholder('{yes/only}'),
        description:
            'Include results from repository forks or filter results to only repository forks. Results in repository forks are exluded by default.',
        commonRank: 80,
        examples: ['fork:yes repo:sourcegraph'],
    },
    {
        type: FilterType.lang,
        placeholder: parsePlaceholder('{language-name}'),
        description: 'Only include results from files in the specified programming language.',
        commonRank: 40,
        examples: ['lang:typescript encoding', '-lang:typescript encoding'],
    },
    {
        type: FilterType.message,
        placeholder: parsePlaceholder('"{any string}"'),
        description: `Only include results from diffs or commits which have commit messages containing the string.

To use this filter, the search query must contain \`type:diff\` or \`type:commit\`.`,
        examples: ['type:commit message:"testing"', 'type:diff message:"testing"'],
    },
    {
        type: FilterType.repo,
        placeholder: parsePlaceholder('{regexp-pattern}'),
        description:
            'Only include results from repositories whose path matches the regexp-pattern. A repository’s path is a string such as *github.com/myteam/abc* or *code.example.com/xyz* that depends on your organization’s repository host. If the regexp ends in `@rev`, that revision is searched instead of the default branch (usually `master`). `repo:regexp-pattern@rev` is equivalent to `repo:regexp-pattern rev:rev`.',
        commonRank: 10,
        examples: [
            'repo:gorilla/mux testroute',
            'repo:^github.com/sourcegraph/sourcegraph$@v3.14.0 mux',
            'repo:alice/ -repo:old-repo',
            'repo:vscode@*refs/heads/:^refs/heads/master type:diff task',
        ],
    },
    {
        type: FilterType.repogroup,
        placeholder: parsePlaceholder('{group-name}'),
        description:
            'Only include results from the named group of repositories (defined by the server admin). Same as using a repo: keyword that matches all of the group’s repositories. Use repo: unless you know that the group exists.',
    },
    {
        type: FilterType.repo,
        placeholder: parsePlaceholder('contains.file({path})'),
        description: 'Search only inside repositories that contain a file path matching the regular expression.',
        examples: ['repo:contains.file(README)'],
        showSuggestions: false,
    },
    {
        type: FilterType.repo,
        placeholder: parsePlaceholder('contains.content({content})'),
        description: 'Search only inside repositories that contain file content matching the regular expression.',
        examples: ['repo:contains.content(TODO)'],
        showSuggestions: false,
    },
    {
        type: FilterType.repo,
        placeholder: parsePlaceholder('contains({file:path content:content})'),
        description:
            'Search only inside repositories that contain a file matching the `file:` with `content:` filters.',
        examples: ['repo:contains(file:CHANGELOG content:fix)'],
        showSuggestions: false,
    },
    {
        type: FilterType.repo,
        placeholder: parsePlaceholder('contains.commit.after({date})'),
        description:
            'Search only inside repositories that contain a a commit after some specified time. See [git date formats](https://github.com/git/git/blob/master/Documentation/date-formats.txt) for accepted formats. Use this to filter out stale repositories that don’t contain commits past the specified time frame. This parameter is experimental.',
        examples: ['repo:contains.commit.after(1 month ago)', 'repo:contains.commit.after(june 25 2017)'],
        showSuggestions: false,
    },
    {
        type: FilterType.rev,
        placeholder: parsePlaceholder('{revision}'),
        commonRank: 20,
        description:
            'Search a revision instead of the default branch. `rev:` can only be used in conjunction with `repo:` and may not be used more than once. See our [revision syntax documentation](https://docs.sourcegraph.com/code_search/reference/queries#repository-revisions) to learn more.',
    },
    {
        type: FilterType.select,
        placeholder: parsePlaceholder('{result-types}'),
        commonRank: 50,
        description:
            'Shows only query results for a given type. For example, `select:repo` displays only distinct reopsitory paths from search results. See [language definition](https://docs.sourcegraph.com/code_search/reference/language#select) for possible values.',
        examples: ['fmt.Errorf select:repo'],
    },
    {
        type: FilterType.type,
        placeholder: parsePlaceholder('{diff/commit/...}'),
        commonRank: 1,
        description:
            'Specifies the type of search. By default, searches are executed on all code at a given point in time (a branch or a commit). Specify the `type:` if you want to search over changes to code or commit messages instead (diffs or commits).',
        examples: ['type:symbol path', 'type:diff func', 'type:commit test'],
    },
    {
        type: FilterType.timeout,
        placeholder: parsePlaceholder('{golang-duration-value}'),
        description:
            'Customizes the timeout for searches. The value of the parameter is a string that can be parsed by the [Go time package’s `ParseDuration`](https://golang.org/pkg/time/#ParseDuration) (e.g. 10s, 100ms). By default, the timeout is set to 10 seconds, and the search will optimize for returning results as soon as possible. The timeout value cannot be set longer than 1 minute. When provided, the search is given the full timeout to complete.',
        examples: ['repo:^github.com/sourcegraph timeout:15s func count:10000'],
    },
    {
        type: FilterType.visibility,
        placeholder: parsePlaceholder('{any/private/public}'),
        description:
            'Filter results to only public or private repositories. The default is to include both private and public repositories.',
        examples: ['type:repo visibility:public'],
    },
]

for (const info of filterInfos) {
    augmentFilterInfo(info)
}

const commonFilters = filterInfos
    .filter(info => info.commonRank !== undefined)
    // commonRank will never be undefined here, but TS doesn't seem to know
    .sort((a, b) => (a.commonRank as number) - (b.commonRank as number))

const operatorInfo: OperatorInfo[] = [
    {
        operator: 'AND',
        alias: 'and',
        placeholder: parsePlaceholder('{a} AND {b}'),
        description:
            'Returns results for files containing matches on the left and right side of the `and` (set intersection).',
        examples: ['conf.Get( and log15.Error(', 'conf.Get( AND log15.Error( AND after'],
    },
    {
        operator: 'OR',
        alias: 'or',
        placeholder: parsePlaceholder('{a} OR {b}'),
        description:
            'Returns file content matching either on the left or right side, or both (set union). The number of results reports the number of matches of both strings. Note the regex or operator `|` may not work as expected with certain operators for example `file:(internal/repos)|(internal/gitserver)`, to recieve the expected results use [subexpressions](https://docs.sourcegraph.com/code_search/tutorials/search_subexpressions), `(file:internal/repos or file:internal/gitserver)`',
        examples: ['conf.Get( or log15.Error(', 'conf.Get( OR log15.Error( OR after'],
    },
    {
        operator: 'NOT',
        alias: 'not',
        placeholder: parsePlaceholder('NOT {a}'),
        description:
            '`NOT` can be used in place of `-` to negate keywords, such as `file`, `content`, `lang`, `repohasfile`, and `repo`. For search patterns, `NOT` excludes documents that contain the term after `NOT`. For readability, you can also include the `AND` operator before a `NOT` (i.e. `panic NOT ever` is equivalent to `panic AND NOT ever`).',
        examples: ['lang:go not file:main.go panic', 'panic NOT ever'],
    },
]

/**
 * Returns true if the provided regular expressions all match the provided
 * filter information (name, description, ...)
 */
function matches(searchTerms: RegExp[], info: FilterInfo): boolean {
    return searchTerms.every(term => term.test(info.type) || term.test(info.description || ''))
}

/**
 * Convert the search input into an array of regular expressions. Each word in
 * the input becomes a regular expression starting with a word boundary check.
 */
function parseSearchInput(searchInput: string): RegExp[] {
    const terms = searchInput.split(/\s+/)
    return terms.map(term => new RegExp(`\\b${escapeRegExp(term)}`))
}

/**
 * Given a Placeholder object, this function returns the Monaco Selections
 * corresponding to each value in the Placeholder.
 */
function selectionForPlaceholder(
    placeholder: Placeholder,
    offset: number = 0,
    forSuggestions: boolean = false
): Selection {
    const token = placeholder.tokens.find(token => token.type === 'value')
    if (!token) {
        throw new Error('Search reference does not contain placeholder.')
    }
    const selectionStart = offset + token.start + 1
    // For filters with suggestions we create an "empty" selection to position
    // the cursor right after the colon.
    const selectionEnd = forSuggestions ? selectionStart : offset + token.end
    return new Selection(1, selectionStart, 1, selectionEnd)
}

/**
 * Whether or not to trigger the suggestion popover when adding this filter to
 * the query.
 */
function shouldShowSuggestions(searchReference: FilterInfo): boolean {
    return Boolean(searchReference.showSuggestions !== false && FILTERS[searchReference.type].discreteValues)
}

/**
 * This helper function will update the current query with the provided filter,
 * updating existing filters if necessary.
 * exported only for test purposes
 */
export function updateQueryWithFilter(
    currentQueryState: QueryState,
    filterInfo: FilterInfo,
    negate: boolean,
    allFilters: typeof FILTERS
): QueryState {
    const { singular } = allFilters[filterInfo.type]
    let { query } = currentQueryState
    const showSuggestions = shouldShowSuggestions(filterInfo)
    let selection: Selection | undefined
    let revealRange: Range
    let field: string = filterInfo.type

    if (negate && isNegatableFilter(filterInfo.type)) {
        field = '-' + field
    }

    const existingFilter = findFilter(query, filterInfo.type, FilterKind.Global)

    if (existingFilter && singular) {
        // Filter can only appear once
        // Select or remove the existing filter value
        if (showSuggestions) {
            query = updateFilter(query, field, '')
        }
        const selectionStart = (existingFilter.value?.range.start || existingFilter.field.range.end) + 1
        // For filters with suggestions, we create an "empty" selection to
        // position the cursor after the colon
        const selectionEnd = showSuggestions ? selectionStart : existingFilter.range.end + 1
        selection = new Selection(1, selectionStart, 1, selectionEnd)
        // A separate range is needed to make sure that the full filter, including
        // the field name, is scrolled into view.
        revealRange = new Range(1, existingFilter.range.start + 1, 1, selectionEnd)
    } else {
        // Filter can appear multiple times or doesn't exist yet. Always
        // append.

        // +1 because appendFilter inserts a whitespace character at the end of
        // the query
        const rangeStart = query.length + 1
        query = appendFilter(query, field, showSuggestions ? '' : filterInfo.placeholder.text)
        const offset = query.length - (showSuggestions ? 0 : filterInfo.placeholder.text.length)
        selection = selectionForPlaceholder(
            filterInfo.placeholder,
            offset,
            // If we need to trigger the suggestion popover we have to make
            // sure the input cursor is positioned at the beginning of the
            // selection (it usually is at the end)
            showSuggestions
        )
        // A separate range is needed to make sure that the full filter, including
        // the field name, is scrolled into view.
        revealRange = new Range(1, rangeStart + 1, 1, selection.endColumn)
    }

    return {
        changeSource: QueryChangeSource.searchReference,
        query,
        selection,
        showSuggestions,
        revealRange,
    }
}

interface Placeholder {
    tokens: { type: 'text' | 'value'; content: string; start: number; end: number }[]
    text: string
}

export function parsePlaceholder(placeholder: string): Placeholder {
    const valuePattern = /{([^}]+)}/g
    let currentIndex = 0
    const parsedPlaceholder: Placeholder = { tokens: [], text: '' }
    let match
    while ((match = valuePattern.exec(placeholder))) {
        if (currentIndex !== match.index) {
            parsedPlaceholder.tokens.push({
                type: 'text',
                content: placeholder.slice(currentIndex, match.index),
                start: currentIndex,
                end: match.index - 1,
            })
        }
        parsedPlaceholder.tokens.push({
            type: 'value',
            content: match[1],
            start: match.index,
            end: match.index + match[0].length - 1,
        })
        currentIndex = match.index + match[0].length
    }

    if (currentIndex < placeholder.length) {
        parsedPlaceholder.tokens.push({
            type: 'text',
            content: placeholder.slice(currentIndex),
            start: currentIndex,
            end: placeholder.length - 1,
        })
    }
    parsedPlaceholder.text = parsedPlaceholder.tokens.map(token => token.content).join('')
    return parsedPlaceholder
}

function isFilterInfo(searchReference: SearchReferenceInfo): searchReference is FilterInfo {
    return (searchReference as FilterInfo).type !== undefined
}

const classNameTokenMap = {
    text: 'search-filter-keyword',
    value: styles.placeholder,
}

interface SearchReferenceExampleProps {
    example: string
    onClick?: (example: string) => void
}

const SearchReferenceExample: React.FunctionComponent<SearchReferenceExampleProps> = ({ example, onClick }) => {
    // All current examples are literal queries
    const scanResult = scanSearchQuery(example, false, SearchPatternType.literal)
    // We only use valid queries as examples, so this will always be true
    if (scanResult.type === 'success') {
        return (
            <button className="btn p-0 flex-1" type="button" onClick={() => onClick?.(example)}>
                {scanResult.term.map((term, index) => {
                    switch (term.type) {
                        case 'filter':
                            return (
                                <React.Fragment key={index}>
                                    <span className="search-filter-keyword">{term.field.value}:</span>
                                    {term.value?.quoted ? `"${term.value.value}"` : term.value?.value}
                                </React.Fragment>
                            )
                        case 'keyword':
                            // We are using example.slice instead of term.value
                            // to get the actual character sequence in the
                            // example. term.value doesn't preserve case
                            return (
                                <span className="search-filter-keyword">
                                    {example.slice(term.range.start, term.range.end)}
                                </span>
                            )
                        default:
                            return example.slice(term.range.start, term.range.end)
                    }
                })}
            </button>
        )
    }
    return null
}

interface SearchReferenceEntryProps<T extends SearchReferenceInfo> {
    searchReference: T
    onClick: (searchReference: T, negate: boolean) => void
    onExampleClick?: (example: string) => void
}

const SearchReferenceEntry = <T extends SearchReferenceInfo>({
    searchReference,
    onClick,
    onExampleClick,
}: SearchReferenceEntryProps<T>): ReactElement | null => {
    const [collapsed, setCollapsed] = useState(true)
    const CollapseIcon = collapsed ? ChevronLeftIcon : ChevronDownIcon

    let buttonTextPrefix: ReactElement | null = null
    if (isFilterInfo(searchReference)) {
        buttonTextPrefix = <span className="search-filter-keyword">{searchReference.type}:</span>
    }

    return (
        <li>
            <span
                className={classNames(styles.item, sidebarStyles.sidebarSectionListItem, {
                    [styles.active]: !collapsed,
                })}
            >
                <button
                    className="btn p-0 flex-1"
                    type="button"
                    onClick={event => onClick(searchReference, event.altKey)}
                >
                    <span className="text-monospace">
                        {buttonTextPrefix}
                        {searchReference.placeholder.tokens.map(token => (
                            <span key={token.start} className={classNameTokenMap[token.type]}>
                                {token.content}
                            </span>
                        ))}
                    </span>
                </button>
                <button
                    type="button"
                    className={classNames('btn btn-icon', styles.collapseButton)}
                    onClick={event => {
                        event.stopPropagation()
                        setCollapsed(collapsed => !collapsed)
                    }}
                    aria-label={collapsed ? 'Show filter description' : 'Hide filter description'}
                >
                    <small className="text-monospace">i</small>
                    <CollapseIcon className="icon-inline" />
                </button>
            </span>
            <Collapse isOpen={!collapsed}>
                <div className={styles.description}>
                    {searchReference.description && (
                        <Markdown dangerousInnerHTML={renderMarkdown(searchReference.description)} />
                    )}
                    {searchReference.alias && (
                        <p>
                            Alias:{' '}
                            <span className="text-code search-filter-keyword">
                                {searchReference.alias}
                                {isFilterInfo(searchReference) ? ':' : ''}
                            </span>
                        </p>
                    )}
                    {isFilterInfo(searchReference) && isNegatableFilter(searchReference.type) && (
                        <p>
                            Negation: <span className="test-code search-filter-keyword">-{searchReference.type}:</span>
                            {searchReference.alias && (
                                <>
                                    {' '}
                                    | <span className="test-code search-filter-keyword">-{searchReference.alias}:</span>
                                </>
                            )}
                            <br />
                            <span className={styles.placeholder}>(opt + click filter in reference list)</span>
                        </p>
                    )}
                    {searchReference.examples && (
                        <>
                            <div className="font-weight-medium">Examples</div>
                            <div className={classNames('text-code', styles.examples)}>
                                {searchReference.examples.map(example => (
                                    <p key={example}>
                                        <SearchReferenceExample example={example} onClick={onExampleClick} />
                                    </p>
                                ))}
                            </div>
                        </>
                    )}
                </div>
            </Collapse>
        </li>
    )
}

interface FilterInfoListProps<T extends SearchReferenceInfo> {
    filters: T[]
    onClick: (info: T, negate: boolean) => void
    onExampleClick: (example: string) => void
}

const FilterInfoList = ({ filters, onClick, onExampleClick }: FilterInfoListProps<FilterInfo>): ReactElement => (
    <ul className={styles.list}>
        {filters.map(filterInfo => (
            <SearchReferenceEntry
                key={filterInfo.type + filterInfo.placeholder.text}
                searchReference={filterInfo}
                onClick={onClick}
                onExampleClick={onExampleClick}
            />
        ))}
    </ul>
)

export interface SearchReferenceProps
    extends Omit<PatternTypeProps, 'setPatternType'>,
        Omit<CaseSensitivityProps, 'setCaseSensitivity'>,
        VersionContextProps,
        TelemetryProps,
        Pick<SearchContextProps, 'selectedSearchContextSpec'> {
    query: string
    filter: string
    navbarSearchQueryState: QueryState
    onNavbarQueryChange: (queryState: QueryState) => void
    isSourcegraphDotCom: boolean
}

const SearchReference = (props: SearchReferenceProps): ReactElement => {
    const [selectedTab, setSelectedTab] = useLocalStorage(SEARCH_REFERENCE_TAB_KEY, 0)

    const { onNavbarQueryChange, navbarSearchQueryState, filter, telemetryService } = props
    const hasFilter = filter.length === 0

    const selectedFilters = useMemo(() => {
        if (!hasFilter) {
            return filterInfos
        }
        const searchTerms = parseSearchInput(filter)
        return filterInfos.filter(info => matches(searchTerms, info))
    }, [filter, hasFilter])

    const updateQuery = useCallback(
        (searchReference: FilterInfo, negate: boolean) => {
            onNavbarQueryChange(updateQueryWithFilter(navbarSearchQueryState, searchReference, negate, FILTERS))
        },
        [onNavbarQueryChange, navbarSearchQueryState]
    )
    const updateQueryWithOperator = useCallback(
        (info: OperatorInfo) => {
            onNavbarQueryChange({
                query: navbarSearchQueryState.query + ` ${info.operator} `,
            })
        },
        [onNavbarQueryChange, navbarSearchQueryState]
    )
    const updateQueryWithExample = useCallback(
        (example: string) => {
            telemetryService.log(hasFilter ? 'SearchReferenceSearchedAndClicked' : 'SearchReferenceFilterClicked')
            onNavbarQueryChange({ query: navbarSearchQueryState.query.trimEnd() + ' ' + example })
        },
        [onNavbarQueryChange, navbarSearchQueryState, hasFilter, telemetryService]
    )

    const filterList = (
        <FilterInfoList filters={selectedFilters} onClick={updateQuery} onExampleClick={updateQueryWithExample} />
    )

    return (
        <div>
            {props.filter ? (
                filterList
            ) : (
                <Tabs index={selectedTab} onChange={setSelectedTab}>
                    <TabList className={styles.tablist}>
                        <Tab>Common</Tab>
                        <Tab>All filters</Tab>
                        <Tab>Operators</Tab>
                    </TabList>
                    <TabPanels>
                        <TabPanel>
                            <FilterInfoList
                                filters={commonFilters}
                                onClick={updateQuery}
                                onExampleClick={updateQueryWithExample}
                            />
                        </TabPanel>
                        <TabPanel>{filterList}</TabPanel>
                        <TabPanel>
                            <ul className={styles.list}>
                                {operatorInfo.map(operatorInfo => (
                                    <SearchReferenceEntry
                                        searchReference={operatorInfo}
                                        key={operatorInfo.operator + operatorInfo.placeholder.text}
                                        onClick={updateQueryWithOperator}
                                        onExampleClick={updateQueryWithExample}
                                    />
                                ))}
                            </ul>
                        </TabPanel>
                    </TabPanels>
                </Tabs>
            )}
            <p className={styles.footer}>
                <small>
                    <Link target="blank" to="https://docs.sourcegraph.com/code_search/reference/queries">
                        Search syntax <ExternalLinkIcon className="icon-inline" />
                    </Link>
                </small>
            </p>
        </div>
    )
}

export function getSearchReferenceFactory(
    props: Omit<SearchReferenceProps, 'filter'>
): (filter: string) => ReactElement {
    return (filter: string) => <SearchReference {...props} filter={filter} />
}
