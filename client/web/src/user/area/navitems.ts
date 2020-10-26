import FeatureSearchOutlineIcon from 'mdi-react/FeatureSearchOutlineIcon'
import SettingsIcon from 'mdi-react/SettingsIcon'
import { namespaceAreaHeaderNavItems } from '../../namespaces/navitems'
import { UserAreaHeaderNavItem } from './UserAreaHeader'

export const userAreaHeaderNavItems: readonly UserAreaHeaderNavItem[] = [
    {
        to: '',
        exact: true,
        label: 'Profile',
    },
    {
        to: '/settings',
        label: 'Settings',
        icon: SettingsIcon,
        condition: ({ user: { viewerCanAdminister } }) => viewerCanAdminister,
    },
    {
        to: '/searches',
        label: 'Saved searches',
        icon: FeatureSearchOutlineIcon,
        condition: ({ user: { viewerCanAdminister } }) => viewerCanAdminister,
    },
    ...namespaceAreaHeaderNavItems,
]
