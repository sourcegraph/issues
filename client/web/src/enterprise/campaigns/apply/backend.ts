import { diffStatFields, fileDiffFields } from '../../../backend/diff'
import { gql, dataOrThrowErrors } from '../../../../../shared/src/graphql/graphql'
import {
    Scalars,
    CampaignSpecFields,
    CampaignSpecByIDVariables,
    CampaignSpecByIDResult,
    CampaignSpecChangesetSpecsResult,
    CampaignSpecChangesetSpecsVariables,
    ChangesetSpecFileDiffsVariables,
    ChangesetSpecFileDiffsResult,
    ChangesetSpecFileDiffsFields,
    CreateCampaignVariables,
    CreateCampaignResult,
    ApplyCampaignResult,
    ApplyCampaignVariables,
} from '../../../graphql-operations'
import { Observable } from 'rxjs'
import { map } from 'rxjs/operators'
import { requestGraphQL } from '../../../backend/graphql'

export const campaignSpecFragment = gql`
    fragment CampaignSpecFields on CampaignSpec {
        id
        description {
            name
            description
        }
        appliesToCampaign {
            id
            name
            url
        }
        createdAt
        creator {
            username
            url
        }
        expiresAt
        namespace {
            namespaceName
            url
        }
        viewerCanAdminister
        diffStat {
            ...DiffStatFields
        }
    }

    ${diffStatFields}
`

export const fetchCampaignSpecById = (campaignSpec: Scalars['ID']): Observable<CampaignSpecFields | null> =>
    requestGraphQL<CampaignSpecByIDResult, CampaignSpecByIDVariables>(
        gql`
            query CampaignSpecByID($campaignSpec: ID!) {
                node(id: $campaignSpec) {
                    __typename
                    ... on CampaignSpec {
                        ...CampaignSpecFields
                    }
                }
            }
            ${campaignSpecFragment}
        `,
        { campaignSpec }
    ).pipe(
        map(dataOrThrowErrors),
        map(({ node }) => {
            if (!node) {
                return null
            }
            if (node.__typename !== 'CampaignSpec') {
                throw new Error(`The given ID is a ${node.__typename}, not a CampaignSpec`)
            }
            return node
        })
    )

export const changesetSpecFieldsFragment = gql`
    fragment ChangesetSpecFields on ChangesetSpec {
        __typename
        ... on HiddenChangesetSpec {
            ...HiddenChangesetSpecFields
        }
        ... on VisibleChangesetSpec {
            ...VisibleChangesetSpecFields
        }
    }

    fragment HiddenChangesetSpecFields on HiddenChangesetSpec {
        __typename
        id
        expiresAt
        type
    }

    fragment VisibleChangesetSpecFields on VisibleChangesetSpec {
        __typename
        id
        expiresAt
        type
        description {
            __typename
            ...ExistingChangesetReferenceFields
            ...GitBranchChangesetDescriptionFields
        }
    }

    fragment ExistingChangesetReferenceFields on ExistingChangesetReference {
        baseRepository {
            name
            url
        }
        externalID
    }

    fragment GitBranchChangesetDescriptionFields on GitBranchChangesetDescription {
        baseRepository {
            name
            url
        }
        title
        published
        body
        commits {
            author {
                avatarURL
                email
                displayName
                user {
                    username
                    displayName
                    url
                }
            }
            subject
            body
        }
        baseRef
        headRef
        diffStat {
            ...DiffStatFields
        }
    }

    ${diffStatFields}
`

export const queryChangesetSpecs = ({
    campaignSpec,
    first,
    after,
}: CampaignSpecChangesetSpecsVariables): Observable<
    (CampaignSpecChangesetSpecsResult['node'] & { __typename: 'CampaignSpec' })['changesetSpecs']
> =>
    requestGraphQL<CampaignSpecChangesetSpecsResult, CampaignSpecChangesetSpecsVariables>(
        gql`
            query CampaignSpecChangesetSpecs($campaignSpec: ID!, $first: Int, $after: String) {
                node(id: $campaignSpec) {
                    __typename
                    ... on CampaignSpec {
                        changesetSpecs(first: $first, after: $after) {
                            totalCount
                            pageInfo {
                                endCursor
                                hasNextPage
                            }
                            nodes {
                                ...ChangesetSpecFields
                            }
                        }
                    }
                }
            }

            ${changesetSpecFieldsFragment}
        `,
        { campaignSpec, first, after }
    ).pipe(
        map(dataOrThrowErrors),
        map(({ node }) => {
            if (!node) {
                throw new Error(`CampaignSpec with ID ${campaignSpec} does not exist`)
            }
            if (node.__typename !== 'CampaignSpec') {
                throw new Error(`The given ID is a ${node.__typename}, not a CampaignSpec`)
            }
            return node.changesetSpecs
        })
    )

export const changesetSpecFileDiffsFields = gql`
    fragment ChangesetSpecFileDiffsFields on VisibleChangesetSpec {
        description {
            __typename
            ... on GitBranchChangesetDescription {
                diff {
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
    }

    ${fileDiffFields}
`

export const queryChangesetSpecFileDiffs = ({
    changesetSpec,
    first,
    after,
    isLightTheme,
}: ChangesetSpecFileDiffsVariables): Observable<
    (ChangesetSpecFileDiffsFields['description'] & { __typename: 'GitBranchChangesetDescription' })['diff']
> =>
    requestGraphQL<ChangesetSpecFileDiffsResult, ChangesetSpecFileDiffsVariables>(
        gql`
            query ChangesetSpecFileDiffs($changesetSpec: ID!, $first: Int, $after: String, $isLightTheme: Boolean!) {
                node(id: $changesetSpec) {
                    __typename
                    ...ChangesetSpecFileDiffsFields
                }
            }

            ${changesetSpecFileDiffsFields}
        `,
        { changesetSpec, first, after, isLightTheme }
    ).pipe(
        map(dataOrThrowErrors),
        map(({ node }) => {
            if (!node) {
                throw new Error(`ChangesetSpec with ID ${changesetSpec} does not exist`)
            }
            if (node.__typename !== 'VisibleChangesetSpec') {
                throw new Error(`The given ID is a ${node.__typename}, not a VisibleChangesetSpec`)
            }
            if (node.description.__typename !== 'GitBranchChangesetDescription') {
                throw new Error('The given ChangesetSpec is no GitBranchChangesetDescription')
            }
            return node.description.diff
        })
    )

export const createCampaign = ({
    campaignSpec,
}: CreateCampaignVariables): Promise<CreateCampaignResult['createCampaign']> =>
    requestGraphQL<CreateCampaignResult, CreateCampaignVariables>(
        gql`
            mutation CreateCampaign($campaignSpec: ID!) {
                createCampaign(campaignSpec: $campaignSpec) {
                    id
                    url
                }
            }
        `,
        { campaignSpec }
    )
        .pipe(
            map(dataOrThrowErrors),
            map(data => data.createCampaign)
        )
        .toPromise()

export const applyCampaign = ({
    campaignSpec,
    campaign,
}: ApplyCampaignVariables): Promise<ApplyCampaignResult['applyCampaign']> =>
    requestGraphQL<ApplyCampaignResult, ApplyCampaignVariables>(
        gql`
            mutation ApplyCampaign($campaignSpec: ID!, $campaign: ID!) {
                applyCampaign(campaignSpec: $campaignSpec, ensureCampaign: $campaign) {
                    id
                    url
                }
            }
        `,
        { campaignSpec, campaign }
    )
        .pipe(
            map(dataOrThrowErrors),
            map(data => data.applyCampaign)
        )
        .toPromise()
