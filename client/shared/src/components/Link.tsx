import * as H from 'history'
import React from 'react'

export type LinkProps = { to: string | H.LocationDescriptor<any> } & Pick<
    React.AnchorHTMLAttributes<HTMLAnchorElement>,
    Exclude<keyof React.AnchorHTMLAttributes<HTMLAnchorElement>, 'href'>
>

/**
 * The component used to render a link. All shared code must use this component for links—not <a>, <Link>, etc.
 *
 * Different platforms (web app vs. browser extension) require the use of different link components:
 *
 * The web app uses <RouterLinkOrAnchor>, which uses react-router-dom's <Link> for relative URLs (for page
 * navigation using the HTML history API) and <a> for absolute URLs. The react-router-dom <Link> component only
 * works inside a react-router <BrowserRouter> context, so it wouldn't work in the browser extension.
 *
 * The browser extension uses <a> for everything (because code hosts don't generally use react-router). A
 * react-router-dom <Link> wouldn't work in the browser extension, because there is no <BrowserRouter>.
 *
 * This variable must be set at initialization time by calling {@link setLinkComponent}.
 *
 * The `to` property holds the destination URL (do not use `href`). If <a> is used, the `to` property value is
 * given as the `href` property value on the <a> element.
 *
 * @see setLinkComponent
 */
export let Link: React.ComponentType<LinkProps>

if (process.env.NODE_ENV !== 'production') {
    // Fail with helpful message if setLinkComponent has not been called when the <Link> component is used.
    Link = () => {
        throw new Error('No Link component set. You must call setLinkComponent to set the Link component to use.')
    }
}

/**
 * Sets (globally) the component to use for links. This must be set at initialization time.
 *
 * @see Link
 * @see AnchorLink
 */
export function setLinkComponent(component: typeof Link): void {
    Link = component
}

/**
 * A link component (to be passed to {@link setLinkComponent}) that renders a normal <a>. This
 * should be used everywhere except when a HTML5 history API router is in use (e.g., react-router).
 *
 * @see setLinkComponent
 */
export const AnchorLink: React.FunctionComponent<LinkProps> = ({ to, children, ...props }) => (
    <a href={to && typeof to !== 'string' ? H.createPath(to) : to} {...props}>
        {children}
    </a>
)
