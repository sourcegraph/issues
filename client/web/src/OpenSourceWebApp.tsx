import React from 'react'

import { extensionAreaHeaderNavItems } from './extensions/extension/extensionAreaHeaderNavItems'
import { extensionAreaRoutes } from './extensions/extension/routes'
import { extensionsAreaHeaderActionButtons } from './extensions/extensionsAreaHeaderActionButtons'
import { extensionsAreaRoutes } from './extensions/routes'
import './SourcegraphWebApp.scss'
import { KEYBOARD_SHORTCUTS } from './keyboardShortcuts/keyboardShortcuts'
import { orgAreaHeaderNavItems } from './org/area/navitems'
import { orgAreaRoutes } from './org/area/routes'
import { repoHeaderActionButtons } from './repo/repoHeaderActionButtons'
import { repoContainerRoutes, repoRevisionContainerRoutes } from './repo/routes'
import { repoSettingsAreaRoutes } from './repo/settings/routes'
import { repoSettingsSideBarGroups } from './repo/settings/sidebaritems'
import { routes } from './routes'
import { siteAdminOverviewComponents } from './site-admin/overview/overviewComponents'
import { siteAdminAreaRoutes } from './site-admin/routes'
import { siteAdminSidebarGroups } from './site-admin/sidebaritems'
import { SourcegraphWebApp } from './SourcegraphWebApp'
import { userAreaHeaderNavItems } from './user/area/navitems'
import { userAreaRoutes } from './user/area/routes'
import { userSettingsAreaRoutes } from './user/settings/routes'
import { userSettingsSideBarItems } from './user/settings/sidebaritems'

// Entry point for the app without enterprise functionality.
// For more info see: https://docs.sourcegraph.com/admin/subscriptions#paid-subscriptions-for-sourcegraph-enterprise
export const OpenSourceWebApp: React.FunctionComponent = () => (
    <SourcegraphWebApp
        extensionAreaRoutes={extensionAreaRoutes}
        extensionAreaHeaderNavItems={extensionAreaHeaderNavItems}
        extensionsAreaRoutes={extensionsAreaRoutes}
        extensionsAreaHeaderActionButtons={extensionsAreaHeaderActionButtons}
        siteAdminAreaRoutes={siteAdminAreaRoutes}
        siteAdminSideBarGroups={siteAdminSidebarGroups}
        siteAdminOverviewComponents={siteAdminOverviewComponents}
        userAreaRoutes={userAreaRoutes}
        userAreaHeaderNavItems={userAreaHeaderNavItems}
        userSettingsSideBarItems={userSettingsSideBarItems}
        userSettingsAreaRoutes={userSettingsAreaRoutes}
        orgAreaRoutes={orgAreaRoutes}
        orgAreaHeaderNavItems={orgAreaHeaderNavItems}
        repoContainerRoutes={repoContainerRoutes}
        repoRevisionContainerRoutes={repoRevisionContainerRoutes}
        repoHeaderActionButtons={repoHeaderActionButtons}
        repoSettingsAreaRoutes={repoSettingsAreaRoutes}
        repoSettingsSidebarGroups={repoSettingsSideBarGroups}
        routes={routes}
        keyboardShortcuts={KEYBOARD_SHORTCUTS}
        showBatchChanges={false}
    />
)
