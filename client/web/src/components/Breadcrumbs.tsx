import React, { useState, useEffect, useMemo, useCallback } from 'react'
import ChevronRightIcon from 'mdi-react/ChevronRightIcon'
import { Link } from '../../../shared/src/components/Link'
import { sortBy } from 'lodash'
import { Unsubscribable } from 'sourcegraph'
import { isDefined } from '../../../shared/src/util/types'
import * as H from 'history'
import classNames from 'classnames'

export type Breadcrumb = ElementBreadcrumb | LinkBreadcrumb

interface BaseBreadcrumb {
    /** A unique key for the breadcrumb. */
    key: string

    /** A CSS class name to apply to the container of the breadcrumb element. */
    className?: string

    /**
     * Optionally a custom divider displayed before the element.
     * By default a chevron icon `>` is used.
     */
    divider?: React.ReactNode
}

interface ElementBreadcrumb extends BaseBreadcrumb {
    /** The breadcrumb element being displayed. */
    element: React.ReactNode
}

interface LinkBreadcrumb extends BaseBreadcrumb {
    /**
     * Specification for links. When this breadcrumb is the last breadcrumb and
     * the URL hash is empty, the label is rendered as plain text instead of a link.
     */
    link: { label: string; to: string }
}

/** Type guard to differentiate arbitrary elements and links */
function isElementBreadcrumb(breadcrumb: Breadcrumb): breadcrumb is ElementBreadcrumb {
    return (breadcrumb as ElementBreadcrumb).element !== undefined
}

/**
 * Props of the `Breadcrumbs` component.
 */
export interface BreadcrumbsProps {
    /**
     * All current breadcrumbs.
     */
    breadcrumbs: BreadcrumbAtDepth[]
}

type NullableBreadcrumb = Breadcrumb | null | undefined

/**
 * Exposes APIs for class and function components to register breadcrumbs.
 */
export interface BreadcrumbSetters {
    /**
     * Hook for function components to register a breadcrumb.
     *
     * @param breadcrumb The breadcrumb to register. If a falsy value is passed the breadcrumb will not be included. You can
     * pass an arbitrary element or a link config object for simpler breadcrumbs. **NOTE: The argument MUST be wrapped in `useMemo()`**.
     * @returns Another breadcrumb setters object to pass down to child components to register child breadcrumbs.
     */
    useBreadcrumb: (breadcrumb: NullableBreadcrumb) => BreadcrumbSetters

    /**
     * Imperative method for class components to register a breadcrumb.
     *
     * @param breadcrumb The breadcrumb to register. If a falsy value is passed the breadcrumb will not be included. You can
     * pass an arbitrary element or a link config object for simpler breadcrumbs.
     * @returns Another breadcrumb setters object to pass down to child components to register child breadcrumbs,
     * with a method to remove the breadcrumb again. The object should be added to a [subscription
     * bag](https://about.sourcegraph.com/handbook/engineering/languages/typescript#subscription-bag).
     */
    setBreadcrumb: (breadcrumb: NullableBreadcrumb) => BreadcrumbSetters & Unsubscribable
}

interface BreadcrumbAtDepth {
    /**
     * The position of the breadcrumb in the sequence of breadcrumbs
     */
    depth: number

    /**
     * The breadcrumb to render at the position.
     */
    breadcrumb: NullableBreadcrumb
}

/**
 * Hook to start a breadcrumb tree.
 *
 * @returns An object with `breadcrumbs` to be passed to the `Breadcrumbs` component and a pair of breadcrumb setters
 * to pass down to child components to register child breadcrumbs. Be sure to pass down the returned breadcrumb setters,
 * not the setters that were passed to the component. Otherwise, your breadcrumbs may render out of order.
 *
 */
export const useBreadcrumbs = (): BreadcrumbsProps & BreadcrumbSetters => {
    const [breadcrumbsByDepth, setBreadcrumbsByDepth] = useState<BreadcrumbAtDepth[]>([
        { depth: 0, breadcrumb: { key: 'home', element: <Link to="/search">Home</Link>, divider: null } },
    ])

    /**
     * @param depth The relative depth of the next breadcrumb to be added with the
     * returned breadcrumb setters. This should always be called with $CURRENT_DEPTH + 1.
     */
    const createBreadcrumbSetters = useCallback((depth: number = 1): BreadcrumbSetters => {
        /** Shared logic between plain function and hook */
        function internalSetBreadcrumb(breadcrumb: NullableBreadcrumb): () => void {
            const entry: BreadcrumbAtDepth = { depth, breadcrumb }
            setBreadcrumbsByDepth(breadcrumbs => [...breadcrumbs, entry])
            // cleanup
            return () => {
                setBreadcrumbsByDepth(breadcrumbs => breadcrumbs.filter(breadcrumb => breadcrumb !== entry))
            }
        }

        /** Convenience hook for function components */
        function useBreadcrumb(breadcrumb: NullableBreadcrumb): BreadcrumbSetters {
            useEffect(() => internalSetBreadcrumb(breadcrumb), [breadcrumb])

            return useMemo(() => createBreadcrumbSetters(depth + 1), [])
        }

        /** Plain function for backcompat with class components */
        function setBreadcrumb(breadcrumb: NullableBreadcrumb): BreadcrumbSetters & Unsubscribable {
            const cleanup = internalSetBreadcrumb(breadcrumb)
            const setters = createBreadcrumbSetters(depth + 1)
            return { unsubscribe: cleanup, ...setters }
        }

        return {
            useBreadcrumb,
            setBreadcrumb,
        }
    }, [])

    const breadcrumbSetters = useMemo(() => createBreadcrumbSetters(), [createBreadcrumbSetters])

    return {
        breadcrumbs: breadcrumbsByDepth,
        ...breadcrumbSetters,
    }
}

/**
 * Renders breadcrumbs by depth.
 */
export const Breadcrumbs: React.FC<{ breadcrumbs: BreadcrumbAtDepth[]; location: H.Location }> = ({
    breadcrumbs,
    location,
}) => (
    <nav className="d-flex py-2 container-fluid flex-nowrap flex-shrink-past-contents" aria-label="Breadcrumbs">
        {sortBy(breadcrumbs, 'depth')
            .map(({ breadcrumb }) => breadcrumb)
            .filter(isDefined)
            .map((breadcrumb, index, validBreadcrumbs) => {
                const divider =
                    breadcrumb.divider === undefined ? <ChevronRightIcon className="icon-inline" /> : breadcrumb.divider
                // When the last breadcrumbs is a link and the hash is empty (to allow user to reset hash),
                // render link breadcrumbs as plain text
                return (
                    <span
                        key={breadcrumb.key}
                        className={classNames(
                            'text-muted d-flex align-items-center test-breadcrumb',
                            breadcrumb.className
                        )}
                    >
                        <span className="font-weight-semibold">{divider}</span>
                        {isElementBreadcrumb(breadcrumb) ? (
                            breadcrumb.element
                        ) : index === validBreadcrumbs.length - 1 && !location.hash ? (
                            breadcrumb.link.label
                        ) : (
                            <Link to={breadcrumb.link.to}>{breadcrumb.link.label}</Link>
                        )}
                    </span>
                )
            })}
    </nav>
)

/**
 * To be used in unit tests, it minimally fulfills the BreadcrumbSetters interface.
 */
export const NOOP_BREADCRUMB_SETTERS: BreadcrumbSetters = {
    setBreadcrumb: () => ({ ...NOOP_BREADCRUMB_SETTERS, unsubscribe: () => undefined }),
    useBreadcrumb: () => ({
        ...NOOP_BREADCRUMB_SETTERS,
    }),
}
