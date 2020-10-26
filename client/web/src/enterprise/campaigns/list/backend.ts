import { map } from 'rxjs/operators'
import { dataOrThrowErrors, gql } from '../../../../../shared/src/graphql/graphql'
import { Observable } from 'rxjs'
import {
    CampaignsVariables,
    CampaignsResult,
    CampaignsByUserResult,
    CampaignsByUserVariables,
    CampaignsByOrgResult,
    CampaignsByOrgVariables,
} from '../../../graphql-operations'
import { requestGraphQL } from '../../../backend/graphql'

const ListCampaignFragment = gql`
    fragment ListCampaign on Campaign {
        id
        url
        name
        namespace {
            namespaceName
            url
        }
        description
        createdAt
        closedAt
        changesets {
            stats {
                open
                closed
                merged
            }
        }
    }
`

export const queryCampaigns = ({
    first,
    after,
    state,
    viewerCanAdminister,
}: Partial<CampaignsVariables>): Observable<CampaignsResult['campaigns']> =>
    requestGraphQL<CampaignsResult, CampaignsVariables>(
        gql`
            query Campaigns($first: Int, $after: String, $state: CampaignState, $viewerCanAdminister: Boolean) {
                campaigns(first: $first, after: $after, state: $state, viewerCanAdminister: $viewerCanAdminister) {
                    nodes {
                        ...ListCampaign
                    }
                    pageInfo {
                        endCursor
                        hasNextPage
                    }
                    totalCount
                }
            }

            ${ListCampaignFragment}
        `,
        {
            first: first ?? null,
            after: after ?? null,
            state: state ?? null,
            viewerCanAdminister: viewerCanAdminister ?? null,
        }
    ).pipe(
        map(dataOrThrowErrors),
        map(data => data.campaigns)
    )

export const queryCampaignsByUser = ({
    userID,
    first,
    after,
    state,
    viewerCanAdminister,
}: CampaignsByUserVariables): Observable<CampaignsResult['campaigns']> =>
    requestGraphQL<CampaignsByUserResult, CampaignsByUserVariables>(
        gql`
            query CampaignsByUser(
                $userID: ID!
                $first: Int
                $after: String
                $state: CampaignState
                $viewerCanAdminister: Boolean
            ) {
                node(id: $userID) {
                    __typename
                    ... on User {
                        campaigns(
                            first: $first
                            after: $after
                            state: $state
                            viewerCanAdminister: $viewerCanAdminister
                        ) {
                            nodes {
                                ...ListCampaign
                            }
                            pageInfo {
                                endCursor
                                hasNextPage
                            }
                            totalCount
                        }
                    }
                }
            }

            ${ListCampaignFragment}
        `,
        { first, after, state, viewerCanAdminister, userID }
    ).pipe(
        map(dataOrThrowErrors),
        map(data => {
            if (!data.node) {
                throw new Error('User not found')
            }
            if (data.node.__typename !== 'User') {
                throw new Error(`Requested node is a ${data.node.__typename}, not a User`)
            }
            return data.node.campaigns
        })
    )

export const queryCampaignsByOrg = ({
    orgID,
    first,
    after,
    state,
    viewerCanAdminister,
}: CampaignsByOrgVariables): Observable<CampaignsResult['campaigns']> =>
    requestGraphQL<CampaignsByOrgResult, CampaignsByOrgVariables>(
        gql`
            query CampaignsByOrg(
                $orgID: ID!
                $first: Int
                $after: String
                $state: CampaignState
                $viewerCanAdminister: Boolean
            ) {
                node(id: $orgID) {
                    __typename
                    ... on Org {
                        campaigns(
                            first: $first
                            after: $after
                            state: $state
                            viewerCanAdminister: $viewerCanAdminister
                        ) {
                            nodes {
                                ...ListCampaign
                            }
                            pageInfo {
                                endCursor
                                hasNextPage
                            }
                            totalCount
                        }
                    }
                }
            }

            ${ListCampaignFragment}
        `,
        { first, after, state, viewerCanAdminister, orgID }
    ).pipe(
        map(dataOrThrowErrors),
        map(data => {
            if (!data.node) {
                throw new Error('Org not found')
            }
            if (data.node.__typename !== 'Org') {
                throw new Error(`Requested node is a ${data.node.__typename}, not an Org`)
            }
            return data.node.campaigns
        })
    )
