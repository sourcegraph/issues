import React from 'react'
import { Redirect, RouteComponentProps } from 'react-router'

import { userAreaRoutes } from '../../user/area/routes'
import { UserAreaRoute, UserAreaRouteContext } from '../../user/area/UserArea'
import { lazyComponent } from '../../util/lazyComponent'
import { NamespaceBatchChangesAreaProps } from '../batches/global/GlobalBatchChangesArea'
import { SHOW_BUSINESS_FEATURES } from '../dotcom/productSubscriptions/features'
import { enterpriseNamespaceAreaRoutes } from '../namespaces/routes'

const NamespaceBatchChangesArea = lazyComponent<NamespaceBatchChangesAreaProps, 'NamespaceBatchChangesArea'>(
    () => import('../batches/global/GlobalBatchChangesArea'),
    'NamespaceBatchChangesArea'
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
        condition: () => SHOW_BUSINESS_FEATURES,
    },
    {
        path: '/campaigns',
        render: ({ location }) => <Redirect to={location.pathname.replace('/campaigns', '/batch-changes')} />,
    },
    {
        path: '/batch-changes',
        render: props => <NamespaceBatchChangesArea {...props} namespaceID={props.user.id} />,
        condition: ({ batchChangesEnabled }) => batchChangesEnabled,
    },
]
