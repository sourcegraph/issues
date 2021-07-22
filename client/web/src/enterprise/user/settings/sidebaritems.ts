import { userSettingsSideBarItems } from '../../../user/settings/sidebaritems'
import { UserSettingsSidebarItems } from '../../../user/settings/UserSettingsSidebar'
import { SHOW_BUSINESS_FEATURES } from '../../dotcom/productSubscriptions/features'
import { authExp } from '../../site-admin/SiteAdminAuthenticationProvidersPage'

export const enterpriseUserSettingsSideBarItems: UserSettingsSidebarItems = [
    ...userSettingsSideBarItems.slice(0, 2),
    {
        label: 'Subscriptions',
        to: '/subscriptions',
        condition: ({ user }) => SHOW_BUSINESS_FEATURES && user.viewerCanAdminister,
    },
    {
        label: 'External accounts',
        to: '/external-accounts',
        exact: true,
        condition: () => authExp,
    },
    {
        to: '/batch-changes',
        label: 'Batch Changes',
        condition: ({ batchChangesEnabled, user: { viewerCanAdminister } }) =>
            batchChangesEnabled && viewerCanAdminister,
    },
    ...userSettingsSideBarItems.slice(2),
    {
        label: 'Permissions',
        to: '/permissions',
        exact: true,
        condition: ({ authenticatedUser }) => !!authenticatedUser.siteAdmin,
    },
    {
        to: '/event-log',
        label: 'Event log',
        condition: ({ user: { viewerCanAdminister } }) => viewerCanAdminister,
    },
]
