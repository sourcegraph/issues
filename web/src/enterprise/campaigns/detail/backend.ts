import { map } from 'rxjs/operators'
import { dataOrThrowErrors, gql } from '../../../../../shared/src/graphql/graphql'
import { queryGraphQL, mutateGraphQL } from '../../../backend/graphql'
import { Observable } from 'rxjs'
import { ID } from '../../../../../shared/src/graphql/schema'
import { DiffStatFields, RepositoryComparisonFields } from '../../../backend/diff'
import { FilteredConnectionQueryArgs } from '../../../components/FilteredConnection'
import {
    CampaignByIDResult,
    CampaignChangesetsVariables,
    CampaignChangesetsResult,
    ExternalChangesetFileDiffsResult,
    CampaignFields,
    SyncChangesetResult,
} from '../../../graphql-operations'

const campaignFragment = gql`
    fragment CampaignFields on Campaign {
        __typename
        id
        name
        description
        author {
            username
            avatarURL
        }
        branch
        createdAt
        updatedAt
        closedAt
        viewerCanAdminister
        changesets {
            totalCount
            stats {
                total
                closed
                merged
            }
        }
        # TODO move to separate query and configure from/to
        changesetCountsOverTime {
            date
            merged
            closed
            openApproved
            openChangesRequested
            openPending
            total
        }
        diffStat {
            ...DiffStatFields
        }
    }

    ${DiffStatFields}
`

const changesetLabelFragment = gql`
    fragment ChangesetLabelFields on ChangesetLabel {
        color
        description
        text
    }
`

export const fetchCampaignById = (campaign: ID): Observable<CampaignFields | null> =>
    queryGraphQL<CampaignByIDResult>(
        gql`
            query CampaignByID($campaign: ID!) {
                node(id: $campaign) {
                    __typename
                    ... on Campaign {
                        ...CampaignFields
                    }
                }
            }
            ${campaignFragment}
        `,
        { campaign }
    ).pipe(
        map(dataOrThrowErrors),
        map(({ node }) => {
            if (!node) {
                return null
            }
            if (node.__typename !== 'Campaign') {
                throw new Error(`The given ID is a ${node.__typename}, not a Campaign`)
            }
            return node
        })
    )

export const ChangesetFieldsFragment = gql`
    fragment ChangesetFields on Changeset {
        __typename

        state
        createdAt
        updatedAt
        nextSyncAt
        externalState
        ... on HiddenExternalChangeset {
            id
        }
        ... on ExternalChangeset {
            id
            title
            body
            reviewState
            checkState
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
            diff {
                __typename
                ... on PreviewRepositoryComparison {
                    fileDiffs {
                        diffStat {
                            ...DiffStatFields
                        }
                    }
                }
                ... on RepositoryComparison {
                    fileDiffs {
                        diffStat {
                            ...DiffStatFields
                        }
                    }
                }
            }
            diffStat {
                added
                changed
                deleted
            }
        }
    }

    ${DiffStatFields}

    ${changesetLabelFragment}
`

export const queryChangesets = ({
    campaign,
    first,
    state,
    reviewState,
    checkState,
}: CampaignChangesetsVariables): Observable<
    (CampaignChangesetsResult['node'] & { __typename: 'Campaign' })['changesets']
> =>
    queryGraphQL<CampaignChangesetsResult>(
        gql`
            query CampaignChangesets(
                $campaign: ID!
                $first: Int
                $state: ChangesetState
                $reviewState: ChangesetReviewState
                $checkState: ChangesetCheckState
            ) {
                node(id: $campaign) {
                    __typename
                    ... on Campaign {
                        changesets(first: $first, state: $state, reviewState: $reviewState, checkState: $checkState) {
                            totalCount
                            nodes {
                                ...ChangesetFields
                            }
                        }
                    }
                }
            }

            ${ChangesetFieldsFragment}
        `,
        { campaign, first, state, reviewState, checkState }
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

export async function syncChangeset(changeset: ID): Promise<void> {
    const result = await mutateGraphQL<SyncChangesetResult>(
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

export type ExternalChangesetGraphQlNode = ExternalChangesetFileDiffsResult['node'] & {
    __typename: 'ExternalChangeset'
}

export const queryExternalChangesetWithFileDiffs = (
    externalChangeset: ID,
    { first, after, isLightTheme }: FilteredConnectionQueryArgs & { isLightTheme: boolean }
): Observable<ExternalChangesetGraphQlNode> =>
    queryGraphQL<ExternalChangesetFileDiffsResult>(
        gql`
            query ExternalChangesetFileDiffs(
                $externalChangeset: ID!
                $first: Int
                $after: String
                $isLightTheme: Boolean!
            ) {
                node(id: $externalChangeset) {
                    __typename
                    ... on ExternalChangeset {
                        diff {
                            __typename
                            ... on RepositoryComparison {
                                ...RepositoryComparisonFields
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
                                    diffStat {
                                        ...DiffStatFields
                                    }
                                }
                            }
                        }
                    }
                }
            }

            ${RepositoryComparisonFields}
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
