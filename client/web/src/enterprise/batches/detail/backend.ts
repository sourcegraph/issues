import { EMPTY, Observable } from 'rxjs'
import { expand, map, reduce } from 'rxjs/operators'

import { dataOrThrowErrors, gql } from '@sourcegraph/shared/src/graphql/graphql'

import { diffStatFields, fileDiffFields } from '../../../backend/diff'
import { requestGraphQL } from '../../../backend/graphql'
import {
    BatchChangeChangesetsVariables,
    BatchChangeChangesetsResult,
    BatchChangeFields,
    BatchChangeByNamespaceResult,
    BatchChangeByNamespaceVariables,
    ExternalChangesetFileDiffsResult,
    ExternalChangesetFileDiffsVariables,
    ExternalChangesetFileDiffsFields,
    SyncChangesetResult,
    SyncChangesetVariables,
    Scalars,
    ChangesetCountsOverTimeVariables,
    ChangesetCountsOverTimeFields,
    ChangesetCountsOverTimeResult,
    DeleteBatchChangeResult,
    ChangesetDiffResult,
    ChangesetDiffVariables,
    ReenqueueChangesetVariables,
    ReenqueueChangesetResult,
    ChangesetFields,
    DeleteBatchChangeVariables,
    DetachChangesetsVariables,
    DetachChangesetsResult,
    ChangesetScheduleEstimateResult,
    ChangesetScheduleEstimateVariables,
    CreateChangesetCommentsResult,
    CreateChangesetCommentsVariables,
    BatchChangeBulkOperationsResult,
    BatchChangeBulkOperationsVariables,
    BulkOperationConnectionFields,
    AllChangesetIDsResult,
    AllChangesetIDsVariables,
    ChangesetIDConnectionFields,
    ReenqueueChangesetsResult,
    ReenqueueChangesetsVariables,
    MergeChangesetsResult,
    MergeChangesetsVariables,
    CloseChangesetsResult,
    CloseChangesetsVariables,
    PublishChangesetsResult,
    PublishChangesetsVariables,
} from '../../../graphql-operations'

const changesetsStatsFragment = gql`
    fragment ChangesetsStatsFields on ChangesetsStats {
        total
        closed
        deleted
        draft
        merged
        open
        unpublished
        archived
    }
`

const bulkOperationFragment = gql`
    fragment BulkOperationFields on BulkOperation {
        id
        type
        state
        progress
        errors {
            changeset {
                __typename
                ... on ExternalChangeset {
                    title
                    externalURL {
                        url
                    }
                    repository {
                        name
                        url
                    }
                }
            }
            error
        }
        initiator {
            username
            url
        }
        changesetCount
        createdAt
        finishedAt
    }
`

const batchChangeFragment = gql`
    fragment BatchChangeFields on BatchChange {
        __typename
        id
        url
        name
        namespace {
            namespaceName
            url
        }
        description

        createdAt
        initialApplier {
            username
            url
        }

        lastAppliedAt
        lastApplier {
            username
            url
        }

        diffStat {
            ...DiffStatFields
        }

        updatedAt
        closedAt
        viewerCanAdminister

        changesetsStats {
            ...ChangesetsStatsFields
        }

        bulkOperations(first: 0) {
            totalCount
        }

        activeBulkOperations: bulkOperations(first: 50, createdAfter: $createdAfter) {
            ...ActiveBulkOperationsConnectionFields
        }

        currentSpec {
            originalInput
            supersedingBatchSpec {
                createdAt
                applyURL
            }
        }
    }

    fragment ActiveBulkOperationsConnectionFields on BulkOperationConnection {
        totalCount
        nodes {
            ...ActiveBulkOperationFields
        }
    }

    ${changesetsStatsFragment}

    ${diffStatFields}

    fragment ActiveBulkOperationFields on BulkOperation {
        id
        state
    }
`

const changesetLabelFragment = gql`
    fragment ChangesetLabelFields on ChangesetLabel {
        color
        description
        text
    }
`

export const fetchBatchChangeByNamespace = (
    namespaceID: Scalars['ID'],
    batchChange: BatchChangeFields['name'],
    createdAfter: BatchChangeByNamespaceVariables['createdAfter']
): Observable<BatchChangeFields | null> =>
    requestGraphQL<BatchChangeByNamespaceResult, BatchChangeByNamespaceVariables>(
        gql`
            query BatchChangeByNamespace($namespaceID: ID!, $batchChange: String!, $createdAfter: DateTime!) {
                batchChange(namespace: $namespaceID, name: $batchChange) {
                    ...BatchChangeFields
                }
            }
            ${batchChangeFragment}
        `,
        { namespaceID, batchChange, createdAfter }
    ).pipe(
        map(dataOrThrowErrors),
        map(({ batchChange }) => {
            if (!batchChange) {
                return null
            }
            return batchChange
        })
    )

export const hiddenExternalChangesetFieldsFragment = gql`
    fragment HiddenExternalChangesetFields on HiddenExternalChangeset {
        __typename
        id
        createdAt
        updatedAt
        nextSyncAt
        state
    }
`
export const externalChangesetFieldsFragment = gql`
    fragment ExternalChangesetFields on ExternalChangeset {
        __typename
        id
        title
        body
        state
        reviewState
        checkState
        error
        syncerError
        labels {
            ...ChangesetLabelFields
        }
        repository {
            id
            name
            url
        }
        externalURL {
            url
        }
        externalID
        diffStat {
            ...DiffStatFields
        }
        createdAt
        updatedAt
        nextSyncAt
        currentSpec {
            id
            type
            description {
                __typename
                ... on GitBranchChangesetDescription {
                    headRef
                }
            }
        }
    }

    ${diffStatFields}

    ${changesetLabelFragment}
`

export const changesetFieldsFragment = gql`
    fragment ChangesetFields on Changeset {
        __typename
        ... on HiddenExternalChangeset {
            ...HiddenExternalChangesetFields
        }
        ... on ExternalChangeset {
            ...ExternalChangesetFields
        }
    }

    ${hiddenExternalChangesetFieldsFragment}

    ${externalChangesetFieldsFragment}
`

export const queryChangesets = ({
    batchChange,
    first,
    after,
    state,
    reviewState,
    checkState,
    onlyPublishedByThisBatchChange,
    search,
    onlyArchived,
}: BatchChangeChangesetsVariables): Observable<
    (BatchChangeChangesetsResult['node'] & { __typename: 'BatchChange' })['changesets']
> =>
    requestGraphQL<BatchChangeChangesetsResult, BatchChangeChangesetsVariables>(
        gql`
            query BatchChangeChangesets(
                $batchChange: ID!
                $first: Int
                $after: String
                $state: ChangesetState
                $reviewState: ChangesetReviewState
                $checkState: ChangesetCheckState
                $onlyPublishedByThisBatchChange: Boolean
                $search: String
                $onlyArchived: Boolean
            ) {
                node(id: $batchChange) {
                    __typename
                    ... on BatchChange {
                        changesets(
                            first: $first
                            after: $after
                            state: $state
                            reviewState: $reviewState
                            checkState: $checkState
                            onlyPublishedByThisBatchChange: $onlyPublishedByThisBatchChange
                            search: $search
                            onlyArchived: $onlyArchived
                        ) {
                            totalCount
                            pageInfo {
                                endCursor
                                hasNextPage
                            }
                            nodes {
                                ...ChangesetFields
                            }
                        }
                    }
                }
            }

            ${changesetFieldsFragment}
        `,
        {
            batchChange,
            first,
            after,
            state,
            reviewState,
            checkState,
            onlyPublishedByThisBatchChange,
            search,
            onlyArchived,
        }
    ).pipe(
        map(dataOrThrowErrors),
        map(({ node }) => {
            if (!node) {
                throw new Error(`Batch change with ID ${batchChange} does not exist`)
            }
            if (node.__typename !== 'BatchChange') {
                throw new Error(`The given ID is a ${node.__typename}, not a BatchChange`)
            }
            return node.changesets
        })
    )

export async function syncChangeset(changeset: Scalars['ID']): Promise<void> {
    const result = await requestGraphQL<SyncChangesetResult, SyncChangesetVariables>(
        gql`
            mutation SyncChangeset($changeset: ID!) {
                syncChangeset(changeset: $changeset) {
                    alwaysNil
                }
            }
        `,
        { changeset }
    ).toPromise()
    dataOrThrowErrors(result)
}

export async function reenqueueChangeset(changeset: Scalars['ID']): Promise<ChangesetFields> {
    return requestGraphQL<ReenqueueChangesetResult, ReenqueueChangesetVariables>(
        gql`
            mutation ReenqueueChangeset($changeset: ID!) {
                reenqueueChangeset(changeset: $changeset) {
                    ...ChangesetFields
                }
            }

            ${changesetFieldsFragment}
        `,
        { changeset }
    )
        .pipe(
            map(dataOrThrowErrors),
            map(data => data.reenqueueChangeset)
        )
        .toPromise()
}

// Because thats the name in the API:
// eslint-disable-next-line unicorn/prevent-abbreviations
export const gitRefSpecFields = gql`
    fragment GitRefSpecFields on GitRevSpec {
        __typename
        ... on GitObject {
            oid
        }
        ... on GitRef {
            target {
                oid
            }
        }
        ... on GitRevSpecExpr {
            object {
                oid
            }
        }
    }
`

export const externalChangesetFileDiffsFields = gql`
    fragment ExternalChangesetFileDiffsFields on ExternalChangeset {
        diff {
            __typename
            ... on RepositoryComparison {
                range {
                    base {
                        ...GitRefSpecFields
                    }
                    head {
                        ...GitRefSpecFields
                    }
                }
                fileDiffs(first: $first, after: $after) {
                    nodes {
                        ...FileDiffFields
                    }
                    totalCount
                    pageInfo {
                        hasNextPage
                        endCursor
                    }
                }
            }
            ... on PreviewRepositoryComparison {
                fileDiffs(first: $first, after: $after) {
                    nodes {
                        ...FileDiffFields
                    }
                    totalCount
                    pageInfo {
                        hasNextPage
                        endCursor
                    }
                }
            }
        }
    }

    ${fileDiffFields}

    ${gitRefSpecFields}
`

export const queryExternalChangesetWithFileDiffs = ({
    externalChangeset,
    first,
    after,
    isLightTheme,
}: ExternalChangesetFileDiffsVariables): Observable<ExternalChangesetFileDiffsFields> =>
    requestGraphQL<ExternalChangesetFileDiffsResult, ExternalChangesetFileDiffsVariables>(
        gql`
            query ExternalChangesetFileDiffs(
                $externalChangeset: ID!
                $first: Int
                $after: String
                $isLightTheme: Boolean!
            ) {
                node(id: $externalChangeset) {
                    __typename
                    ...ExternalChangesetFileDiffsFields
                }
            }

            ${externalChangesetFileDiffsFields}
        `,
        { externalChangeset, first, after, isLightTheme }
    ).pipe(
        map(dataOrThrowErrors),
        map(({ node }) => {
            if (!node) {
                throw new Error(`Changeset with ID ${externalChangeset} does not exist`)
            }
            if (node.__typename !== 'ExternalChangeset') {
                throw new Error(`The given ID is a ${node.__typename}, not an ExternalChangeset`)
            }
            return node
        })
    )

const changesetCountsOverTimeFragment = gql`
    fragment ChangesetCountsOverTimeFields on ChangesetCounts {
        date
        merged
        closed
        draft
        openApproved
        openChangesRequested
        openPending
        total
    }
`

export const queryChangesetCountsOverTime = ({
    batchChange,
    includeArchived,
}: ChangesetCountsOverTimeVariables): Observable<ChangesetCountsOverTimeFields[]> =>
    requestGraphQL<ChangesetCountsOverTimeResult, ChangesetCountsOverTimeVariables>(
        gql`
            query ChangesetCountsOverTime($batchChange: ID!, $includeArchived: Boolean!) {
                node(id: $batchChange) {
                    __typename
                    ... on BatchChange {
                        changesetCountsOverTime(includeArchived: $includeArchived) {
                            ...ChangesetCountsOverTimeFields
                        }
                    }
                }
            }

            ${changesetCountsOverTimeFragment}
        `,
        { batchChange, includeArchived }
    ).pipe(
        map(dataOrThrowErrors),
        map(({ node }) => {
            if (!node) {
                throw new Error(`BatchChange with ID ${batchChange} does not exist`)
            }
            if (node.__typename !== 'BatchChange') {
                throw new Error(`The given ID is a ${node.__typename}, not a BatchChange`)
            }
            return node.changesetCountsOverTime
        })
    )

export async function deleteBatchChange(batchChange: Scalars['ID']): Promise<void> {
    const result = await requestGraphQL<DeleteBatchChangeResult, DeleteBatchChangeVariables>(
        gql`
            mutation DeleteBatchChange($batchChange: ID!) {
                deleteBatchChange(batchChange: $batchChange) {
                    alwaysNil
                }
            }
        `,
        { batchChange }
    ).toPromise()
    dataOrThrowErrors(result)
}

const changesetDiffFragment = gql`
    fragment ChangesetDiffFields on ExternalChangeset {
        currentSpec {
            description {
                ... on GitBranchChangesetDescription {
                    commits {
                        diff
                    }
                }
            }
        }
    }
`

export async function getChangesetDiff(changeset: Scalars['ID']): Promise<string> {
    return requestGraphQL<ChangesetDiffResult, ChangesetDiffVariables>(
        gql`
            query ChangesetDiff($changeset: ID!) {
                node(id: $changeset) {
                    __typename
                    ...ChangesetDiffFields
                }
            }

            ${changesetDiffFragment}
        `,
        { changeset }
    )
        .pipe(
            map(dataOrThrowErrors),
            map(({ node }) => {
                if (!node) {
                    throw new Error(`Changeset with ID ${changeset} does not exist`)
                } else if (node.__typename === 'HiddenExternalChangeset') {
                    throw new Error(`You do not have permission to view changeset ${changeset}`)
                } else if (node.__typename !== 'ExternalChangeset') {
                    throw new Error(`The given ID is a ${node.__typename}, not an ExternalChangeset`)
                }

                const commits = node.currentSpec?.description.commits
                if (!commits) {
                    throw new Error(`No commit available for changeset ID ${changeset}`)
                } else if (commits.length !== 1) {
                    throw new Error(`Unexpected number of commits on changeset ${changeset}: ${commits.length}`)
                }

                return commits[0].diff
            })
        )
        .toPromise()
}

const changesetScheduleEstimateFragment = gql`
    fragment ChangesetScheduleEstimateFields on ExternalChangeset {
        scheduleEstimateAt
    }
`

export async function getChangesetScheduleEstimate(changeset: Scalars['ID']): Promise<Scalars['DateTime'] | null> {
    return requestGraphQL<ChangesetScheduleEstimateResult, ChangesetScheduleEstimateVariables>(
        gql`
            query ChangesetScheduleEstimate($changeset: ID!) {
                node(id: $changeset) {
                    __typename
                    ...ChangesetScheduleEstimateFields
                }
            }

            ${changesetScheduleEstimateFragment}
        `,
        { changeset }
    )
        .pipe(
            map(dataOrThrowErrors),
            map(({ node }) => {
                if (!node) {
                    throw new Error(`Changeset with ID ${changeset} does not exist`)
                } else if (node.__typename === 'HiddenExternalChangeset') {
                    throw new Error(`You do not have permission to view changeset ${changeset}`)
                } else if (node.__typename !== 'ExternalChangeset') {
                    throw new Error(`The given ID is a ${node.__typename}, not an ExternalChangeset`)
                }

                return node.scheduleEstimateAt
            })
        )
        .toPromise()
}

export async function detachChangesets(batchChange: Scalars['ID'], changesets: Scalars['ID'][]): Promise<void> {
    const result = await requestGraphQL<DetachChangesetsResult, DetachChangesetsVariables>(
        gql`
            mutation DetachChangesets($batchChange: ID!, $changesets: [ID!]!) {
                detachChangesets(batchChange: $batchChange, changesets: $changesets) {
                    id
                }
            }
        `,
        { batchChange, changesets }
    ).toPromise()
    dataOrThrowErrors(result)
}

export async function createChangesetComments(
    batchChange: Scalars['ID'],
    changesets: Scalars['ID'][],
    body: string
): Promise<void> {
    const result = await requestGraphQL<CreateChangesetCommentsResult, CreateChangesetCommentsVariables>(
        gql`
            mutation CreateChangesetComments($batchChange: ID!, $changesets: [ID!]!, $body: String!) {
                createChangesetComments(batchChange: $batchChange, changesets: $changesets, body: $body) {
                    id
                }
            }
        `,
        { batchChange, changesets, body }
    ).toPromise()
    dataOrThrowErrors(result)
}

export async function reenqueueChangesets(batchChange: Scalars['ID'], changesets: Scalars['ID'][]): Promise<void> {
    const result = await requestGraphQL<ReenqueueChangesetsResult, ReenqueueChangesetsVariables>(
        gql`
            mutation ReenqueueChangesets($batchChange: ID!, $changesets: [ID!]!) {
                reenqueueChangesets(batchChange: $batchChange, changesets: $changesets) {
                    id
                }
            }
        `,
        { batchChange, changesets }
    ).toPromise()
    dataOrThrowErrors(result)
}

export async function mergeChangesets(
    batchChange: Scalars['ID'],
    changesets: Scalars['ID'][],
    squash: boolean
): Promise<void> {
    const result = await requestGraphQL<MergeChangesetsResult, MergeChangesetsVariables>(
        gql`
            mutation MergeChangesets($batchChange: ID!, $changesets: [ID!]!, $squash: Boolean!) {
                mergeChangesets(batchChange: $batchChange, changesets: $changesets, squash: $squash) {
                    id
                }
            }
        `,
        { batchChange, changesets, squash }
    ).toPromise()
    dataOrThrowErrors(result)
}

export async function closeChangesets(batchChange: Scalars['ID'], changesets: Scalars['ID'][]): Promise<void> {
    const result = await requestGraphQL<CloseChangesetsResult, CloseChangesetsVariables>(
        gql`
            mutation CloseChangesets($batchChange: ID!, $changesets: [ID!]!) {
                closeChangesets(batchChange: $batchChange, changesets: $changesets) {
                    id
                }
            }
        `,
        { batchChange, changesets }
    ).toPromise()
    dataOrThrowErrors(result)
}

export async function publishChangesets(
    batchChange: Scalars['ID'],
    changesets: Scalars['ID'][],
    draft: boolean
): Promise<void> {
    const result = await requestGraphQL<PublishChangesetsResult, PublishChangesetsVariables>(
        gql`
            mutation PublishChangesets($batchChange: ID!, $changesets: [ID!]!, $draft: Boolean!) {
                publishChangesets(batchChange: $batchChange, changesets: $changesets, draft: $draft) {
                    id
                }
            }
        `,
        { batchChange, changesets, draft }
    ).toPromise()
    dataOrThrowErrors(result)
}

export const queryBulkOperations = (
    args: BatchChangeBulkOperationsVariables
): Observable<BulkOperationConnectionFields> =>
    requestGraphQL<BatchChangeBulkOperationsResult, BatchChangeBulkOperationsVariables>(
        gql`
            query BatchChangeBulkOperations($batchChange: ID!, $first: Int, $after: String) {
                node(id: $batchChange) {
                    __typename
                    ... on BatchChange {
                        bulkOperations(first: $first, after: $after) {
                            ...BulkOperationConnectionFields
                        }
                    }
                }
            }

            fragment BulkOperationConnectionFields on BulkOperationConnection {
                totalCount
                pageInfo {
                    hasNextPage
                    endCursor
                }
                nodes {
                    ...BulkOperationFields
                }
            }

            ${bulkOperationFragment}
        `,
        args
    ).pipe(
        map(dataOrThrowErrors),
        map(({ node }) => {
            if (!node) {
                throw new Error(`Batch change with ID ${args.batchChange} does not exist`)
            }
            if (node.__typename !== 'BatchChange') {
                throw new Error(`The given ID is a ${node.__typename}, not a BatchChange`)
            }
            return node.bulkOperations
        })
    )

export const queryAllChangesetIDs = ({
    batchChange,
    state,
    reviewState,
    checkState,
    onlyPublishedByThisBatchChange,
    search,
    onlyArchived,
}: Omit<AllChangesetIDsVariables, 'after'>): Observable<Scalars['ID'][]> => {
    const request = (after: string | null): Observable<ChangesetIDConnectionFields> =>
        requestGraphQL<AllChangesetIDsResult, AllChangesetIDsVariables>(
            gql`
                query AllChangesetIDs(
                    $batchChange: ID!
                    $after: String
                    $state: ChangesetState
                    $reviewState: ChangesetReviewState
                    $checkState: ChangesetCheckState
                    $onlyPublishedByThisBatchChange: Boolean
                    $search: String
                    $onlyArchived: Boolean
                ) {
                    node(id: $batchChange) {
                        __typename
                        ... on BatchChange {
                            changesets(
                                first: 10000
                                after: $after
                                state: $state
                                reviewState: $reviewState
                                checkState: $checkState
                                onlyPublishedByThisBatchChange: $onlyPublishedByThisBatchChange
                                search: $search
                                onlyArchived: $onlyArchived
                            ) {
                                ...ChangesetIDConnectionFields
                            }
                        }
                    }
                }

                fragment ChangesetIDConnectionFields on ChangesetConnection {
                    nodes {
                        __typename
                        id
                    }
                    pageInfo {
                        hasNextPage
                        endCursor
                    }
                }
            `,
            {
                batchChange,
                after,
                state,
                reviewState,
                checkState,
                onlyPublishedByThisBatchChange,
                search,
                onlyArchived,
            }
        ).pipe(
            map(dataOrThrowErrors),
            map(({ node }) => {
                if (!node) {
                    throw new Error(`Batch change with ID ${batchChange} does not exist`)
                }
                if (node.__typename !== 'BatchChange') {
                    throw new Error(`The given ID is a ${node.__typename}, not a BatchChange`)
                }
                return node.changesets
            })
        )

    return request(null).pipe(
        expand(connection => (connection.pageInfo.hasNextPage ? request(connection.pageInfo.endCursor) : EMPTY)),
        reduce(
            (previous, next) =>
                previous.concat(
                    next.nodes.filter(node => node.__typename === 'ExternalChangeset').map(node => node.id)
                ),
            [] as Scalars['ID'][]
        )
    )
}
