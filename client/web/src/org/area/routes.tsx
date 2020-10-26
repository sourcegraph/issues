import React from 'react'
import { Redirect } from 'react-router'
import { namespaceAreaRoutes } from '../../namespaces/routes'
import { lazyComponent } from '../../util/lazyComponent'
import { OrgAreaRoute } from './OrgArea'

const OrgSettingsArea = lazyComponent(() => import('../settings/OrgSettingsArea'), 'OrgSettingsArea')

export const orgAreaRoutes: readonly OrgAreaRoute[] = [
    {
        path: '',
        exact: true,
        render: lazyComponent(() => import('./OrgOverviewPage'), 'OrgOverviewPage'),
    },
    {
        path: '/members',
        render: lazyComponent(() => import('./OrgMembersPage'), 'OrgMembersPage'),
    },
    {
        path: '/settings',
        render: props => <OrgSettingsArea {...props} isLightTheme={props.isLightTheme} />,
    },
    ...namespaceAreaRoutes,

    // Redirect from previous /orgs/:orgname/account -> /orgs/:orgname/settings/profile.
    {
        path: '/account',
        render: props => <Redirect to={`${props.match.url}/settings/profile`} />,
    },
]
