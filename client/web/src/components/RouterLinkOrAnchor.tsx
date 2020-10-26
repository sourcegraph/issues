import * as React from 'react'
import { Link, LinkProps as ReactRouterLinkProps } from 'react-router-dom'
import isAbsoluteUrl from 'is-absolute-url'
import { LinkProps } from '../../../shared/src/components/Link'

/**
 * Uses react-router-dom's <Link> for relative URLs, <a> for absolute URLs. This is useful because passing an
 * absolute URL to <Link> will create an (almost certainly invalid) URL where the absolute URL is resolved to the
 * current URL, such as https://example.com/a/b/https://example.com/c/d.
 */
export const RouterLinkOrAnchor: React.FunctionComponent<ReactRouterLinkProps & LinkProps> = props =>
    typeof props.to === 'string' && isAbsoluteUrl(props.to) ? <a href={props.to} {...props} /> : <Link {...props} />
