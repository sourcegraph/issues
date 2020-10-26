import LockIcon from 'mdi-react/LockIcon'
import { ExtensionAreaHeaderNavItem } from '../../../extensions/extension/ExtensionAreaHeader'
import { extensionAreaHeaderNavItems } from '../../../extensions/extension/extensionAreaHeaderNavItems'

export const enterpriseExtensionAreaHeaderNavItems: readonly ExtensionAreaHeaderNavItem[] = [
    ...extensionAreaHeaderNavItems,
    {
        condition: context =>
            !!context.extension.registryExtension && context.extension.registryExtension.viewerCanAdminister,
        to: '/-/manage',
        icon: LockIcon,
        label: 'Manage',
    },
]
