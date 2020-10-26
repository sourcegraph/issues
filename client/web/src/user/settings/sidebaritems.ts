import { UserSettingsSidebarItems } from './UserSettingsSidebar'

export const userSettingsSideBarItems: UserSettingsSidebarItems = {
    account: [
        {
            label: 'Settings',
            to: '',
            exact: true,
        },
        {
            label: 'Profile',
            to: '/profile',
            exact: true,
        },
        {
            label: 'Password',
            to: '/password',
            exact: true,
            // Only the builtin auth provider has a password.
            condition: ({ user }) => user.builtinAuth,
        },
        {
            label: 'Emails',
            to: '/emails',
            exact: true,
        },
        {
            label: 'Access tokens',
            to: '/tokens',
            condition: () => window.context.accessTokensAllow !== 'none',
        },
        {
            label: 'Manage repositories',
            to: '/external-services',
            condition: props =>
                window.context.externalServicesUserModeEnabled ||
                (props.user.id === props.authenticatedUser.id &&
                    props.authenticatedUser.tags.includes('AllowUserExternalServicePublic')) ||
                props.user.tags?.includes('AllowUserExternalServicePublic'),
        },
    ],
}
