import * as H from 'history'
import React, { useEffect, useState, useCallback } from 'react'
import { Link } from 'react-router-dom'
import { concat, of, timer } from 'rxjs'
import { debounce, delay, map, switchMap, takeUntil, tap, distinctUntilChanged } from 'rxjs/operators'

import { Form } from '@sourcegraph/branded/src/components/Form'
import { ConfiguredRegistryExtension } from '@sourcegraph/shared/src/extensions/extension'
import { gql } from '@sourcegraph/shared/src/graphql/graphql'
import { PlatformContextProps } from '@sourcegraph/shared/src/platform/context'
import { ExtensionCategory, EXTENSION_CATEGORIES } from '@sourcegraph/shared/src/schema/extensionSchema'
import { Settings, SettingsCascadeProps, SettingsCascadeOrError } from '@sourcegraph/shared/src/settings/settings'
import { ThemeProps } from '@sourcegraph/shared/src/theme'
import { createAggregateError, ErrorLike, isErrorLike } from '@sourcegraph/shared/src/util/errors'
import { useEventObservable } from '@sourcegraph/shared/src/util/useObservable'

import { PageTitle } from '../components/PageTitle'
import {
    RegistryExtensionsResult,
    RegistryExtensionFieldsForList,
    RegistryExtensionsVariables,
} from '../graphql-operations'
import { eventLogger } from '../tracking/eventLogger'

import { ExtensionBanner } from './ExtensionBanner'
import { ExtensionRegistrySidenav } from './ExtensionRegistrySidenav'
import { configureExtensionRegistry, ConfiguredExtensionRegistry } from './extensions'
import { ExtensionsAreaRouteContext } from './ExtensionsArea'
import { ExtensionsList } from './ExtensionsList'

interface Props
    extends Pick<ExtensionsAreaRouteContext, 'authenticatedUser' | 'subject'>,
        PlatformContextProps<'settings' | 'updateSettings' | 'requestGraphQL'>,
        SettingsCascadeProps,
        ThemeProps {
    location: H.Location
    history: H.History
}

const LOADING = 'loading' as const
const URL_QUERY_PARAM = 'query'
const URL_CATEGORY_PARAM = 'category'

export type ExtensionListData = typeof LOADING | (ConfiguredExtensionRegistry & { error: string | null }) | ErrorLike

export type ExtensionsEnablement = 'all' | 'enabled' | 'disabled'

const extensionRegistryQuery = gql`
    query RegistryExtensions($query: String, $prioritizeExtensionIDs: [String!]!) {
        extensionRegistry {
            extensions(query: $query, prioritizeExtensionIDs: $prioritizeExtensionIDs) {
                nodes {
                    ...RegistryExtensionFieldsForList
                }
                error
            }
        }
    }
    fragment RegistryExtensionFieldsForList on RegistryExtension {
        id
        publisher {
            __typename
            ... on User {
                id
                username
                displayName
                url
            }
            ... on Org {
                id
                name
                displayName
                url
            }
        }
        extensionID
        extensionIDWithoutRegistry
        name
        manifest {
            raw
            description
        }
        createdAt
        updatedAt
        url
        remoteURL
        registryName
        isLocal
        isWorkInProgress
        viewerCanAdminister
    }
`

export type ConfiguredExtensionCache = Map<
    string,
    Pick<ConfiguredRegistryExtension<RegistryExtensionFieldsForList>, 'manifest' | 'id'>
>

/** A page that displays overview information about the available extensions. */
export const ExtensionRegistry: React.FunctionComponent<Props> = props => {
    useEffect(() => eventLogger.logViewEvent('ExtensionsOverview'), [])

    const { history, location, settingsCascade, platformContext, authenticatedUser } = props

    // Update the cache after each response. This speeds up client-side filtering.
    // Lazy initialize cache ref. Don't mind the `useState` abuse:
    // - can't use `useMemo` here (https://github.com/facebook/react/issues/14490#issuecomment-454973512)
    // - `useRef` is just `useState`
    const [configuredExtensionCache] = useState<ConfiguredExtensionCache>(
        () => new Map<string, ConfiguredRegistryExtension<RegistryExtensionFieldsForList>>()
    )

    const [query, setQuery] = useState(getQueryFromLocation(location))

    const [selectedCategory, setSelectedCategory] = useState<ExtensionCategory | 'All'>(
        getCategoryFromLocation(location) || 'All'
    )

    // Filter extensions by enablement state: enabled, disabled, or all.
    const [enablementFilter, setEnablementFilter] = useState<ExtensionsEnablement>('all')
    // Programming language extensions are hidden by default. Users cannot un-show PL extensions once toggled.
    const [showMoreExtensions, setShowMoreExtensions] = useState(false)

    /**
     * Note: pass `settingsCascade` instead of making it a dependency to prevent creating
     * new subscriptions when user toggles extensions
     */
    const [nextQueryInput, data] = useEventObservable<
        {
            query: string
            category: ExtensionCategory | 'All'
            immediate: boolean
            settingsCascade: SettingsCascadeOrError<Settings>
        },
        ExtensionListData
    >(
        useCallback(
            newQueries =>
                newQueries.pipe(
                    distinctUntilChanged(
                        (previous, current) =>
                            previous.query === current.query && previous.category === current.category
                    ),
                    tap(({ query, category }) => {
                        setQuery(query)
                        setSelectedCategory(category)

                        history.replace({
                            search: new URLSearchParams(
                                query
                                    ? {
                                          [URL_QUERY_PARAM]: query,
                                          [URL_CATEGORY_PARAM]: category,
                                      }
                                    : {
                                          [URL_CATEGORY_PARAM]: category,
                                      }
                            ).toString(),
                            hash: window.location.hash,
                        })
                    }),
                    debounce(({ immediate }) => timer(immediate ? 0 : 50)),
                    distinctUntilChanged(
                        (previous, current) =>
                            previous.query === current.query && previous.category === current.category
                    ),
                    switchMap(({ query, category, immediate, settingsCascade }) => {
                        let viewerConfiguredExtensions: string[] = []
                        if (!isErrorLike(settingsCascade.final)) {
                            if (settingsCascade.final?.extensions) {
                                viewerConfiguredExtensions = Object.keys(settingsCascade.final.extensions)
                            }
                        }

                        if (category !== 'All') {
                            query = `${query} category:"${category}"`
                        }

                        const resultOrError = platformContext.requestGraphQL<
                            RegistryExtensionsResult,
                            RegistryExtensionsVariables
                        >({
                            request: extensionRegistryQuery,
                            variables: { query, prioritizeExtensionIDs: viewerConfiguredExtensions },
                            mightContainPrivateInfo: true,
                        })

                        return concat(
                            of(LOADING).pipe(delay(immediate ? 0 : 250), takeUntil(resultOrError)),
                            resultOrError
                        )
                    }),
                    map(resultOrErrorOrLoading => {
                        if (resultOrErrorOrLoading === LOADING) {
                            return resultOrErrorOrLoading
                        }

                        const { data, errors } = resultOrErrorOrLoading

                        if (!data?.extensionRegistry?.extensions) {
                            return createAggregateError(errors)
                        }

                        const { error, nodes } = data.extensionRegistry.extensions

                        return {
                            error,
                            ...configureExtensionRegistry(nodes, configuredExtensionCache),
                        }
                    })
                ),
            [platformContext, history, configuredExtensionCache]
        )
    )

    const onQueryChangeEvent = useCallback(
        (event: React.FormEvent<HTMLInputElement>) =>
            nextQueryInput({
                query: event.currentTarget.value,
                category: getCategoryFromLocation(window.location),
                immediate: false,
                settingsCascade,
            }),
        [nextQueryInput, settingsCascade]
    )

    const onQueryChangeImmediate = useCallback(
        () =>
            nextQueryInput({
                query: getQueryFromLocation(window.location),
                category: getCategoryFromLocation(window.location),
                immediate: true,
                settingsCascade,
            }),
        [nextQueryInput, settingsCascade]
    )

    const onSelectCategory = useCallback(
        (category: ExtensionCategory | 'All') => {
            const query = getQueryFromLocation(window.location)

            history.push({
                search: new URLSearchParams(
                    query
                        ? {
                              [URL_QUERY_PARAM]: query,
                              [URL_CATEGORY_PARAM]: category,
                          }
                        : {
                              [URL_CATEGORY_PARAM]: category,
                          }
                ).toString(),
                hash: window.location.hash,
            })
        },
        [history]
    )

    // Keep state in sync with URL
    useEffect(() => {
        // kicks off initial request
        onQueryChangeImmediate()
    }, [location, onQueryChangeImmediate])

    const isLoading = !data || data === LOADING

    return (
        <>
            <div className="container">
                <PageTitle title="Extensions" />
                <div className="d-flex mt-3 pt-3">
                    <ExtensionRegistrySidenav
                        selectedCategory={selectedCategory}
                        onSelectCategory={onSelectCategory}
                        enablementFilter={enablementFilter}
                        setEnablementFilter={setEnablementFilter}
                    />
                    <div className="flex-grow-1">
                        <div className="mb-5">
                            <div className="row">
                                <span className="mb-3 col-lg-10">
                                    Connect all your other tools to get things like test coverage, 1-click open file in
                                    editor, custom highlighting, and information from your other favorite services all
                                    in one place on Sourcegraph.
                                </span>
                            </div>
                            <Form onSubmit={preventDefault} className="form-inline">
                                <div className="shadow flex-grow-1 mb-2">
                                    <input
                                        className="form-control w-100 test-extension-registry-input"
                                        type="search"
                                        placeholder="Search extensions..."
                                        name="query"
                                        value={query}
                                        onChange={onQueryChangeEvent}
                                        autoFocus={true}
                                        autoComplete="off"
                                        autoCorrect="off"
                                        autoCapitalize="off"
                                        spellCheck={false}
                                    />
                                </div>
                            </Form>
                            {!authenticatedUser && (
                                <div className="alert alert-info my-4">
                                    <span>An account is required to create, enable and disable extensions. </span>
                                    <Link to="/sign-up?returnTo=/extensions">
                                        <span className="alert-link">Register now!</span>
                                    </Link>
                                </div>
                            )}
                            <ExtensionsList
                                {...props}
                                data={data}
                                query={query}
                                enablementFilter={enablementFilter}
                                selectedCategory={selectedCategory}
                                showMoreExtensions={showMoreExtensions}
                                onShowFullCategoryClicked={onSelectCategory}
                            />
                        </div>
                        {!isLoading && !showMoreExtensions && selectedCategory === 'All' && (
                            <div className="d-flex justify-content-center">
                                <button
                                    type="button"
                                    className="btn btn-outline-secondary"
                                    onClick={() => setShowMoreExtensions(true)}
                                >
                                    Show more extensions
                                </button>
                            </div>
                        )}
                        {/* Only show the banner when there are no selected categories and it is not loading */}
                        {selectedCategory === 'All' && !isLoading && (
                            <>
                                <hr className="mt-5" />
                                <div className="my-5 justify-content-center">
                                    <ExtensionBanner />
                                </div>
                            </>
                        )}
                    </div>
                </div>
            </div>
        </>
    )
}

function getQueryFromLocation(location: Pick<H.Location, 'search'>): string {
    const parameters = new URLSearchParams(location.search)
    return parameters.get(URL_QUERY_PARAM) || ''
}

function getCategoryFromLocation(location: Pick<H.Location, 'search'>): ExtensionCategory | 'All' {
    const parameters = new URLSearchParams(location.search)
    const category = parameters.get(URL_CATEGORY_PARAM)

    if (category && isExtensionCategory(category)) {
        return category
    }

    return 'All'
}

function isExtensionCategory(category: string): category is ExtensionCategory {
    return EXTENSION_CATEGORIES.includes(category as ExtensionCategory)
}

function preventDefault(event: React.FormEvent): void {
    event.preventDefault()
}
