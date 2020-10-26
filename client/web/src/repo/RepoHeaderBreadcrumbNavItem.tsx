import * as React from 'react'

/**
 * An element in the repo header breadcrumb (e.g., "Foo" in "myrepo > Foo" or "myrepo > myrev > Foo").
 *
 * @example
 * <RepoHeaderContributionPortal
 *     position="nav"
 *     element={
 *         <RepoHeaderBreadcrumbNavItem key="foo">Foo</RepoHeaderBreadcrumbNavItem>
 *     }
 * />
 */
export const RepoHeaderBreadcrumbNavItem: React.FunctionComponent<{ children: React.ReactFragment }> = ({
    children,
}) => <span className="repo-header-breadcrumb-nav-item">{children}</span>
