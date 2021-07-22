import BrainIcon from 'mdi-react/BrainIcon'
import BriefcaseIcon from 'mdi-react/BriefcaseIcon'
import PuzzleOutlineIcon from 'mdi-react/PuzzleOutlineIcon'

import { BatchChangesIcon } from '../../batches/icons'
import {
    apiConsoleGroup,
    configurationGroup as ossConfigurationGroup,
    maintenanceGroup,
    overviewGroup,
    repositoriesGroup,
    usersGroup,
} from '../../site-admin/sidebaritems'
import { SiteAdminSideBarGroup, SiteAdminSideBarGroups } from '../../site-admin/SiteAdminSidebar'
import { SHOW_BUSINESS_FEATURES } from '../dotcom/productSubscriptions/features'

const configurationGroup: SiteAdminSideBarGroup = {
    ...ossConfigurationGroup,
    items: [
        ...ossConfigurationGroup.items,
        {
            label: 'License',
            to: '/site-admin/license',
        },
    ],
}

const extensionsGroup: SiteAdminSideBarGroup = {
    header: {
        label: 'Extensions',
        icon: PuzzleOutlineIcon,
    },
    items: [
        {
            label: 'Extensions',
            to: '/site-admin/registry/extensions',
        },
    ],
}

export const batchChangesGroup: SiteAdminSideBarGroup = {
    header: {
        label: 'Batch Changes',
        icon: BatchChangesIcon,
    },
    items: [
        {
            label: 'Batch Changes',
            to: '/site-admin/batch-changes',
        },
    ],
    condition: ({ batchChangesEnabled }) => batchChangesEnabled,
}

const businessGroup: SiteAdminSideBarGroup = {
    header: { label: 'Business', icon: BriefcaseIcon },
    items: [
        {
            label: 'Customers',
            to: '/site-admin/dotcom/customers',
            condition: () => SHOW_BUSINESS_FEATURES,
        },
        {
            label: 'Subscriptions',
            to: '/site-admin/dotcom/product/subscriptions',
            condition: () => SHOW_BUSINESS_FEATURES,
        },
        {
            label: 'License key lookup',
            to: '/site-admin/dotcom/product/licenses',
            condition: () => SHOW_BUSINESS_FEATURES,
        },
    ],
    condition: () => SHOW_BUSINESS_FEATURES,
}

const codeIntelGroup: SiteAdminSideBarGroup = {
    header: { label: 'Code intelligence', icon: BrainIcon },
    items: [
        {
            to: '/site-admin/code-intelligence/uploads',
            label: 'Uploads',
        },
        {
            to: '/site-admin/code-intelligence/indexes',
            label: 'Auto indexing',
            condition: () => Boolean(window.context?.codeIntelAutoIndexingEnabled),
        },
    ],
}

export const enterpriseSiteAdminSidebarGroups: SiteAdminSideBarGroups = [
    overviewGroup,
    configurationGroup,
    repositoriesGroup,
    codeIntelGroup,
    usersGroup,
    maintenanceGroup,
    extensionsGroup,
    batchChangesGroup,
    businessGroup,
    apiConsoleGroup,
]
