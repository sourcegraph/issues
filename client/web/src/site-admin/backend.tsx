import { parse as parseJSONC } from '@sqs/jsonc-parser'
import { Observable } from 'rxjs'
import { map, tap, mapTo } from 'rxjs/operators'
import { repeatUntil } from '../../../shared/src/util/rxjs/repeatUntil'
import {
    createInvalidGraphQLMutationResponseError,
    dataOrThrowErrors,
    isErrorGraphQLResult,
    gql,
} from '../../../shared/src/graphql/graphql'
import { createAggregateError } from '../../../shared/src/util/errors'
import * as GQL from '../../../shared/src/graphql/schema'
import { resetAllMemoizationCaches } from '../../../shared/src/util/memoizeObservable'
import { mutateGraphQL, queryGraphQL, requestGraphQL } from '../backend/graphql'
import { Settings } from '../../../shared/src/settings/settings'
import { RepositoriesVariables, RepositoriesResult, ExternalServiceKind, UserActivePeriod } from '../graphql-operations'

/**
 * Fetches all users.
 */
export function fetchAllUsers(args: { first?: number; query?: string }): Observable<GQL.IUserConnection> {
    return queryGraphQL(
        gql`
            query Users($first: Int, $query: String) {
                users(first: $first, query: $query) {
                    nodes {
                        id
                        username
                        displayName
                        emails {
                            email
                            verified
                            verificationPending
                            viewerCanManuallyVerify
                        }
                        createdAt
                        siteAdmin
                        latestSettings {
                            createdAt
                            contents
                        }
                        organizations {
                            nodes {
                                name
                            }
                        }
                    }
                    totalCount
                }
            }
        `,
        args
    ).pipe(
        map(dataOrThrowErrors),
        map(data => data.users)
    )
}

/**
 * Fetches all organizations.
 */
export function fetchAllOrganizations(args: { first?: number; query?: string }): Observable<GQL.IOrgConnection> {
    return queryGraphQL(
        gql`
            query Organizations($first: Int, $query: String) {
                organizations(first: $first, query: $query) {
                    nodes {
                        id
                        name
                        displayName
                        createdAt
                        latestSettings {
                            createdAt
                            contents
                        }
                        members {
                            totalCount
                        }
                    }
                    totalCount
                }
            }
        `,
        args
    ).pipe(
        map(dataOrThrowErrors),
        map(data => data.organizations)
    )
}

const siteAdminRepositoryFieldsFragment = gql`
    fragment SiteAdminRepositoryFields on Repository {
        id
        name
        createdAt
        viewerCanAdminister
        url
        mirrorInfo {
            cloned
            cloneInProgress
            updatedAt
        }
    }
`

/**
 * Fetches all repositories.
 *
 * @returns Observable that emits the list of repositories
 */
function fetchAllRepositories(args: Partial<RepositoriesVariables>): Observable<RepositoriesResult['repositories']> {
    return requestGraphQL<RepositoriesResult, RepositoriesVariables>(
        gql`
            query Repositories(
                $first: Int
                $query: String
                $cloned: Boolean
                $notCloned: Boolean
                $indexed: Boolean
                $notIndexed: Boolean
            ) {
                repositories(
                    first: $first
                    query: $query
                    cloned: $cloned
                    notCloned: $notCloned
                    indexed: $indexed
                    notIndexed: $notIndexed
                ) {
                    nodes {
                        ...SiteAdminRepositoryFields
                    }
                    totalCount(precise: true)
                    pageInfo {
                        hasNextPage
                    }
                }
            }

            ${siteAdminRepositoryFieldsFragment}
        `,
        {
            cloned: args.cloned ?? true,
            notCloned: args.notCloned ?? true,
            indexed: args.indexed ?? true,
            notIndexed: args.notIndexed ?? true,
            first: args.first ?? null,
            query: args.query ?? null,
        }
    ).pipe(
        map(dataOrThrowErrors),
        map(data => data.repositories)
    )
}

export function fetchAllRepositoriesAndPollIfEmptyOrAnyCloning(
    args: Partial<RepositoriesVariables>
): Observable<RepositoriesResult['repositories']> {
    return fetchAllRepositories(args).pipe(
        // Poll every 5000ms if repositories are being cloned or the list is empty.
        repeatUntil(
            result =>
                result.nodes &&
                result.nodes.length > 0 &&
                result.nodes.every(nodes => !nodes.mirrorInfo.cloneInProgress && nodes.mirrorInfo.cloned),
            { delay: 5000 }
        )
    )
}

export function updateMirrorRepository(args: { repository: GQL.ID }): Observable<void> {
    return mutateGraphQL(
        gql`
            mutation UpdateMirrorRepository($repository: ID!) {
                updateMirrorRepository(repository: $repository) {
                    alwaysNil
                }
            }
        `,
        args
    ).pipe(
        map(dataOrThrowErrors),
        tap(() => resetAllMemoizationCaches()),
        map(() => undefined)
    )
}

export function checkMirrorRepositoryConnection(
    args:
        | {
              repository: GQL.ID
          }
        | {
              name: string
          }
): Observable<GQL.ICheckMirrorRepositoryConnectionResult> {
    return mutateGraphQL(
        gql`
            mutation CheckMirrorRepositoryConnection($repository: ID, $name: String) {
                checkMirrorRepositoryConnection(repository: $repository, name: $name) {
                    error
                }
            }
        `,
        args
    ).pipe(
        map(dataOrThrowErrors),
        tap(() => resetAllMemoizationCaches()),
        map(data => data.checkMirrorRepositoryConnection)
    )
}

export function scheduleRepositoryPermissionsSync(args: { repository: GQL.ID }): Observable<void> {
    return mutateGraphQL(
        gql`
            mutation ScheduleRepositoryPermissionsSync($repository: ID!) {
                scheduleRepositoryPermissionsSync(repository: $repository) {
                    alwaysNil
                }
            }
        `,
        args
    ).pipe(
        map(dataOrThrowErrors),
        tap(() => resetAllMemoizationCaches()),
        mapTo(undefined)
    )
}

export function scheduleUserPermissionsSync(args: { user: GQL.ID }): Observable<void> {
    return mutateGraphQL(
        gql`
            mutation ScheduleUserPermissionsSync($user: ID!) {
                scheduleUserPermissionsSync(user: $user) {
                    alwaysNil
                }
            }
        `,
        args
    ).pipe(
        map(dataOrThrowErrors),
        tap(() => resetAllMemoizationCaches()),
        mapTo(undefined)
    )
}

/**
 * Fetches usage statistics for all users.
 *
 * @returns Observable that emits the list of users and their usage data
 */
export function fetchUserUsageStatistics(args: {
    activePeriod?: UserActivePeriod
    query?: string
    first?: number
}): Observable<GQL.IUserConnection> {
    return queryGraphQL(
        gql`
            query UserUsageStatistics($activePeriod: UserActivePeriod, $query: String, $first: Int) {
                users(activePeriod: $activePeriod, query: $query, first: $first) {
                    nodes {
                        id
                        username
                        usageStatistics {
                            searchQueries
                            pageViews
                            codeIntelligenceActions
                            lastActiveTime
                            lastActiveCodeHostIntegrationTime
                        }
                    }
                    totalCount
                }
            }
        `,
        args
    ).pipe(
        map(dataOrThrowErrors),
        map(data => data.users)
    )
}

/**
 * Fetches site-wide usage statitics.
 *
 * @returns Observable that emits the list of users and their usage data
 */
export function fetchSiteUsageStatistics(): Observable<GQL.ISiteUsageStatistics> {
    return queryGraphQL(gql`
        query SiteUsageStatistics {
            site {
                usageStatistics {
                    daus {
                        userCount
                        registeredUserCount
                        anonymousUserCount
                        startTime
                    }
                    waus {
                        userCount
                        registeredUserCount
                        anonymousUserCount
                        startTime
                    }
                    maus {
                        userCount
                        registeredUserCount
                        anonymousUserCount
                        startTime
                    }
                }
            }
        }
    `).pipe(
        map(dataOrThrowErrors),
        map(data => data.site.usageStatistics)
    )
}

/**
 * Fetches the site and its configuration.
 *
 * @returns Observable that emits the site
 */
export function fetchSite(): Observable<GQL.ISite> {
    return queryGraphQL(gql`
        query Site {
            site {
                id
                canReloadSite
                configuration {
                    id
                    effectiveContents
                    validationMessages
                }
            }
        }
    `).pipe(
        map(dataOrThrowErrors),
        map(data => data.site)
    )
}

/**
 * Placeholder for the type of the external service config (to avoid explicit 'any' type)
 */
interface ExternalServiceConfig {}

type SettingsSubject = Pick<GQL.SettingsSubject, 'settingsURL' | '__typename'> & {
    contents: Settings
}

/**
 * All configuration and settings in one place.
 */
interface AllConfig {
    site: GQL.ISiteConfiguration
    externalServices: Partial<Record<ExternalServiceKind, ExternalServiceConfig>>
    settings: {
        subjects: SettingsSubject[]
        final: Settings | null
    }
}

/**
 * Fetches all the configuration and settings (requires site admin privileges).
 */
export function fetchAllConfigAndSettings(): Observable<AllConfig> {
    return queryGraphQL(
        gql`
            query AllConfig($first: Int) {
                site {
                    id
                    configuration {
                        id
                        effectiveContents
                    }
                    latestSettings {
                        contents
                    }
                    settingsCascade {
                        final
                    }
                }

                externalServices(first: $first) {
                    nodes {
                        id
                        kind
                        displayName
                        config
                        createdAt
                        updatedAt
                        warning
                    }
                }

                viewerSettings {
                    ...SiteAdminSettingsCascadeFields
                }
            }

            fragment SiteAdminSettingsCascadeFields on SettingsCascade {
                subjects {
                    __typename
                    latestSettings {
                        id
                        contents
                    }
                    settingsURL
                }
                final
            }
        `,
        { first: 100 } // assume no more than 100 external services added
    ).pipe(
        map(dataOrThrowErrors),
        map(data => {
            const externalServices: Partial<Record<
                ExternalServiceKind,
                ExternalServiceConfig[]
            >> = data.externalServices.nodes
                .filter(svc => svc.config)
                .map(svc => [svc.kind, parseJSONC(svc.config) as ExternalServiceConfig] as const)
                .reduce<Partial<{ [k in ExternalServiceKind]: ExternalServiceConfig[] }>>(
                    (externalServicesByKind, [kind, config]) => {
                        let services = externalServicesByKind[kind]
                        if (!services) {
                            services = []
                            externalServicesByKind[kind] = services
                        }
                        services.push(config)
                        return externalServicesByKind
                    },
                    {}
                )
            const settingsSubjects = data.viewerSettings.subjects.map(settings => ({
                __typename: settings.__typename,
                settingsURL: settings.settingsURL,
                contents: settings.latestSettings ? parseJSONC(settings.latestSettings.contents) : null,
            }))
            const finalSettings = parseJSONC(data.viewerSettings.final)
            return {
                site:
                    data.site?.configuration?.effectiveContents &&
                    parseJSONC(data.site.configuration.effectiveContents),
                externalServices,
                settings: {
                    subjects: settingsSubjects,
                    final: finalSettings,
                },
            }
        })
    )
}

/**
 * Updates the site's configuration.
 *
 * @returns An observable indicating whether or not a service restart is
 * required for the update to be applied.
 */
export function updateSiteConfiguration(lastID: number, input: string): Observable<boolean> {
    return mutateGraphQL(
        gql`
            mutation UpdateSiteConfiguration($lastID: Int!, $input: String!) {
                updateSiteConfiguration(lastID: $lastID, input: $input)
            }
        `,
        { lastID, input }
    ).pipe(
        map(dataOrThrowErrors),
        map(data => data.updateSiteConfiguration)
    )
}

/**
 * Reloads the site.
 */
export function reloadSite(): Observable<void> {
    return mutateGraphQL(
        gql`
            mutation ReloadSite {
                reloadSite {
                    alwaysNil
                }
            }
        `
    ).pipe(
        map(dataOrThrowErrors),
        map(data => {
            if (!data.reloadSite) {
                throw createInvalidGraphQLMutationResponseError('ReloadSite')
            }
        })
    )
}

export function setUserIsSiteAdmin(userID: GQL.ID, siteAdmin: boolean): Observable<void> {
    return mutateGraphQL(
        gql`
            mutation SetUserIsSiteAdmin($userID: ID!, $siteAdmin: Boolean!) {
                setUserIsSiteAdmin(userID: $userID, siteAdmin: $siteAdmin) {
                    alwaysNil
                }
            }
        `,
        { userID, siteAdmin }
    ).pipe(
        map(dataOrThrowErrors),
        map(() => undefined)
    )
}

export function invalidateSessionsByID(userID: GQL.ID): Observable<void> {
    return mutateGraphQL(
        gql`
            mutation invalidateSessionsByID($userID: ID!) {
                invalidateSessionsByID(userID: $userID) {
                    alwaysNil
                }
            }
        `,
        { userID }
    ).pipe(
        map(dataOrThrowErrors),
        map(() => undefined)
    )
}

export function randomizeUserPassword(user: GQL.ID): Observable<GQL.IRandomizeUserPasswordResult> {
    return mutateGraphQL(
        gql`
            mutation RandomizeUserPassword($user: ID!) {
                randomizeUserPassword(user: $user) {
                    resetPasswordURL
                }
            }
        `,
        { user }
    ).pipe(
        map(dataOrThrowErrors),
        map(data => data.randomizeUserPassword)
    )
}

export function deleteUser(user: GQL.ID, hard?: boolean): Observable<void> {
    return mutateGraphQL(
        gql`
            mutation DeleteUser($user: ID!, $hard: Boolean) {
                deleteUser(user: $user, hard: $hard) {
                    alwaysNil
                }
            }
        `,
        { user, hard }
    ).pipe(
        map(dataOrThrowErrors),
        map(data => {
            if (!data.deleteUser) {
                throw createInvalidGraphQLMutationResponseError('DeleteUser')
            }
        })
    )
}

export function createUser(username: string, email: string | undefined): Observable<GQL.ICreateUserResult> {
    return mutateGraphQL(
        gql`
            mutation CreateUser($username: String!, $email: String) {
                createUser(username: $username, email: $email) {
                    resetPasswordURL
                }
            }
        `,
        { username, email }
    ).pipe(
        map(dataOrThrowErrors),
        map(data => data.createUser)
    )
}

export function deleteOrganization(organization: GQL.ID): Observable<void> {
    return mutateGraphQL(
        gql`
            mutation DeleteOrganization($organization: ID!) {
                deleteOrganization(organization: $organization) {
                    alwaysNil
                }
            }
        `,
        { organization }
    ).pipe(
        map(dataOrThrowErrors),
        map(data => {
            if (!data.deleteOrganization) {
                throw createInvalidGraphQLMutationResponseError('DeleteOrganization')
            }
        })
    )
}

export function fetchSiteUpdateCheck(): Observable<{
    buildVersion: string
    productVersion: string
    updateCheck: GQL.IUpdateCheck
}> {
    return queryGraphQL(
        gql`
            query SiteUpdateCheck {
                site {
                    buildVersion
                    productVersion
                    updateCheck {
                        pending
                        checkedAt
                        errorMessage
                        updateVersionAvailable
                    }
                }
            }
        `
    ).pipe(
        map(dataOrThrowErrors),
        map(data => data.site)
    )
}

/**
 * Resolves to `false` if prometheus API is unavailable (due to being disabled or not configured in this deployment)
 *
 * @param days number of days of data to fetch
 */
export function fetchMonitoringStats(days: number): Observable<GQL.IMonitoringStatistics | false> {
    // more details in /internal/prometheusutil.ErrPrometheusUnavailable
    const errorPrometheusUnavailable = 'prometheus API is unavailable'
    return queryGraphQL(
        gql`
            query SiteMonitoringStatistics($days: Int!) {
                site {
                    monitoringStatistics(days: $days) {
                        alerts {
                            serviceName
                            name
                            timestamp
                            average
                            owner
                        }
                    }
                }
            }
        `,
        { days }
    ).pipe(
        map(result => {
            if (isErrorGraphQLResult(result)) {
                if (result.errors.find(error => error.message.includes(errorPrometheusUnavailable))) {
                    return false
                }
                throw createAggregateError(result.errors)
            }
            return result.data
        }),
        map(data => {
            if (data) {
                return data.site.monitoringStatistics
            }
            return data
        })
    )
}
