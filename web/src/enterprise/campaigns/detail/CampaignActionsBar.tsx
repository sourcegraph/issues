import React from 'react'
import * as GQL from '../../../../../shared/src/graphql/schema'
import { CampaignsIcon } from '../icons'
import { Link } from '../../../../../shared/src/components/Link'

interface Props {
    campaign: Pick<GQL.Campaign, 'name' | 'closedAt' | 'viewerCanAdminister'> & {
        changesets: {
            totalCount: GQL.Campaign['changesets']['totalCount']
            stats: Pick<GQL.Campaign['changesets']['stats'], 'total' | 'closed' | 'merged'>
        }
    }
}

export const CampaignActionsBar: React.FunctionComponent<Props> = ({ campaign }) => {
    const campaignClosed = !!campaign.closedAt

    // const percentComplete = (
    //     (((campaign.changesets.stats.closed as number) + (campaign.changesets.stats.merged as number)) /
    //         campaign.changesets.stats.total) *
    //     100
    // ).toFixed(0)

    return (
        <>
            <div className="mb-2">
                <span>
                    <Link to="/campaigns">Campaigns</Link>
                </span>
                <span className="text-muted d-inline-block mx-1">/</span>
                <span>{campaign.name}</span>
            </div>
            <div className="d-flex mb-2 position-relative">
                <div>
                    <h1 className="m-0">{campaign.name}</h1>
                    <h2 className="m-0">
                        <CampaignStateBadge isClosed={campaignClosed} />
                        <small className="text-muted">
                            {0}% complete. {campaign.changesets.totalCount} changesets total
                        </small>
                    </h2>
                </div>
            </div>
        </>
    )
}

const CampaignStateBadge: React.FunctionComponent<{ isClosed: boolean }> = ({ isClosed }) => {
    if (isClosed) {
        return (
            <span className="badge badge-danger mr-2">
                <CampaignsIcon className="icon-inline campaign-actions-bar__campaign-icon" /> Closed
            </span>
        )
    }
    return (
        <span className="badge badge-success mr-2">
            <CampaignsIcon className="icon-inline campaign-actions-bar__campaign-icon" /> Open
        </span>
    )
}
