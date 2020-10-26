import { userAreaHeaderNavItems } from '../../user/area/navitems'
import { UserAreaHeaderNavItem } from '../../user/area/UserAreaHeader'
import { enterpriseNamespaceAreaHeaderNavItems } from '../namespaces/navitems'

export const enterpriseUserAreaHeaderNavItems: readonly UserAreaHeaderNavItem[] = [
    ...userAreaHeaderNavItems,
    ...enterpriseNamespaceAreaHeaderNavItems,
]
