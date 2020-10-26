import React from 'react'
import { OrgAreaRoute } from '../../org/area/OrgArea'
import { orgAreaRoutes } from '../../org/area/routes'
import { enterpriseNamespaceAreaRoutes } from '../namespaces/routes'
import { lazyComponent } from '../../util/lazyComponent'
import { OrgCampaignsAreaProps } from '../campaigns/global/GlobalCampaignsArea'

const OrgCampaignsArea = lazyComponent<OrgCampaignsAreaProps, 'OrgCampaignsArea'>(
    () => import('../campaigns/global/GlobalCampaignsArea'),
    'OrgCampaignsArea'
)

export const enterpriseOrganizationAreaRoutes: readonly OrgAreaRoute[] = [
    ...orgAreaRoutes,
    ...enterpriseNamespaceAreaRoutes,
    {
        path: '/campaigns',
        render: props => <OrgCampaignsArea {...props} orgID={props.org.id} />,
        condition: props => !props.isSourcegraphDotCom && window.context.campaignsEnabled,
    },
]
