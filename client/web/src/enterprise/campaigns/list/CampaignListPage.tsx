import React, { useEffect, useCallback } from 'react'
import { queryCampaigns as _queryCampaigns, queryCampaignsByUser, queryCampaignsByOrg } from './backend'
import { RouteComponentProps } from 'react-router'
import { FilteredConnection, FilteredConnectionFilter } from '../../../components/FilteredConnection'
import { CampaignNode, CampaignNodeProps } from './CampaignNode'
import { TelemetryProps } from '../../../../../shared/src/telemetry/telemetryService'
import {
    ListCampaign,
    CampaignState,
    Scalars,
    CampaignsByUserVariables,
    CampaignsByOrgVariables,
} from '../../../graphql-operations'
import PlusIcon from 'mdi-react/PlusIcon'
import { Link } from '../../../../../shared/src/components/Link'
import { PageHeader } from '../../../components/PageHeader'
import { CampaignsIconFlushLeft } from '../icons'

export interface CampaignListPageProps extends TelemetryProps, Pick<RouteComponentProps, 'history' | 'location'> {
    displayNamespace?: boolean
    queryCampaigns?: typeof _queryCampaigns
}

const FILTERS: FilteredConnectionFilter[] = [
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
    {
        label: 'All',
        id: 'all',
        tooltip: 'Show all campaigns',
        args: {},
    },
]

/**
 * A list of all campaigns on the Sourcegraph instance.
 */
export const CampaignListPage: React.FunctionComponent<CampaignListPageProps> = ({
    queryCampaigns = _queryCampaigns,
    displayNamespace = true,
    location,
    ...props
}) => {
    useEffect(() => props.telemetryService.logViewEvent('CampaignsListPage'), [props.telemetryService])
    return (
        <>
            <PageHeader
                icon={CampaignsIconFlushLeft}
                title="Campaigns"
                className="justify-content-end test-campaign-list-page"
                actions={
                    <Link to={`${location.pathname}/create`} className="btn btn-primary">
                        <PlusIcon className="icon-inline" /> New campaign
                    </Link>
                }
            />
            <FilteredConnection<ListCampaign, Omit<CampaignNodeProps, 'node'>>
                {...props}
                location={location}
                nodeComponent={CampaignNode}
                nodeComponentProps={{ history: props.history, displayNamespace }}
                queryConnection={queryCampaigns}
                hideSearch={true}
                defaultFirst={15}
                filters={FILTERS}
                noun="campaign"
                pluralNoun="campaigns"
                listComponent="div"
                listClassName="campaign-list-page__grid mb-3"
                className="mb-3"
                cursorPaging={true}
                noSummaryIfAllNodesVisible={true}
            />
        </>
    )
}

export interface UserCampaignListPageProps extends CampaignListPageProps {
    userID: Scalars['ID']
}

/**
 * A list of all campaigns in a users namespace.
 */
export const UserCampaignListPage: React.FunctionComponent<UserCampaignListPageProps> = ({ userID, ...props }) => {
    const queryConnection = useCallback(
        (args: Partial<CampaignsByUserVariables>) =>
            queryCampaignsByUser({
                userID,
                first: args.first ?? null,
                after: args.after ?? null,
                // The types for FilteredConnectionQueryArguments don't allow access to the filter arguments.
                state: (args as { state: CampaignState | undefined }).state ?? null,
                viewerCanAdminister: null,
            }),
        [userID]
    )
    return <CampaignListPage {...props} displayNamespace={false} queryCampaigns={queryConnection} />
}

export interface OrgCampaignListPageProps extends CampaignListPageProps {
    orgID: Scalars['ID']
}

/**
 * A list of all campaigns in an orgs namespace.
 */
export const OrgCampaignListPage: React.FunctionComponent<OrgCampaignListPageProps> = ({ orgID, ...props }) => {
    const queryConnection = useCallback(
        (args: Partial<CampaignsByOrgVariables>) =>
            queryCampaignsByOrg({
                orgID,
                first: args.first ?? null,
                after: args.after ?? null,
                // The types for FilteredConnectionQueryArguments don't allow access to the filter arguments.
                state: (args as { state: CampaignState | undefined }).state ?? null,
                viewerCanAdminister: null,
            }),
        [orgID]
    )
    return <CampaignListPage {...props} displayNamespace={false} queryCampaigns={queryConnection} />
}
