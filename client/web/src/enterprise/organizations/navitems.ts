import { orgAreaHeaderNavItems } from '../../org/area/navitems'
import { OrgAreaHeaderNavItem } from '../../org/area/OrgHeader'
import { enterpriseNamespaceAreaHeaderNavItems } from '../namespaces/navitems'

export const enterpriseOrgAreaHeaderNavItems: readonly OrgAreaHeaderNavItem[] = [
    ...orgAreaHeaderNavItems,
    ...enterpriseNamespaceAreaHeaderNavItems,
]
