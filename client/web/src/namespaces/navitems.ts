import { NavItemWithIconDescriptor } from '../util/contributions'

export const namespaceAreaHeaderNavItems: readonly Pick<
    NavItemWithIconDescriptor,
    Exclude<keyof NavItemWithIconDescriptor, 'condition'>
>[] = []
