/**
 * Provides convenience functions for interacting with the Sourcegraph API from tests.
 */

import {
    gql,
    dataOrThrowErrors,
    createInvalidGraphQLMutationResponseError,
    isErrorGraphQLResult,
} from '../../../../shared/src/graphql/graphql'
import * as GQL from '../../../../shared/src/graphql/schema'
import { GraphQLClient } from './GraphQlClient'
import { map, tap, retryWhen, delayWhen, take, mergeMap } from 'rxjs/operators'
import { zip, timer, concat, throwError, defer, Observable } from 'rxjs'
import {
    CloneInProgressError,
    isCloneInProgressErrorLike,
    isRepoNotFoundErrorLike,
} from '../../../../shared/src/backend/errors'
import { isErrorLike, createAggregateError } from '../../../../shared/src/util/errors'
import { ResourceDestructor } from './TestResourceManager'
import { Config } from '../../../../shared/src/testing/config'
import { PlatformContext } from '../../../../shared/src/platform/context'

type WaitForRepoOptions = Partial<
    Pick<Config, 'logStatusMessages'> & {
        /**
         * If true, wait for the repositories *not* to exist, rather than to exist.
         */
        shouldNotExist?: boolean

        /**
         * How frequently to retry the test for repository existence/non-existence.
         */
        retryPeriod?: number

        /**
         * The maximum time to wait for the repository to exist.
         */
        timeout?: number

        /**
         * Wait for repository to be indexed, not just cloned. If shouldNotExist is true, this has
         * no effect.
         */
        indexed?: boolean
    }
>

/**
 * Wait until all repositories in the list exist.
 */
export async function waitForRepos(
    gqlClient: GraphQLClient,
    ensureRepos: string[],
    options: WaitForRepoOptions = {}
): Promise<void> {
    await zip(...ensureRepos.map(repoName => waitForRepo(gqlClient, repoName, options))).toPromise()
}

export function waitForRepo(
    gqlClient: GraphQLClient,
    repoName: string,
    {
        logStatusMessages,
        shouldNotExist = false,
        retryPeriod = 2000, // 2 seconds
        timeout = 60000, // 1 minute
        indexed: mustBeIndexed = false,
    }: WaitForRepoOptions = {}
): Observable<void> {
    const request = gqlClient.queryGraphQL(
        gql`
            query ResolveRev($repoName: String!) {
                repository(name: $repoName) {
                    mirrorInfo {
                        cloned
                    }
                    textSearchIndex {
                        status {
                            updatedAt
                        }
                    }
                }
            }
        `,
        { repoName }
    )
    const numberRetries = Math.ceil(timeout / retryPeriod)
    return shouldNotExist
        ? request.pipe(
              map(result => {
                  // map to true if repo is not found, false if repo is found, throw other errors
                  if (isErrorGraphQLResult(result) && result.errors.some(isRepoNotFoundErrorLike)) {
                      return undefined
                  }
                  const { repository } = dataOrThrowErrors(result)
                  if (!repository) {
                      return undefined
                  }
                  throw new Error('Repo exists')
              }),
              retryWhen(errors =>
                  concat(
                      errors.pipe(
                          delayWhen((error, retryCount) => {
                              if (isErrorLike(error) && error.message === 'Repo exists') {
                                  // Delay retry by 2s.
                                  if (logStatusMessages) {
                                      console.log(
                                          `Waiting for ${repoName} to be removed (attempt ${
                                              retryCount + 1
                                          } of ${numberRetries})`
                                      )
                                  }
                                  return timer(retryPeriod)
                              }
                              // Throw all errors
                              throw error
                          }),
                          take(numberRetries)
                      ),
                      defer(() => throwError(new Error(`Could not resolve repo ${repoName}: too many retries`)))
                  )
              )
          )
        : request.pipe(
              map(dataOrThrowErrors),
              // Wait until the repository is cloned even if it doesn't yet exist.
              // waitForRepos might be called immediately after adding a new external service,
              // and we have no guarantee that all the repositories from that external service
              // will exist when the add-external-service endpoint returns.
              tap(({ repository }) => {
                  if (!repository?.mirrorInfo?.cloned) {
                      throw new CloneInProgressError(repoName)
                  }
                  if (mustBeIndexed && !repository?.textSearchIndex?.status?.updatedAt) {
                      throw new CloneInProgressError(repoName)
                  }
              }),
              retryWhen(errors =>
                  concat(
                      errors.pipe(
                          delayWhen((error, retryCount) => {
                              if (isCloneInProgressErrorLike(error)) {
                                  // Delay retry by 2s.
                                  if (logStatusMessages) {
                                      console.log(
                                          `Waiting for ${repoName} to finish cloning (attempt ${
                                              retryCount + 1
                                          } of ${numberRetries})`
                                      )
                                  }
                                  return timer(retryPeriod)
                              }
                              // Throw all errors other than ECLONEINPROGRESS
                              throw error
                          }),
                          take(numberRetries)
                      ),
                      defer(() => throwError(new Error(`Could not resolve repo ${repoName}: too many retries`)))
                  )
              ),
              map(() => undefined)
          )
}

export async function ensureNoTestExternalServices(
    gqlClient: GraphQLClient,
    options: {
        kind: GQL.ExternalServiceKind
        uniqueDisplayName: string
        deleteIfExist?: boolean
    }
): Promise<void> {
    if (!options.uniqueDisplayName.startsWith('[TEST]')) {
        throw new Error(
            `Test external service name ${JSON.stringify(options.uniqueDisplayName)} must start with "[TEST]".`
        )
    }

    const externalServices = await getExternalServices(gqlClient, options)
    if (externalServices.length === 0) {
        return
    }
    if (!options.deleteIfExist) {
        throw new Error('external services already exist, not deleting')
    }

    for (const externalService of externalServices) {
        await gqlClient
            .mutateGraphQL(
                gql`
                    mutation DeleteExternalService($externalService: ID!) {
                        deleteExternalService(externalService: $externalService) {
                            alwaysNil
                        }
                    }
                `,
                { externalService: externalService.id }
            )
            .toPromise()
    }
}

/**
 * TODO(beyang): remove this after the corresponding API in the main code has been updated to use a
 * dependency-injected `requestGraphQL`.
 */
export function getExternalServices(
    gqlClient: GraphQLClient,
    options: {
        kind?: GQL.ExternalServiceKind
        uniqueDisplayName?: string
    } = {}
): Promise<GQL.IExternalService[]> {
    return gqlClient
        .queryGraphQL(
            gql`
                query ExternalServicesRegression($first: Int) {
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
                }
            `,
            { first: 100 }
        )
        .pipe(
            map(dataOrThrowErrors),
            map(({ externalServices }) =>
                externalServices.nodes.filter(
                    ({ displayName, kind }) =>
                        (options.uniqueDisplayName === undefined || options.uniqueDisplayName === displayName) &&
                        (options.kind === undefined || options.kind === kind)
                )
            )
        )
        .toPromise()
}

export async function updateExternalService(
    gqlClient: GraphQLClient,
    input: GQL.IUpdateExternalServiceInput
): Promise<void> {
    await gqlClient
        .mutateGraphQL(
            gql`
                mutation UpdateExternalServiceRegression($input: UpdateExternalServiceInput!) {
                    updateExternalService(input: $input) {
                        warning
                    }
                }
            `,
            { input }
        )
        .pipe(
            map(dataOrThrowErrors),
            tap(({ updateExternalService: { warning } }) => {
                if (warning) {
                    console.warn('updateExternalService warning:', warning)
                }
            })
        )
        .toPromise()
}

export async function ensureTestExternalService(
    gqlClient: GraphQLClient,
    externalServiceOptions: {
        kind: GQL.ExternalServiceKind
        uniqueDisplayName: string
        config: Record<string, any>
        waitForRepos?: string[]
    },
    waitForReposOptions?: WaitForRepoOptions
): Promise<ResourceDestructor> {
    if (!externalServiceOptions.uniqueDisplayName.startsWith('[TEST]')) {
        throw new Error(
            `Test external service name ${JSON.stringify(
                externalServiceOptions.uniqueDisplayName
            )} must start with "[TEST]".`
        )
    }

    const destroy = (): Promise<void> =>
        ensureNoTestExternalServices(gqlClient, { ...externalServiceOptions, deleteIfExist: true })

    const externalServices = await getExternalServices(gqlClient, externalServiceOptions)
    if (externalServices.length > 0) {
        return destroy
    }

    // Add a new external service if one doesn't already exist.
    const input: GQL.IAddExternalServiceInput = {
        kind: externalServiceOptions.kind,
        displayName: externalServiceOptions.uniqueDisplayName,
        config: JSON.stringify(externalServiceOptions.config),
    }
    await addExternalService(input, gqlClient).toPromise()

    if (externalServiceOptions.waitForRepos && externalServiceOptions.waitForRepos.length > 0) {
        await waitForRepos(gqlClient, externalServiceOptions.waitForRepos, waitForReposOptions)
    }

    return destroy
}

/**
 * TODO(beyang): remove this after the corresponding API in the main code has been updated to use a
 * dependency-injected `requestGraphQL`.
 */
export async function deleteUser(
    { requestGraphQL }: Pick<PlatformContext, 'requestGraphQL'>,
    username: string,
    mustAlreadyExist: boolean = true
): Promise<void> {
    let user: GQL.IUser | null
    try {
        user = await getUser({ requestGraphQL }, username)
    } catch (error) {
        if (mustAlreadyExist) {
            throw error
        } else {
            return
        }
    }

    if (!user) {
        if (mustAlreadyExist) {
            throw new Error(`Fetched user ${username} was null`)
        } else {
            return
        }
    }

    await requestGraphQL<GQL.IMutation>({
        request: gql`
            mutation DeleteUser($user: ID!, $hard: Boolean) {
                deleteUser(user: $user, hard: $hard) {
                    alwaysNil
                }
            }
        `,
        variables: { hard: false, user: user.id },
        mightContainPrivateInfo: false,
    }).toPromise()
}

/**
 * TODO(beyang): remove this after the corresponding API in the main code has been updated to use a
 * dependency-injected `requestGraphQL`.
 */
export async function setUserSiteAdmin(gqlClient: GraphQLClient, userID: GQL.ID, siteAdmin: boolean): Promise<void> {
    await gqlClient
        .mutateGraphQL(
            gql`
                mutation SetUserIsSiteAdmin($userID: ID!, $siteAdmin: Boolean!) {
                    setUserIsSiteAdmin(userID: $userID, siteAdmin: $siteAdmin) {
                        alwaysNil
                    }
                }
            `,
            { userID, siteAdmin }
        )
        .toPromise()
}

/**
 * TODO(beyang): remove this after the corresponding API in the main code has been updated to use a
 * dependency-injected `requestGraphQL`.
 */
export function currentProductVersion(gqlClient: GraphQLClient): Promise<string> {
    return gqlClient
        .queryGraphQL(
            gql`
                query SiteFlags {
                    site {
                        productVersion
                    }
                }
            `,
            {}
        )
        .pipe(
            map(dataOrThrowErrors),
            map(({ site }) => site.productVersion)
        )
        .toPromise()
}

/**
 * TODO(beyang): remove this after the corresponding API in the main code has been updated to use a
 * dependency-injected `requestGraphQL`.
 */
export async function setUserEmailVerified(
    gqlClient: GraphQLClient,
    username: string,
    email: string,
    verified: boolean
): Promise<void> {
    const user = await getUser(gqlClient, username)
    if (!user) {
        throw new Error(`User ${username} does not exist`)
    }
    await gqlClient
        .mutateGraphQL(
            gql`
                mutation SetUserEmailVerified($user: ID!, $email: String!, $verified: Boolean!) {
                    setUserEmailVerified(user: $user, email: $email, verified: $verified) {
                        alwaysNil
                    }
                }
            `,
            { user: user.id, email, verified }
        )
        .pipe(map(dataOrThrowErrors))
        .toPromise()
}

/**
 * TODO(beyang): remove this after the corresponding API in the main code has been updated to use a
 * dependency-injected `requestGraphQL`.
 */
export function getViewerSettings({
    requestGraphQL,
}: Pick<PlatformContext, 'requestGraphQL'>): Promise<GQL.ISettingsCascade> {
    return requestGraphQL<GQL.IQuery>({
        request: gql`
            query ViewerSettings {
                viewerSettings {
                    ...SettingsCascadeFields
                }
            }

            fragment SettingsCascadeFields on SettingsCascade {
                subjects {
                    __typename
                    ... on Org {
                        id
                        name
                        displayName
                    }
                    ... on User {
                        id
                        username
                        displayName
                    }
                    ... on Site {
                        id
                        siteID
                    }
                    latestSettings {
                        id
                        contents
                    }
                    settingsURL
                    viewerCanAdminister
                }
                final
            }
        `,
        variables: {},
        mightContainPrivateInfo: true,
    })
        .pipe(
            map(dataOrThrowErrors),
            map(data => data.viewerSettings)
        )
        .toPromise()
}

/**
 * TODO(beyang): remove this after the corresponding API in the main code has been updated to use a
 * dependency-injected `requestGraphQL`.
 */
export function deleteOrganization(
    { requestGraphQL }: Pick<PlatformContext, 'requestGraphQL'>,
    organization: GQL.ID
): Observable<void> {
    return requestGraphQL<GQL.IMutation>({
        request: gql`
            mutation DeleteOrganization($organization: ID!) {
                deleteOrganization(organization: $organization) {
                    alwaysNil
                }
            }
        `,
        variables: { organization },
        mightContainPrivateInfo: true,
    }).pipe(
        map(dataOrThrowErrors),
        map(data => {
            if (!data.deleteOrganization) {
                throw createInvalidGraphQLMutationResponseError('DeleteOrganization')
            }
        })
    )
}

/**
 * TODO(beyang): remove this after the corresponding API in the main code has been updated to use a
 * dependency-injected `requestGraphQL`.
 */
export function fetchAllOrganizations(
    { requestGraphQL }: Pick<PlatformContext, 'requestGraphQL'>,
    args: { first?: number; query?: string }
): Observable<GQL.IOrgConnection> {
    return requestGraphQL<GQL.IQuery>({
        request: gql`
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
        variables: args,
        mightContainPrivateInfo: true,
    }).pipe(
        map(dataOrThrowErrors),
        map(data => data.organizations)
    )
}

interface EventLogger {
    log: (eventLabel: string, eventProperties?: any) => void
}

/**
 * TODO(beyang): remove this after the corresponding API in the main code has been updated to use a
 * dependency-injected `requestGraphQL`.
 */
export function createOrganization(
    {
        requestGraphQL,
        eventLogger = { log: () => undefined },
    }: Pick<PlatformContext, 'requestGraphQL'> & {
        eventLogger?: EventLogger
    },
    variables: {
        /** The name of the organization. */
        name: string
        /** The new organization's display name (e.g. full name) in the organization profile. */
        displayName?: string
    }
): Observable<GQL.IOrg> {
    return requestGraphQL<GQL.IMutation>({
        request: gql`
            mutation createOrganization($name: String!, $displayName: String) {
                createOrganization(name: $name, displayName: $displayName) {
                    id
                    name
                }
            }
        `,
        variables,
        mightContainPrivateInfo: false,
    }).pipe(
        mergeMap(({ data, errors }) => {
            if (!data?.createOrganization) {
                eventLogger.log('NewOrgFailed')
                throw createAggregateError(errors)
            }
            eventLogger.log('NewOrgCreated')
            return concat([data.createOrganization])
        })
    )
}

/**
 * TODO(beyang): remove this after the corresponding API in the main code has been updated to use a
 * dependency-injected `requestGraphQL`.
 */
export function createUser(
    { requestGraphQL }: Pick<PlatformContext, 'requestGraphQL'>,
    username: string,
    email: string | undefined
): Observable<GQL.ICreateUserResult> {
    return requestGraphQL<GQL.IMutation>({
        request: gql`
            mutation CreateUser($username: String!, $email: String) {
                createUser(username: $username, email: $email) {
                    resetPasswordURL
                }
            }
        `,
        variables: { username, email },
        mightContainPrivateInfo: true,
    }).pipe(
        map(dataOrThrowErrors),
        map(data => data.createUser)
    )
}

/**
 * TODO(beyang): remove this after the corresponding API in the main code has been updated to use a
 * dependency-injected `requestGraphQL`.
 */
export async function getUser(
    { requestGraphQL }: Pick<PlatformContext, 'requestGraphQL'>,
    username: string
): Promise<GQL.IUser | null> {
    const user = await requestGraphQL<GQL.IQuery>({
        request: gql`
            query User($username: String!) {
                user(username: $username) {
                    __typename
                    id
                    username
                    displayName
                    url
                    settingsURL
                    avatarURL
                    viewerCanAdminister
                    siteAdmin
                    createdAt
                    emails {
                        email
                        verified
                    }
                    organizations {
                        nodes {
                            id
                            displayName
                            name
                        }
                    }
                    settingsCascade {
                        subjects {
                            latestSettings {
                                id
                                contents
                            }
                        }
                    }
                }
            }
        `,
        variables: { username },
        mightContainPrivateInfo: true,
    })
        .pipe(
            map(dataOrThrowErrors),
            map(({ user }) => user)
        )
        .toPromise()
    return user
}

/**
 * TODO(beyang): remove this after the corresponding API in the main code has been updated to use a
 * dependency-injected `requestGraphQL`.
 */
export function addExternalService(
    input: GQL.IAddExternalServiceInput,
    {
        eventLogger = { log: () => undefined },
        requestGraphQL,
    }: Pick<PlatformContext, 'requestGraphQL'> & { eventLogger: EventLogger }
): Observable<GQL.IExternalService> {
    return requestGraphQL<GQL.IMutation>({
        request: gql`
            mutation addExternalService($input: AddExternalServiceInput!) {
                addExternalService(input: $input) {
                    id
                    kind
                    displayName
                    warning
                }
            }
        `,
        variables: { input },
        mightContainPrivateInfo: true,
    }).pipe(
        map(({ data, errors }) => {
            if (!data?.addExternalService || (errors && errors.length > 0)) {
                eventLogger.log('AddExternalServiceFailed')
                throw createAggregateError(errors)
            }
            eventLogger.log('AddExternalServiceSucceeded')
            return data.addExternalService
        })
    )
}

const genericSearchResultInterfaceFields = gql`
  label {
      html
  }
  url
  icon
  detail {
      html
  }
  matches {
      url
      body {
          text
          html
      }
      highlights {
          line
          character
          length
      }
  }
`

export function search(
    { requestGraphQL }: Pick<PlatformContext, 'requestGraphQL'>,
    query: string,
    version: string,
    patternType: GQL.SearchPatternType
): Promise<GQL.ISearch> {
    return requestGraphQL<GQL.IQuery>({
        request: gql`
        query Search($query: String!, $version: SearchVersion!, $patternType: SearchPatternType!) {
            search(query: $query, version: $version, patternType: $patternType) {
                results {
                    __typename
                    limitHit
                    resultCount
                    matchCount
                    approximateResultCount
                    missing {
                        name
                    }
                    cloning {
                        name
                    }
                    timedout {
                        name
                    }
                    indexUnavailable
                    dynamicFilters {
                        value
                        label
                        count
                        limitHit
                        kind
                    }
                    results {
                        __typename
                        ... on Repository {
                            id
                            name
                            ${genericSearchResultInterfaceFields}
                        }
                        ... on FileMatch {
                            file {
                                path
                                url
                                commit {
                                    oid
                                }
                            }
                            repository {
                                name
                                url
                            }
                            limitHit
                            symbols {
                                name
                                containerName
                                url
                                kind
                            }
                            lineMatches {
                                preview
                                lineNumber
                                offsetAndLengths
                            }
                        }
                        ... on CommitSearchResult {
                            ${genericSearchResultInterfaceFields}
                        }
                    }
                    alert {
                        title
                        description
                        proposedQueries {
                            description
                            query
                        }
                    }
                    elapsedMilliseconds
                }
            }
        }
        `,
        variables: { query, version, patternType },
        mightContainPrivateInfo: false,
    })
        .pipe(
            map(dataOrThrowErrors),
            map(data => {
                if (!data.search) {
                    throw new Error('no results field in search response')
                }
                return data.search
            })
        )
        .toPromise()
}

/**
 * Fetches the site and its configuration.
 *
 * @returns Observable that emits the site
 */
export function fetchSiteConfiguration({
    requestGraphQL,
}: Pick<PlatformContext, 'requestGraphQL'>): Observable<GQL.ISite> {
    return requestGraphQL<GQL.IQuery>({
        request: gql`
            query Site {
                site {
                    id
                    configuration {
                        id
                        effectiveContents
                        validationMessages
                    }
                }
            }
        `,
        variables: {},
        mightContainPrivateInfo: true,
    }).pipe(
        map(dataOrThrowErrors),
        map(data => data.site)
    )
}

/**
 * Updates the site's configuration.
 *
 * @returns An observable indicating whether or not a service restart is
 * required for the update to be applied.
 */
export function updateSiteConfiguration(
    { requestGraphQL }: Pick<PlatformContext, 'requestGraphQL'>,
    lastID: number,
    input: string
): Observable<boolean> {
    return requestGraphQL<GQL.IMutation>({
        request: gql`
            mutation UpdateSiteConfiguration($lastID: Int!, $input: String!) {
                updateSiteConfiguration(lastID: $lastID, input: $input)
            }
        `,
        variables: { lastID, input },
        mightContainPrivateInfo: true,
    }).pipe(
        map(dataOrThrowErrors),
        map(data => data.updateSiteConfiguration)
    )
}
