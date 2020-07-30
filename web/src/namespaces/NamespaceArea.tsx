import { ExtensionsControllerProps } from '../../../shared/src/extensions/controller'
import * as GQL from '../../../shared/src/graphql/schema'
import { RouteDescriptor } from '../util/contributions'
import { PatternTypeProps } from '../search'
import { ThemeProps } from '../../../shared/src/theme'
import { OptionalAuthProps } from '../auth'

/**
 * Properties passed to all page components in the namespace area.
 */
export interface NamespaceAreaContext
    extends ExtensionsControllerProps,
        ThemeProps,
        OptionalAuthProps,
        Omit<PatternTypeProps, 'setPatternType'> {
    namespace: Pick<GQL.User | GQL.Org, '__typename' | 'id' | 'url'>
}

export interface NamespaceAreaRoute extends RouteDescriptor<NamespaceAreaContext> {}
