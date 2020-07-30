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
import { mutateGraphQL, queryGraphQL } from '../backend/graphql'
import { Settings } from '../../../shared/src/settings/settings'
import {
    UsersResult,
    OrganizationsResult,
    RepositoriesResult,
    UserUsageStatisticsResult,
    SiteUsageStatisticsResult,
    SiteResult,
    AllConfigResult,
    SiteUpdateCheckResult,
    SiteMonitoringStatisticsResult,
    UpdateMirrorRepositoryResult,
    ScheduleRepositoryPermissionsSyncResult,
    ScheduleUserPermissionsSyncResult,
    UpdateSiteConfigurationResult,
    ReloadSiteResult,
    SetUserIsSiteAdminResult,
    DeleteUserResult,
    DeleteOrganizationResult,
} from '../graphql-operations'

/**
 * Fetches all users.
 */
export function fetchAllUsers(args: { first?: number; query?: string }): Observable<GQL.UserConnection> {
    return queryGraphQL<UsersResult>(
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
export function fetchAllOrganizations(args: { first?: number; query?: string }): Observable<GQL.OrgConnection> {
    return queryGraphQL<OrganizationsResult>(
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

interface RepositoryArgs {
    first?: number
    query?: string
    cloned?: boolean
    cloneInProgress?: boolean
    notCloned?: boolean
    indexed?: boolean
    notIndexed?: boolean
}

/**
 * Fetches all repositories.
 *
 * @returns Observable that emits the list of repositories
 */
function fetchAllRepositories(args: RepositoryArgs): Observable<GQL.RepositoryConnection> {
    args = {
        cloned: true,
        cloneInProgress: true,
        notCloned: true,
        indexed: true,
        notIndexed: true,
        ...args,
    } // apply defaults
    return queryGraphQL<RepositoriesResult>(
        gql`
            query Repositories(
                $first: Int
                $query: String
                $cloned: Boolean
                $cloneInProgress: Boolean
                $notCloned: Boolean
                $indexed: Boolean
                $notIndexed: Boolean
            ) {
                repositories(
                    first: $first
                    query: $query
                    cloned: $cloned
                    cloneInProgress: $cloneInProgress
                    notCloned: $notCloned
                    indexed: $indexed
                    notIndexed: $notIndexed
                ) {
                    nodes {
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
                    totalCount(precise: true)
                    pageInfo {
                        hasNextPage
                    }
                }
            }
        `,
        args
    ).pipe(
        map(dataOrThrowErrors),
        map(data => data.repositories)
    )
}

export function fetchAllRepositoriesAndPollIfEmptyOrAnyCloning(
    args: RepositoryArgs
): Observable<GQL.RepositoryConnection> {
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

export function updateMirrorRepository(args: { repository: GQL.Scalars['ID'] }): Observable<void> {
    return mutateGraphQL<UpdateMirrorRepositoryResult>(
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
              repository: GQL.Scalars['ID']
          }
        | {
              name: string
          }
): Observable<GQL.CheckMirrorRepositoryConnectionResult> {
    return mutateGraphQL<GQL.CheckMirrorRepositoryConnectionResult>(
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

export function scheduleRepositoryPermissionsSync(args: { repository: GQL.Scalars['ID'] }): Observable<void> {
    return mutateGraphQL<ScheduleRepositoryPermissionsSyncResult>(
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

export function scheduleUserPermissionsSync(args: { user: GQL.Scalars['ID'] }): Observable<void> {
    return mutateGraphQL<ScheduleUserPermissionsSyncResult>(
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
    activePeriod?: GQL.UserActivePeriod
    query?: string
    first?: number
}): Observable<GQL.UserConnection> {
    return queryGraphQL<UserUsageStatisticsResult>(
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
export function fetchSiteUsageStatistics(): Observable<GQL.SiteUsageStatistics> {
    return queryGraphQL<SiteUsageStatisticsResult>(gql`
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
export function fetchSite(): Observable<GQL.Site> {
    return queryGraphQL<SiteResult>(gql`
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
    site: GQL.SiteConfiguration
    externalServices: Partial<Record<GQL.ExternalServiceKind, ExternalServiceConfig>>
    settings: {
        subjects: SettingsSubject[]
        final: Settings | null
    }
}

/**
 * Fetches all the configuration and settings (requires site admin privileges).
 */
export function fetchAllConfigAndSettings(): Observable<AllConfig> {
    return queryGraphQL<AllConfigResult>(
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
                GQL.ExternalServiceKind,
                ExternalServiceConfig[]
            >> = data.externalServices.nodes
                .filter(svc => svc.config)
                .map(svc => [svc.kind, parseJSONC(svc.config) as ExternalServiceConfig] as const)
                .reduce<Partial<{ [k in GQL.ExternalServiceKind]: ExternalServiceConfig[] }>>(
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
    return mutateGraphQL<UpdateSiteConfigurationResult>(
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
    return mutateGraphQL<ReloadSiteResult>(
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

export function setUserIsSiteAdmin(userID: GQL.Scalars['ID'], siteAdmin: boolean): Observable<void> {
    return mutateGraphQL<SetUserIsSiteAdminResult>(
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

export function randomizeUserPassword(user: GQL.Scalars['ID']): Observable<GQL.RandomizeUserPasswordResult> {
    return mutateGraphQL<GQL.RandomizeUserPasswordResult>(
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

export function deleteUser(user: GQL.Scalars['ID'], hard?: boolean): Observable<void> {
    return mutateGraphQL<DeleteUserResult>(
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

export function createUser(username: string, email: string | undefined): Observable<GQL.CreateUserResult> {
    return mutateGraphQL<GQL.CreateUserResult>(
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

export function deleteOrganization(organization: GQL.Scalars['ID']): Observable<void> {
    return mutateGraphQL<DeleteOrganizationResult>(
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
    updateCheck: GQL.UpdateCheck
}> {
    return queryGraphQL<SiteUpdateCheckResult>(
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
export function fetchMonitoringStats(days: number): Observable<GQL.MonitoringStatistics | false> {
    // more details in /internal/prometheusutil.ErrPrometheusUnavailable
    const errorPrometheusUnavailable = 'prometheus API is unavailable'
    return queryGraphQL<SiteMonitoringStatisticsResult>(
        gql`
            query SiteMonitoringStatistics($days: Int!) {
                site {
                    monitoringStatistics(days: $days) {
                        alerts {
                            serviceName
                            name
                            timestamp
                            average
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
