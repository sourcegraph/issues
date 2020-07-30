import * as H from 'history'
import { isEqual } from 'lodash'
import * as React from 'react'
import { concat, Observable, Subject, Subscription } from 'rxjs'
import { catchError, distinctUntilChanged, filter, map, startWith, switchMap, tap } from 'rxjs/operators'
import {
    parseSearchURLQuery,
    parseSearchURLPatternType,
    PatternTypeProps,
    InteractiveSearchProps,
    CaseSensitivityProps,
    parseSearchURL,
    resolveVersionContext,
} from '..'
import { Contributions, Evaluated } from '../../../../shared/src/api/protocol'
import { FetchFileCtx } from '../../../../shared/src/components/CodeExcerpt'
import { ExtensionsControllerProps } from '../../../../shared/src/extensions/controller'
import * as GQL from '../../../../shared/src/graphql/schema'
import { PlatformContextProps } from '../../../../shared/src/platform/context'
import { isSettingsValid, SettingsCascadeProps } from '../../../../shared/src/settings/settings'
import { TelemetryProps } from '../../../../shared/src/telemetry/telemetryService'
import { ErrorLike, isErrorLike, asError } from '../../../../shared/src/util/errors'
import { PageTitle } from '../../components/PageTitle'
import { Settings } from '../../schema/settings.schema'
import { ThemeProps } from '../../../../shared/src/theme'
import { EventLogger } from '../../tracking/eventLogger'
import { isSearchResults, submitSearch, toggleSearchFilter, getSearchTypeFromQuery, QueryState } from '../helpers'
import { queryTelemetryData } from '../queryTelemetry'
import { SearchResultsFilterBars, SearchScopeWithOptionalName } from './SearchResultsFilterBars'
import { SearchResultsList } from './SearchResultsList'
import { SearchResultTypeTabs } from './SearchResultTypeTabs'
import { buildSearchURLQuery } from '../../../../shared/src/util/url'
import { convertPlainTextToInteractiveQuery } from '../input/helpers'
import { VersionContextProps } from '../../../../shared/src/search/util'
import { VersionContext } from '../../schema/site.schema'
import AlertOutlineIcon from 'mdi-react/AlertOutlineIcon'
import CloseIcon from 'mdi-react/CloseIcon'
import { Remote } from 'comlink'
import { FlatExtHostAPI } from '../../../../shared/src/api/contract'
import { DeployType } from '../../jscontext'
import { OptionalAuthProps } from '../../auth'

export interface SearchResultsProps
    extends ExtensionsControllerProps<'executeCommand' | 'extHostAPI' | 'services'>,
        PlatformContextProps<'forceUpdateTooltip' | 'settings'>,
        SettingsCascadeProps,
        TelemetryProps,
        ThemeProps,
        PatternTypeProps,
        CaseSensitivityProps,
        InteractiveSearchProps,
        VersionContextProps,
        OptionalAuthProps {
    location: H.Location
    history: H.History
    navbarSearchQueryState: QueryState
    telemetryService: Pick<EventLogger, 'log' | 'logViewEvent'>
    fetchHighlightedFileLines: (ctx: FetchFileCtx, force?: boolean) => Observable<string[]>
    searchRequest: (
        query: string,
        version: string,
        patternType: GQL.SearchPatternType,
        versionContext: string | undefined,
        extensionHostPromise: Promise<Remote<FlatExtHostAPI>>
    ) => Observable<GQL.SearchResults | ErrorLike>
    isSourcegraphDotCom: boolean
    deployType: DeployType
    setVersionContext: (versionContext: string | undefined) => void
    availableVersionContexts: VersionContext[] | undefined
    previousVersionContext: string | null
}

interface SearchResultsState {
    /** The loaded search results, error or undefined while loading */
    resultsOrError?: GQL.SearchResults
    allExpanded: boolean

    // Saved Queries
    showSavedQueryModal: boolean
    didSaveQuery: boolean

    /** The contributions, merged from all extensions, or undefined before the initial emission. */
    contributions?: Evaluated<Contributions>

    /** Whether to show a warning saying that the URL has changed the version context. */
    showVersionContextWarning: boolean

    /** Whether the user has dismissed the version context warning. */
    dismissedVersionContextWarning?: boolean
}

/** All values that are valid for the `type:` filter. `null` represents default code search. */
export type SearchType = 'diff' | 'commit' | 'symbol' | 'repo' | 'path' | null

// The latest supported version of our search syntax. Users should never be able to determine the search version.
// The version is set based on the release tag of the instance. Anything before 3.9.0 will not pass a version parameter,
// and will therefore default to V1.
const LATEST_VERSION = 'V2'

export class SearchResults extends React.Component<SearchResultsProps, SearchResultsState> {
    public state: SearchResultsState = {
        didSaveQuery: false,
        showSavedQueryModal: false,
        allExpanded: false,
        showVersionContextWarning: false,
    }
    /** Emits on componentDidUpdate with the new props */
    private componentUpdates = new Subject<SearchResultsProps>()

    private subscriptions = new Subscription()

    public componentDidMount(): void {
        const patternType = parseSearchURLPatternType(this.props.location.search)

        if (!patternType) {
            // If the patternType query parameter does not exist in the URL or is invalid, redirect to a URL which
            // has patternType=regexp appended. This is to ensure old URLs before requiring patternType still work.

            const query = parseSearchURLQuery(this.props.location.search) || ''
            const { navbarQuery, filtersInQuery } = convertPlainTextToInteractiveQuery(query)
            const newLocation =
                '/search?' +
                buildSearchURLQuery(
                    navbarQuery,
                    GQL.SearchPatternType.regexp,
                    this.props.caseSensitive,
                    this.props.versionContext,
                    filtersInQuery
                )
            this.props.history.replace(newLocation)
        }

        this.props.telemetryService.logViewEvent('SearchResults')

        this.subscriptions.add(
            this.componentUpdates
                .pipe(
                    startWith(this.props),
                    map(props => parseSearchURL(props.location.search)),
                    // Search when a new search query was specified in the URL
                    distinctUntilChanged((a, b) => isEqual(a, b)),
                    filter(
                        (
                            queryAndPatternTypeAndCase
                        ): queryAndPatternTypeAndCase is {
                            query: string
                            patternType: GQL.SearchPatternType
                            caseSensitive: boolean
                            versionContext: string | undefined
                        } => !!queryAndPatternTypeAndCase.query && !!queryAndPatternTypeAndCase.patternType
                    ),
                    tap(({ query, caseSensitive }) => {
                        const query_data = queryTelemetryData(query, caseSensitive)
                        this.props.telemetryService.log('SearchResultsQueried', {
                            code_search: { query_data },
                            ...(this.props.splitSearchModes
                                ? { mode: this.props.interactiveSearchMode ? 'interactive' : 'plain' }
                                : {}),
                        })
                        if (query_data.query?.field_type && query_data.query.field_type.value_diff > 0) {
                            this.props.telemetryService.log('DiffSearchResultsQueried')
                        }
                    }),
                    switchMap(({ query, patternType, caseSensitive, versionContext }) =>
                        concat(
                            // Reset view state
                            [
                                {
                                    resultsOrError: undefined,
                                    didSave: false,
                                    activeType: getSearchTypeFromQuery(query),
                                },
                            ],
                            // Do async search request
                            this.props
                                .searchRequest(
                                    caseSensitive ? `${query} case:yes` : query,
                                    LATEST_VERSION,
                                    patternType,
                                    resolveVersionContext(versionContext, this.props.availableVersionContexts),
                                    this.props.extensionsController.extHostAPI
                                )
                                .pipe(
                                    // Log telemetry
                                    tap(
                                        results => {
                                            this.props.telemetryService.log('SearchResultsFetched', {
                                                code_search: {
                                                    // 🚨 PRIVACY: never provide any private data in { code_search: { results } }.
                                                    results: {
                                                        results_count: isErrorLike(results)
                                                            ? 0
                                                            : results.results.length,
                                                        any_cloning: isErrorLike(results)
                                                            ? false
                                                            : results.cloning.length > 0,
                                                    },
                                                },
                                            })
                                            if (patternType && patternType !== this.props.patternType) {
                                                this.props.setPatternType(patternType)
                                            }
                                            if (caseSensitive !== this.props.caseSensitive) {
                                                this.props.setCaseSensitivity(caseSensitive)
                                            }

                                            this.props.setVersionContext(versionContext)
                                        },
                                        error => {
                                            this.props.telemetryService.log('SearchResultsFetchFailed', {
                                                code_search: { error_message: asError(error).message },
                                            })
                                            console.error(error)
                                        }
                                    ),
                                    // Update view with results or error
                                    map(resultsOrError => ({ resultsOrError })),
                                    catchError(error => [{ resultsOrError: error }])
                                )
                        )
                    )
                )
                .subscribe(
                    newState => this.setState(newState as SearchResultsState),
                    error => console.error(error)
                )
        )

        this.subscriptions.add(
            this.componentUpdates
                .pipe(
                    startWith(this.props),
                    distinctUntilChanged((a, b) => isEqual(a.location, b.location))
                )
                .subscribe(props => {
                    const searchParameters = new URLSearchParams(props.location.search)
                    const versionFromURL = searchParameters.get('c')

                    if (searchParameters.has('from-context-toggle')) {
                        // The query param `from-context-toggle` indicates that the version context
                        // changed from the version context toggle. In this case, we don't warn
                        // users that the version context has changed.
                        searchParameters.delete('from-context-toggle')
                        this.props.history.replace({
                            search: searchParameters.toString(),
                            hash: this.props.history.location.hash,
                        })
                        this.setState({ showVersionContextWarning: false })
                    } else {
                        this.setState({
                            showVersionContextWarning:
                                (props.availableVersionContexts && versionFromURL !== props.previousVersionContext) ||
                                false,
                        })
                    }
                })
        )

        this.subscriptions.add(
            this.props.extensionsController.services.contribution
                .getContributions()
                .subscribe(contributions => this.setState({ contributions }))
        )
    }

    public componentDidUpdate(): void {
        this.componentUpdates.next(this.props)
    }

    public componentWillUnmount(): void {
        this.subscriptions.unsubscribe()
    }

    private showSaveQueryModal = (): void => {
        this.setState({ showSavedQueryModal: true, didSaveQuery: false })
    }

    private onDidCreateSavedQuery = (): void => {
        this.props.telemetryService.log('SavedQueryCreated')
        this.setState({ showSavedQueryModal: false, didSaveQuery: true })
    }

    private onModalClose = (): void => {
        this.props.telemetryService.log('SavedQueriesToggleCreating', { queries: { creating: false } })
        this.setState({ didSaveQuery: false, showSavedQueryModal: false })
    }

    private onDismissWarning = (): void => {
        this.setState({ showVersionContextWarning: false })
    }

    public render(): JSX.Element | null {
        const query = parseSearchURLQuery(this.props.location.search)
        const filters = this.getFilters()
        const extensionFilters = this.state.contributions?.searchFilters

        const quickLinks =
            (isSettingsValid<Settings>(this.props.settingsCascade) && this.props.settingsCascade.final.quicklinks) || []

        return (
            <div className="test-search-results search-results d-flex flex-column w-100">
                <PageTitle key="page-title" title={query} />
                {!this.props.interactiveSearchMode && (
                    <SearchResultsFilterBars
                        navbarSearchQuery={this.props.navbarSearchQueryState.query}
                        results={this.state.resultsOrError}
                        filters={filters}
                        extensionFilters={extensionFilters}
                        quickLinks={quickLinks}
                        onFilterClick={this.onDynamicFilterClicked}
                        onShowMoreResultsClick={this.showMoreResults}
                        calculateShowMoreResultsCount={this.calculateCount}
                    />
                )}
                {this.state.showVersionContextWarning && (
                    <div className="mt-2 mx-2">
                        <div className="d-flex alert alert-warning mb-0 justify-content-between">
                            <div>
                                <AlertOutlineIcon className="icon-inline mr-2" />
                                This link changed your version context to{' '}
                                <strong>{this.props.versionContext || 'default'}</strong>. You can switch contexts with
                                the selector to the left of the search bar.
                            </div>
                            <div onClick={this.onDismissWarning}>
                                <CloseIcon className="icon-inline ml-2" />
                            </div>
                        </div>
                    </div>
                )}
                <SearchResultTypeTabs
                    {...this.props}
                    query={this.props.navbarSearchQueryState.query}
                    filtersInQuery={this.props.filtersInQuery}
                />
                <SearchResultsList
                    {...this.props}
                    resultsOrError={this.state.resultsOrError}
                    onShowMoreResultsClick={this.showMoreResults}
                    onExpandAllResultsToggle={this.onExpandAllResultsToggle}
                    allExpanded={this.state.allExpanded}
                    showSavedQueryModal={this.state.showSavedQueryModal}
                    onSaveQueryClick={this.showSaveQueryModal}
                    onSavedQueryModalClose={this.onModalClose}
                    onDidCreateSavedQuery={this.onDidCreateSavedQuery}
                    didSave={this.state.didSaveQuery}
                />
            </div>
        )
    }

    /** Combines dynamic filters and search scopes into a list de-duplicated by value. */
    private getFilters(): SearchScopeWithOptionalName[] {
        const filters = new Map<string, SearchScopeWithOptionalName>()

        if (isSearchResults(this.state.resultsOrError) && this.state.resultsOrError.dynamicFilters) {
            let dynamicFilters = this.state.resultsOrError.dynamicFilters
            dynamicFilters = this.state.resultsOrError.dynamicFilters.filter(filter => filter.kind !== 'repo')
            for (const filter of dynamicFilters) {
                filters.set(filter.value, filter)
            }
        }
        const scopes =
            (isSettingsValid<Settings>(this.props.settingsCascade) &&
                this.props.settingsCascade.final['search.scopes']) ||
            []
        if (isSearchResults(this.state.resultsOrError) && this.state.resultsOrError.dynamicFilters) {
            for (const scope of scopes) {
                if (!filters.has(scope.value)) {
                    filters.set(scope.value, scope)
                }
            }
        } else {
            for (const scope of scopes) {
                // Check for if filter.value already exists and if so, overwrite with user's configured scope name.
                const existingFilter = filters.get(scope.value)
                // This works because user setting configs are the last to be processed after Global and Org.
                // Thus, user set filters overwrite the equal valued existing filters.
                if (existingFilter) {
                    existingFilter.name = scope.name || scope.value
                }
                filters.set(scope.value, existingFilter || scope)
            }
        }

        return [...filters.values()]
    }
    private showMoreResults = (): void => {
        // Requery with an increased max result count.
        const parameters = new URLSearchParams(this.props.location.search)
        let query = parameters.get('q') || ''

        const count = this.calculateCount()
        if (/count:(\d+)/.test(query)) {
            query = query.replace(/count:\d+/g, '').trim() + ` count:${count}`
        } else {
            query = `${query} count:${count}`
        }
        parameters.set('q', query)
        this.props.history.replace({ search: parameters.toString() })
    }

    private calculateCount = (): number => {
        // This function can only get called if the results were successfully loaded,
        // so casting is the right thing to do here
        const results = this.state.resultsOrError as GQL.SearchResults

        const parameters = new URLSearchParams(this.props.location.search)
        const query = parameters.get('q') || ''

        if (/count:(\d+)/.test(query)) {
            return Math.max(results.matchCount * 2, 1000)
        }
        return Math.max(results.matchCount * 2 || 0, 1000)
    }

    private onExpandAllResultsToggle = (): void => {
        this.setState(
            state => ({ allExpanded: !state.allExpanded }),
            () => {
                this.props.telemetryService.log(this.state.allExpanded ? 'allResultsExpanded' : 'allResultsCollapsed')
            }
        )
    }

    private onDynamicFilterClicked = (value: string): void => {
        this.props.telemetryService.log('DynamicFilterClicked', {
            search_filter: { value },
        })

        const newQuery = toggleSearchFilter(this.props.navbarSearchQueryState.query, value)

        submitSearch({ ...this.props, query: newQuery, source: 'filter' })
    }
}
