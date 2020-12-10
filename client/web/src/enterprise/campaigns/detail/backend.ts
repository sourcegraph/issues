import { map } from 'rxjs/operators'
import { dataOrThrowErrors, gql } from '../../../../../shared/src/graphql/graphql'
import { Observable } from 'rxjs'
import { diffStatFields, fileDiffFields } from '../../../backend/diff'
import {
    CampaignFields,
    CampaignChangesetsVariables,
    CampaignChangesetsResult,
    ExternalChangesetFileDiffsResult,
    ExternalChangesetFileDiffsVariables,
    ExternalChangesetFileDiffsFields,
    SyncChangesetResult,
    SyncChangesetVariables,
    Scalars,
    ChangesetCountsOverTimeVariables,
    ChangesetCountsOverTimeFields,
    ChangesetCountsOverTimeResult,
    DeleteCampaignResult,
    DeleteCampaignVariables,
    CampaignByNamespaceResult,
    CampaignByNamespaceVariables,
    ChangesetDiffResult,
    ChangesetDiffVariables,
} from '../../../graphql-operations'
import { requestGraphQL } from '../../../backend/graphql'

const changesetsStatsFragment = gql`
    fragment ChangesetsStatsFields on ChangesetsStats {
        total
        closed
        deleted
        draft
        merged
        open
        unpublished
    }
`

const campaignFragment = gql`
    fragment CampaignFields on Campaign {
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

        updatedAt
        closedAt
        viewerCanAdminister

        changesetsStats {
            ...ChangesetsStatsFields
        }

        currentSpec {
            originalInput
            supersedingCampaignSpec {
                createdAt
                applyURL
            }
        }
    }

    ${changesetsStatsFragment}
`

const changesetLabelFragment = gql`
    fragment ChangesetLabelFields on ChangesetLabel {
        color
        description
        text
    }
`

export const fetchCampaignByNamespace = (
    namespaceID: Scalars['ID'],
    campaign: CampaignFields['name']
): Observable<CampaignFields | null> =>
    requestGraphQL<CampaignByNamespaceResult, CampaignByNamespaceVariables>(
        gql`
            query CampaignByNamespace($namespaceID: ID!, $campaign: String!) {
                campaign(namespace: $namespaceID, name: $campaign) {
                    ...CampaignFields
                }
            }
            ${campaignFragment}
        `,
        { namespaceID, campaign }
    ).pipe(
        map(dataOrThrowErrors),
        map(({ campaign }) => {
            if (!campaign) {
                return null
            }
            return campaign
        })
    )

export const hiddenExternalChangesetFieldsFragment = gql`
    fragment HiddenExternalChangesetFields on HiddenExternalChangeset {
        __typename
        id
        createdAt
        updatedAt
        nextSyncAt
        externalState
        publicationState
        reconcilerState
    }
`
export const externalChangesetFieldsFragment = gql`
    fragment ExternalChangesetFields on ExternalChangeset {
        __typename
        id
        title
        body
        publicationState
        reconcilerState
        externalState
        reviewState
        checkState
        error
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
    campaign,
    first,
    after,
    externalState,
    reviewState,
    checkState,
    publicationState,
    reconcilerState,
    onlyPublishedByThisCampaign,
    search,
}: CampaignChangesetsVariables): Observable<
    (CampaignChangesetsResult['node'] & { __typename: 'Campaign' })['changesets']
> =>
    requestGraphQL<CampaignChangesetsResult, CampaignChangesetsVariables>(
        gql`
            query CampaignChangesets(
                $campaign: ID!
                $first: Int
                $after: String
                $externalState: ChangesetExternalState
                $reviewState: ChangesetReviewState
                $checkState: ChangesetCheckState
                $publicationState: ChangesetPublicationState
                $reconcilerState: [ChangesetReconcilerState!]
                $onlyPublishedByThisCampaign: Boolean
                $search: String
            ) {
                node(id: $campaign) {
                    __typename
                    ... on Campaign {
                        changesets(
                            first: $first
                            after: $after
                            externalState: $externalState
                            publicationState: $publicationState
                            reconcilerState: $reconcilerState
                            reviewState: $reviewState
                            checkState: $checkState
                            onlyPublishedByThisCampaign: $onlyPublishedByThisCampaign
                            search: $search
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
            campaign,
            first,
            after,
            externalState,
            reviewState,
            checkState,
            publicationState,
            reconcilerState,
            onlyPublishedByThisCampaign,
            search,
        }
    ).pipe(
        map(dataOrThrowErrors),
        map(({ node }) => {
            if (!node) {
                throw new Error(`Campaign with ID ${campaign} does not exist`)
            }
            if (node.__typename !== 'Campaign') {
                throw new Error(`The given ID is a ${node.__typename}, not a Campaign`)
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
    campaign,
    from,
    to,
}: ChangesetCountsOverTimeVariables): Observable<ChangesetCountsOverTimeFields[]> =>
    requestGraphQL<ChangesetCountsOverTimeResult, ChangesetCountsOverTimeVariables>(
        gql`
            query ChangesetCountsOverTime($campaign: ID!, $from: DateTime, $to: DateTime) {
                node(id: $campaign) {
                    __typename
                    ... on Campaign {
                        changesetCountsOverTime(from: $from, to: $to) {
                            ...ChangesetCountsOverTimeFields
                        }
                    }
                }
            }

            ${changesetCountsOverTimeFragment}
        `,
        { campaign, from, to }
    ).pipe(
        map(dataOrThrowErrors),
        map(({ node }) => {
            if (!node) {
                throw new Error(`Campaign with ID ${campaign} does not exist`)
            }
            if (node.__typename !== 'Campaign') {
                throw new Error(`The given ID is a ${node.__typename}, not a Campaign`)
            }
            return node.changesetCountsOverTime
        })
    )

export async function deleteCampaign(campaign: Scalars['ID']): Promise<void> {
    const result = await requestGraphQL<DeleteCampaignResult, DeleteCampaignVariables>(
        gql`
            mutation DeleteCampaign($campaign: ID!) {
                deleteCampaign(campaign: $campaign) {
                    alwaysNil
                }
            }
        `,
        { campaign }
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
