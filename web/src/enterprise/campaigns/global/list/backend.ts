import { map } from 'rxjs/operators'
import { dataOrThrowErrors, gql } from '../../../../../../shared/src/graphql/graphql'
import { queryGraphQL } from '../../../../backend/graphql'
import { Observable } from 'rxjs'
import { CampaignsVariables, CampaignsResult } from '../../../../graphql-operations'

const ListCampaignFragment = gql`
    fragment ListCampaign on Campaign {
        id
        name
        description
        createdAt
        closedAt
        author {
            username
        }
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
    state,
    viewerCanAdminister,
}: CampaignsVariables): Observable<CampaignsResult['campaigns']> =>
    queryGraphQL<CampaignsResult>(
        gql`
            query Campaigns($first: Int, $state: CampaignState, $viewerCanAdminister: Boolean) {
                campaigns(first: $first, state: $state, viewerCanAdminister: $viewerCanAdminister) {
                    nodes {
                        ...ListCampaign
                    }
                    totalCount
                }
            }

            ${ListCampaignFragment}
        `,
        { first, state, viewerCanAdminister }
    ).pipe(
        map(dataOrThrowErrors),
        map(data => data.campaigns)
    )
