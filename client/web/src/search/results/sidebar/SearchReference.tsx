import { Tab, TabList, TabPanel, TabPanels, Tabs } from '@reach/tabs'
import classNames from 'classnames'
import { escapeRegExp } from 'lodash'
import ChevronDownIcon from 'mdi-react/ChevronDownIcon'
import ChevronLeftIcon from 'mdi-react/ChevronLeftIcon'
import ExternalLinkIcon from 'mdi-react/ExternalLinkIcon'
import { Range, Selection, SelectionDirection } from 'monaco-editor'
import React, { ReactElement, useCallback, useMemo, useState } from 'react'
import { Collapse } from 'reactstrap'

import { Link } from '@sourcegraph/shared/src/components/Link'
import { Markdown } from '@sourcegraph/shared/src/components/Markdown'
import { FILTERS, FilterType, isNegatableFilter } from '@sourcegraph/shared/src/search/query/filters'
import { parseSearchQuery } from '@sourcegraph/shared/src/search/query/parser'
import { appendFilter } from '@sourcegraph/shared/src/search/query/transformer'
import { findFilter, FilterKind } from '@sourcegraph/shared/src/search/query/validate'
import { VersionContextProps } from '@sourcegraph/shared/src/search/util'
import { renderMarkdown } from '@sourcegraph/shared/src/util/markdown'
import { useLocalStorage } from '@sourcegraph/shared/src/util/useLocalStorage'

import { CaseSensitivityProps, PatternTypeProps, SearchContextProps } from '../..'
import { QueryChangeSource, QueryState } from '../../helpers'

import styles from './SearchReference.module.scss'
import sidebarStyles from './SearchSidebarSection.module.scss'

const SEARCH_REFERENCE_TAB_KEY = 'SearchProduct.SearchReference.Tab'

export interface SearchReferenceInfo {
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

/**
 * Adds additional search reference information from the existing filters list.
 */
function augmentSearchReference(searchReference: SearchReferenceInfo): void {
    const filter = FILTERS[searchReference.type]
    if (filter?.alias) {
        searchReference.alias = filter.alias
    }
}

const searchReferenceInfo: SearchReferenceInfo[] = [
    {
        type: FilterType.after,
        placeholder: parsePlaceholder('"{last week}"'),
        description:
            'Only include results from diffs or commits which have a commit date after the specified time frame.',
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
        type: FilterType.case,
        placeholder: parsePlaceholder('{yes}'),
        description: 'Perform a case sensitive query. Without this, everything is matched case insensitively.',
        examples: ['OPEN_FILE case:yes'],
    },
    {
        type: FilterType.content,
        placeholder: parsePlaceholder('"{pattern}"'),
        description:
            'Set the search pattern with a dedicated parameter. Useful when searching literally for a string that may conflict with the [search pattern syntax](https://docs.sourcegraph.com/code_search/reference/queries#search-pattern-syntax). In between the quotes, the `\\` character will need to be escaped (`\\\\` to evaluate for `\\`).',
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
        type: FilterType.repo,
        placeholder: parsePlaceholder('{regexp-pattern}'),
        description:
            'Only include results from repositories whose path matches the regexp-pattern. A repository’s path is a string such as *github.com/myteam/abc* or *code.example.com/xyz* that depends on your organization’s repository host. If the regexp ends in [`@rev`](https://docs.sourcegraph.com/code_search/reference/queries#repository-revisions), that revision is searched instead of the default branch (usually `master`). `repo:regexp-pattern@rev` is equivalent to `repo:regexp-pattern rev:rev`.',
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
        placeholder: parsePlaceholder('contains.{file/content/commit}'),
        description:
            'Conditionally search inside repositories only if contain certain files or commits after some specified time. See [git date formats](https://github.com/git/git/blob/master/Documentation/date-formats.txt) for accepted formats.',
        examples: [
            'repo:contains.commit.after(yesterday)',
            'repo:contains.commit.after(june 25 2017)',
            'repo:contains.file(.py) file:Dockerfile pip',
        ],
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
        commonRank: 90,
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
        placeholder: parsePlaceholder('{any}'),
        description:
            'Filter results to only public or private repositories. The default is to include both private and public repositories.',
        examples: ['type:repo visibility:public'],
    },
]

for (const searchReference of searchReferenceInfo) {
    augmentSearchReference(searchReference)
}

const commonFilters = searchReferenceInfo
    .filter(info => info.commonRank !== undefined)
    // commonRank will never be undefined here, but TS doesn't seem to know
    .sort((a, b) => (a.commonRank as number) - (b.commonRank as number))

/**
 * Returns true if the provided regular expressions all match the provided
 * filter information (name, description, ...)
 */
function matches(searchTerms: RegExp[], info: SearchReferenceInfo): boolean {
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
    direction: SelectionDirection = SelectionDirection.LTR
): Selection {
    const token = placeholder.tokens.find(token => token.type === 'value')
    if (!token) {
        throw new Error('Search reference does not contain placeholder.')
    }
    return Selection.createWithDirection(1, offset + token.start + 1, 1, offset + token.end, direction)
}

/**
 * Whether or not to trigger the suggestion popover when adding this filter to
 * the query.
 */
function shouldShowSuggestions(searchReference: SearchReferenceInfo): boolean {
    return Boolean(searchReference.showSuggestions !== false && FILTERS[searchReference.type].discreteValues)
}

/**
 * This helper function will update the current query with the provided filter,
 * updating existing filters if necessary.
 * exported only for test purposes
 */
export function updateQueryWithFilter(
    currentQueryState: QueryState,
    searchReference: SearchReferenceInfo,
    negate: boolean,
    allFilters: typeof FILTERS
): QueryState {
    const { singular } = allFilters[searchReference.type]
    let { query } = currentQueryState
    const showSuggestions = shouldShowSuggestions(searchReference)
    let selection: Selection | undefined
    let revealRange: Range
    let field: string = searchReference.type

    if (negate && isNegatableFilter(searchReference.type)) {
        field = '-' + field
    }

    const existingFilter = findFilter(query, searchReference.type, FilterKind.Global)

    if (existingFilter && singular) {
        // Filter can only appear once
        // Select the existing filter value or append the filter and select the
        // placeholder
        selection = Selection.createWithDirection(
            1,
            (existingFilter.value?.range.start || existingFilter.field.range.end) + 1,
            1,
            existingFilter.range.end + 1,
            showSuggestions ? SelectionDirection.RTL : SelectionDirection.LTR
        )
        revealRange = new Range(1, existingFilter.range.start + 1, 1, existingFilter.range.end + 1)
    } else {
        // Filter can appear multiple times or doesn't exist yet. Always
        // append.

        // +1 because appendFilter inserts a whitespace character at the end of
        // the query
        const rangeStart = query.length + 1
        query = appendFilter(query, field, searchReference.placeholder.text)
        selection = selectionForPlaceholder(
            searchReference.placeholder,
            query.length - searchReference.placeholder.text.length,
            // If we need to trigger the suggestion popover we have to make
            // sure the input cursor is positioned at the beginning of the
            // selection (it usually is at the end)
            showSuggestions ? SelectionDirection.RTL : SelectionDirection.LTR
        )
        revealRange = new Range(1, rangeStart + 1, 1, query.length + 1)
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

function interleave<T>(values: T[], filler: T): T[] {
    const result = []
    if (values.length > 0) {
        result.push(values[0])
    }
    for (let index = 1; index < values.length; index++) {
        result.push(filler)
        result.push(values[index])
    }

    return result
}

const classNameTokenMap = {
    text: 'search-filter-keyword',
    value: styles.placeholder,
}

interface SearchReferenceExampleProps {
    example: string
}

const SearchReferenceExample: React.FunctionComponent<SearchReferenceExampleProps> = ({ example }) => {
    const parseResult = parseSearchQuery(example)
    // We only use valid queries as examples, so this will always be true
    if (parseResult.type === 'success') {
        return (
            <>
                {interleave(
                    parseResult.nodes
                        .map(node => {
                            switch (node.type) {
                                case 'parameter':
                                    return (
                                        <>
                                            <span className="search-filter-keyword">{node.field}:</span>
                                            {node.value}
                                        </>
                                    )
                                case 'pattern':
                                    return node.value
                                case 'operator':
                                    // we currently don't use operators in examples,
                                    // but we need an entry to make TS happy. Once
                                    // we do support operators, the query needs to
                                    // be parsed/rendered differently
                                    return node.kind
                            }
                        })
                        .filter(Boolean),
                    ' '
                )}
            </>
        )
    }
    return null
}

interface SearchReferenceEntryProps {
    searchReference: SearchReferenceInfo
    onClick: (searchReference: SearchReferenceInfo, negate: boolean) => void
}

const SearchReferenceEntry: React.FunctionComponent<SearchReferenceEntryProps> = ({ searchReference, onClick }) => {
    const [collapsed, setCollapsed] = useState(true)
    const CollapseIcon = collapsed ? ChevronLeftIcon : ChevronDownIcon

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
                        <span className="search-filter-keyword">{searchReference.type}:</span>
                        {searchReference.placeholder.tokens.map(token => (
                            <span key={token.start} className={classNameTokenMap[token.type]}>
                                {token.content}
                            </span>
                        ))}
                    </span>
                </button>
                <button
                    type="button"
                    className={classNames('btn btn-icon mr-1', styles.collapseButton)}
                    onClick={event => {
                        event.stopPropagation()
                        setCollapsed(collapsed => !collapsed)
                    }}
                    aria-label={collapsed ? 'Show filter description' : 'Hide filter description'}
                >
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
                            Alias: <span className="text-code search-filter-keyword">{searchReference.alias}:</span>
                        </p>
                    )}
                    {isNegatableFilter(searchReference.type) && (
                        <p>
                            Negation: <span className="test-code search-filter-keyword">-{searchReference.type}:</span>
                            {searchReference.alias && (
                                <>
                                    {' '}
                                    | <span className="test-code search-filter-keyword">-{searchReference.alias}:</span>
                                </>
                            )}
                        </p>
                    )}
                    {searchReference.examples && (
                        <>
                            <div className="font-weight-medium">Examples</div>
                            <div className={classNames('text-code', styles.examples)}>
                                {searchReference.examples.map(example => (
                                    <p key={example}>
                                        <SearchReferenceExample example={example} />
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

interface SearchReferenceListProps {
    filters: SearchReferenceInfo[]
    onClick: (info: SearchReferenceInfo, negate: boolean) => void
}

const SearchReferenceList = ({ filters, onClick }: SearchReferenceListProps): ReactElement => (
    <ul className={styles.list}>
        {filters.map(filterInfo => (
            <SearchReferenceEntry
                searchReference={filterInfo}
                key={filterInfo.type + filterInfo.placeholder.text}
                onClick={onClick}
            />
        ))}
    </ul>
)

export interface SearchReferenceProps
    extends Omit<PatternTypeProps, 'setPatternType'>,
        Omit<CaseSensitivityProps, 'setCaseSensitivity'>,
        VersionContextProps,
        Pick<SearchContextProps, 'selectedSearchContextSpec'> {
    query: string
    filter: string
    navbarSearchQueryState: QueryState
    onNavbarQueryChange: (queryState: QueryState) => void
    isSourcegraphDotCom: boolean
}

const SearchReference = (props: SearchReferenceProps): ReactElement => {
    const [selectedTab, setSelectedTab] = useLocalStorage(SEARCH_REFERENCE_TAB_KEY, 0)

    const selectedFilters = useMemo(() => {
        if (props.filter === '') {
            return searchReferenceInfo
        }
        const searchTerms = parseSearchInput(props.filter)
        return searchReferenceInfo.filter(info => matches(searchTerms, info))
    }, [props.filter])

    const { onNavbarQueryChange, navbarSearchQueryState } = props
    const updateQuery = useCallback(
        (searchReference: SearchReferenceInfo, negate: boolean) => {
            onNavbarQueryChange(updateQueryWithFilter(navbarSearchQueryState, searchReference, negate, FILTERS))
        },
        [onNavbarQueryChange, navbarSearchQueryState]
    )

    const filterList = <SearchReferenceList filters={selectedFilters} onClick={updateQuery} />

    return (
        <div>
            {props.filter ? (
                filterList
            ) : (
                <Tabs index={selectedTab} onChange={setSelectedTab}>
                    <TabList className={styles.tablist}>
                        <Tab>Common</Tab>
                        <Tab>All filters</Tab>
                    </TabList>
                    <TabPanels>
                        <TabPanel>
                            <SearchReferenceList filters={commonFilters} onClick={updateQuery} />
                        </TabPanel>
                        <TabPanel>{filterList}</TabPanel>
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
