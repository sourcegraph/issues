import React, { Attributes, PropsWithChildren, PropsWithRef } from 'react'

/**
 * Returns a lazy-loaded reference to a React component in another module.
 *
 * This should be used in URL routes and anywhere else that Webpack code splitting can occur, to
 * avoid all referenced components being in the initial bundle.
 *
 * @param componentFactory Asynchronously imports the component's module; e.g., `() =>
 * import('./MyComponent')`.
 * @param name The export binding name of the component in its module.
 */
export const lazyComponent = <P extends {}, K extends string>(
    componentFactory: () => Promise<{ [k in K]: React.ComponentType<P> }>,
    name: K
): React.FunctionComponent<PropsWithRef<PropsWithChildren<P>> & Attributes> => {
    // Force returning a React.FunctionComponent-like so our result is callable (because it's used
    // in <Route render={...} /> elements where it is expected to be callable).
    const LazyComponent = React.lazy(async () => {
        const component: React.ComponentType<P> = (await componentFactory())[name]
        return { default: component }
    })
    return props => <LazyComponent {...props} />
}
