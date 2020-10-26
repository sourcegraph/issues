import { RepoSettingsSideBarGroups } from './RepoSettingsSidebar'
import GearIcon from 'mdi-react/GearIcon'

export const settingsGroup = {
    header: { label: 'Settings', icon: GearIcon },
    items: [
        {
            to: '',
            exact: true,
            label: 'Options',
        },
        {
            to: '/index',
            exact: true,
            label: 'Indexing',
        },
        {
            to: '/mirror',
            exact: true,
            label: 'Mirroring',
        },
    ],
}

export const repoSettingsSideBarGroups: RepoSettingsSideBarGroups = [settingsGroup]
