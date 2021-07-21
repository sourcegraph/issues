import { Observable } from 'rxjs'
import { map, mapTo } from 'rxjs/operators'

import { gql, dataOrThrowErrors } from '@sourcegraph/shared/src/graphql/graphql'
import { TelemetryService } from '@sourcegraph/shared/src/telemetry/telemetryService'
import { createAggregateError, isErrorLike, ErrorLike } from '@sourcegraph/shared/src/util/errors'

import { requestGraphQL } from '../../backend/graphql'
import {
    UpdateExternalServiceResult,
    UpdateExternalServiceVariables,
    Scalars,
    AddExternalServiceVariables,
    AddExternalServiceResult,
    ExternalServiceFields,
    ExternalServiceVariables,
    ExternalServiceResult,
    DeleteExternalServiceVariables,
    DeleteExternalServiceResult,
    ExternalServicesVariables,
    ExternalServicesResult,
    SetExternalServiceReposVariables,
    SetExternalServiceReposResult,
    AffiliatedRepositoriesVariables,
    AffiliatedRepositoriesResult,
} from '../../graphql-operations'

export const externalServiceFragment = gql`
    fragment ExternalServiceFields on ExternalService {
        id
        kind
        displayName
        config
        warning
        lastSyncError
        repoCount
        webhookURL
        lastSyncAt
        nextSyncAt
        updatedAt
        createdAt
        namespace {
            id
            namespaceName
            url
        }
    }
`

export async function addExternalService(
    variables: AddExternalServiceVariables,
    eventLogger: TelemetryService
): Promise<AddExternalServiceResult['addExternalService']> {
    return requestGraphQL<AddExternalServiceResult, AddExternalServiceVariables>(
        gql`
            mutation AddExternalService($input: AddExternalServiceInput!) {
                addExternalService(input: $input) {
                    ...ExternalServiceFields
                }
            }

            ${externalServiceFragment}
        `,
        variables
    )
        .pipe(
            map(({ data, errors }) => {
                if (!data || !data.addExternalService || (errors && errors.length > 0)) {
                    eventLogger.log('AddExternalServiceFailed')
                    throw createAggregateError(errors)
                }
                eventLogger.log('AddExternalServiceSucceeded')
                return data.addExternalService
            })
        )
        .toPromise()
}

export function isExternalService(
    externalServiceOrError?: ExternalServiceFields | ErrorLike
): externalServiceOrError is ExternalServiceFields {
    return !!externalServiceOrError && !isErrorLike(externalServiceOrError)
}

export function updateExternalService(
    variables: UpdateExternalServiceVariables
): Promise<UpdateExternalServiceResult['updateExternalService']> {
    return requestGraphQL<UpdateExternalServiceResult, UpdateExternalServiceVariables>(
        gql`
            mutation UpdateExternalService($input: UpdateExternalServiceInput!) {
                updateExternalService(input: $input) {
                    ...ExternalServiceFields
                }
            }
            ${externalServiceFragment}
        `,
        variables
    )
        .pipe(
            map(dataOrThrowErrors),
            map(data => data.updateExternalService)
        )
        .toPromise()
}

export function setExternalServiceRepos(variables: SetExternalServiceReposVariables): Promise<void> {
    return requestGraphQL<SetExternalServiceReposResult, SetExternalServiceReposVariables>(
        gql`
            mutation SetExternalServiceRepos($id: ID!, $allRepos: Boolean!, $repos: [String!]) {
                setExternalServiceRepos(id: $id, allRepos: $allRepos, repos: $repos) {
                    alwaysNil
                }
            }
        `,
        variables
    )
        .pipe(map(dataOrThrowErrors), mapTo(undefined))
        .toPromise()
}

export function fetchExternalService(id: Scalars['ID']): Observable<ExternalServiceFields> {
    return requestGraphQL<ExternalServiceResult, ExternalServiceVariables>(
        gql`
            query ExternalService($id: ID!) {
                node(id: $id) {
                    __typename
                    ...ExternalServiceFields
                }
            }
            ${externalServiceFragment}
        `,
        { id }
    ).pipe(
        map(dataOrThrowErrors),
        map(({ node }) => {
            if (!node) {
                throw new Error('External service not found')
            }
            if (node.__typename !== 'ExternalService') {
                throw new Error(`Node is a ${node.__typename}, not a ExternalService`)
            }
            return node
        })
    )
}

export function listAffiliatedRepositories(
    args: AffiliatedRepositoriesVariables
): Observable<NonNullable<AffiliatedRepositoriesResult>> {
    return requestGraphQL<AffiliatedRepositoriesResult, AffiliatedRepositoriesVariables>(
        gql`
            query AffiliatedRepositories($user: ID!, $codeHost: ID, $query: String) {
                affiliatedRepositories(user: $user, codeHost: $codeHost, query: $query) {
                    nodes {
                        name
                        codeHost {
                            kind
                            id
                            displayName
                        }
                        private
                    }
                }
            }
        `,
        {
            user: args.user,
            codeHost: args.codeHost ?? null,
            query: args.query ?? null,
        }
    ).pipe(map(dataOrThrowErrors))
}

export async function deleteExternalService(externalService: Scalars['ID']): Promise<void> {
    const result = await requestGraphQL<DeleteExternalServiceResult, DeleteExternalServiceVariables>(
        gql`
            mutation DeleteExternalService($externalService: ID!) {
                deleteExternalService(externalService: $externalService) {
                    alwaysNil
                }
            }
        `,
        { externalService }
    ).toPromise()
    dataOrThrowErrors(result)
}

export const listExternalServiceFragment = gql`
    fragment ListExternalServiceFields on ExternalService {
        id
        kind
        displayName
        config
        warning
        lastSyncError
        repoCount
        lastSyncAt
        nextSyncAt
        updatedAt
        createdAt
        namespace {
            id
            namespaceName
            url
        }
        grantedScopes
    }
`

export const EXTERNAL_SERVICES = gql`
    query ExternalServices($first: Int, $after: String, $namespace: ID) {
        externalServices(first: $first, after: $after, namespace: $namespace) {
            nodes {
                ...ListExternalServiceFields
            }
            totalCount
            pageInfo {
                endCursor
                hasNextPage
            }
        }
    }

    ${listExternalServiceFragment}
`

export function queryExternalServices(
    variables: ExternalServicesVariables
): Observable<ExternalServicesResult['externalServices']> {
    return requestGraphQL<ExternalServicesResult, ExternalServicesVariables>(
        gql`
            query ExternalServices($first: Int, $after: String, $namespace: ID) {
                externalServices(first: $first, after: $after, namespace: $namespace) {
                    nodes {
                        ...ListExternalServiceFields
                    }
                    totalCount
                    pageInfo {
                        endCursor
                        hasNextPage
                    }
                }
            }

            ${listExternalServiceFragment}
        `,
        variables
    ).pipe(
        map(({ data, errors }) => {
            if (!data || !data.externalServices || errors) {
                throw createAggregateError(errors)
            }
            return data.externalServices
        })
    )
}

interface ExternalServicesScopeVariables {
    namespace: Scalars['ID']
}

interface ExternalServicesScopeResult {
    externalServices: {
        nodes: {
            id: ExternalServiceFields['id']
            kind: ExternalServiceFields['kind']
            grantedScopes: string[]
        }[]
    }
}

export function queryExternalServicesScope(
    variables: ExternalServicesScopeVariables
): Observable<ExternalServicesScopeResult['externalServices']> {
    return requestGraphQL<ExternalServicesScopeResult, ExternalServicesScopeVariables>(
        gql`
            query ExternalServicesScopes($namespace: ID!) {
                externalServices(first: null, after: null, namespace: $namespace) {
                    nodes {
                        id
                        kind
                        grantedScopes
                    }
                }
            }
        `,
        variables
    ).pipe(
        map(({ data, errors }) => {
            if (!data || !data.externalServices || errors) {
                throw createAggregateError(errors)
            }
            return data.externalServices
        })
    )
}
