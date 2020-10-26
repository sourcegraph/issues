import ServerIcon from 'mdi-react/ServerIcon'
import UsersIcon from 'mdi-react/UsersIcon'
import { SiteAdminSideBarGroup, SiteAdminSideBarGroups } from './SiteAdminSidebar'
import SourceRepositoryIcon from 'mdi-react/SourceRepositoryIcon'
import SettingsIcon from 'mdi-react/SettingsIcon'

export const overviewGroup: SiteAdminSideBarGroup = {
    header: {
        label: 'Site admin',
        icon: ServerIcon,
    },
    items: [
        {
            label: 'Overview',
            to: '/site-admin',
            exact: true,
        },
        {
            label: 'Usage stats',
            to: '/site-admin/usage-statistics',
        },
        {
            label: 'Feedback survey',
            to: '/site-admin/surveys',
        },
    ],
}

const configurationGroup: SiteAdminSideBarGroup = {
    header: {
        label: 'Configuration',
        icon: SettingsIcon,
    },
    items: [
        {
            label: 'Site configuration',
            to: '/site-admin/configuration',
        },
        {
            label: 'Global settings',
            to: '/site-admin/global-settings',
        },
    ],
}

export const repositoriesGroup: SiteAdminSideBarGroup = {
    header: {
        label: 'Repositories',
        icon: SourceRepositoryIcon,
    },
    items: [
        {
            label: 'Manage repositories',
            to: '/site-admin/external-services',
        },
        {
            label: 'Repository status',
            to: '/site-admin/repositories',
        },
    ],
}

export const usersGroup: SiteAdminSideBarGroup = {
    header: {
        label: 'Users & auth',
        icon: UsersIcon,
    },
    items: [
        {
            label: 'Users',
            to: '/site-admin/users',
        },
        {
            label: 'Organizations',
            to: '/site-admin/organizations',
        },
        {
            label: 'Access tokens',
            to: '/site-admin/tokens',
        },
    ],
}

export const otherGroup: SiteAdminSideBarGroup = {
    items: [
        {
            label: 'Updates',
            to: '/site-admin/updates',
        },
        {
            label: 'Pings',
            to: '/site-admin/pings',
        },
        {
            label: 'Report a bug',
            to: '/site-admin/report-bug',
        },
    ],
}

export const siteAdminSidebarGroups: SiteAdminSideBarGroups = [
    overviewGroup,
    configurationGroup,
    repositoriesGroup,
    usersGroup,
    otherGroup,
]
