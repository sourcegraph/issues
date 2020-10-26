import React from 'react'
import { Redirect, RouteComponentProps } from 'react-router'
import { userAreaRoutes } from '../../user/area/routes'
import { UserAreaRoute, UserAreaRouteContext } from '../../user/area/UserArea'
import { enterpriseNamespaceAreaRoutes } from '../namespaces/routes'
import { lazyComponent } from '../../util/lazyComponent'
import { UserCampaignsAreaProps } from '../campaigns/global/GlobalCampaignsArea'

const UserCampaignsArea = lazyComponent<UserCampaignsAreaProps, 'UserCampaignsArea'>(
    () => import('../campaigns/global/GlobalCampaignsArea'),
    'UserCampaignsArea'
)

export const enterpriseUserAreaRoutes: readonly UserAreaRoute[] = [
    ...userAreaRoutes,
    ...enterpriseNamespaceAreaRoutes,

    // Redirect from previous /users/:username/subscriptions -> /users/:username/settings/subscriptions.
    {
        path: '/subscriptions/:page*',
        render: (props: UserAreaRouteContext & RouteComponentProps<{ page: string }>) => (
            <Redirect
                to={`${props.url}/settings/subscriptions${
                    props.match.params.page ? `/${props.match.params.page}` : ''
                }`}
            />
        ),
    },
    {
        path: '/campaigns',
        render: props => <UserCampaignsArea {...props} userID={props.user.id} />,
        condition: props => !props.isSourcegraphDotCom && window.context.campaignsEnabled,
    },
]
