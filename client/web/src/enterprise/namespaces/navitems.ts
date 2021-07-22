import { BatchChangesIconNamespaceNav } from '../../batches/icons'
import { NamespaceAreaNavItem } from '../../namespaces/NamespaceArea'

export const enterpriseNamespaceAreaHeaderNavItems: readonly NamespaceAreaNavItem[] = [
    {
        to: '/batch-changes',
        label: 'Batch Changes',
        icon: BatchChangesIconNamespaceNav,
        condition: ({ batchChangesEnabled }) => batchChangesEnabled,
    },
]
