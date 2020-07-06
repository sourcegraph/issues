import React, { useMemo } from 'react'
import { queryCampaigns, queryCampaignsCount as _queryCampaignsCount } from './backend'
import AddIcon from 'mdi-react/AddIcon'
import { Link } from '../../../../../../shared/src/components/Link'
import { RouteComponentProps } from 'react-router'
import { FilteredConnection, FilteredConnectionFilter } from '../../../../components/FilteredConnection'
import { IUser, CampaignState } from '../../../../../../shared/src/graphql/schema'
import { CampaignNode, CampaignNodeCampaign, CampaignNodeProps } from '../../list/CampaignNode'
import { useObservable } from '../../../../../../shared/src/util/useObservable'
import { Observable } from 'rxjs'

interface Props extends Pick<RouteComponentProps, 'history' | 'location'> {
    authenticatedUser: IUser | null
    queryCampaignsCount?: () => Observable<number>
}

const FILTERS: FilteredConnectionFilter[] = [
    {
        label: 'All',
        id: 'all',
        tooltip: 'Show all campaigns',
        args: {},
    },
    {
        label: 'Open',
        id: 'open',
        tooltip: 'Show only campaigns that are open',
        args: { state: CampaignState.OPEN },
    },
    {
        label: 'Closed',
        id: 'closed',
        tooltip: 'Show only campaigns that are closed',
        args: { state: CampaignState.CLOSED },
    },
]

/**
 * A list of all campaigns on the Sourcegraph instance.
 */
export const GlobalCampaignListPage: React.FunctionComponent<Props> = ({
    queryCampaignsCount = _queryCampaignsCount,
    ...props
}) => {
    const totalCount = useObservable(useMemo(() => queryCampaignsCount(), [queryCampaignsCount]))
    return (
        <>
            <div className="d-flex justify-content-between align-items-end mb-3">
                <div>
                    <h1 className="mb-2">
                        Campaigns <span className="badge badge-info">Beta</span>
                    </h1>
                    <p className="mb-0">
                        Perform and track large-scale code changes.{' '}
                        <a href="https://docs.sourcegraph.com/user/campaigns">Learn how.</a>
                    </p>
                </div>
                {props.authenticatedUser?.siteAdmin && (
                    <Link to="/campaigns/create" className="btn btn-primary ml-3">
                        <AddIcon className="icon-inline" /> New campaign
                    </Link>
                )}
            </div>

            <div className="card mt-4 mb-4">
                <div className="card-body p-3">
                    <h3>
                        Welcome to campaigns <span className="badge badge-info">Beta</span>!
                    </h3>
                    <p className="mb-1">
                        We're excited for you to use campaigns to remove legacy code, fix critical security issues, pay
                        down tech debt, and more. We look forward to hearing about campaigns you run inside your
                        organization. Take a look at some{' '}
                        <a href="https://docs.sourcegraph.com/user/campaigns/examples">examples in our documentation</a>
                        , and <a href="mailto:feedback@sourcegraph.com?subject=Campaigns feedback">get in touch</a> with
                        any questions or feedback!
                    </p>
                </div>
            </div>

            {typeof totalCount === 'number' && totalCount > 0 && (
                <FilteredConnection<CampaignNodeCampaign, Omit<CampaignNodeProps, 'node'>>
                    {...props}
                    nodeComponent={CampaignNode}
                    nodeComponentProps={{ history: props.history }}
                    queryConnection={queryCampaigns}
                    hideSearch={true}
                    filters={FILTERS}
                    noun="campaign"
                    pluralNoun="campaigns"
                    className="mb-3"
                />
            )}
        </>
    )
}
