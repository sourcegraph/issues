import React from 'react'
import { NavLink } from 'react-router-dom'

export const SIDEBAR_CARD_CLASS = 'card mb-3'

export const SIDEBAR_LIST_GROUP_ITEM_ACTION_CLASS = 'list-group-item list-group-item-action py-2'

export const SIDEBAR_BUTTON_CLASS = 'btn btn-secondary d-block w-100 my-2'

/**
 * Item of `SideBarGroupItems`.
 */
export const SidebarNavItem: React.FunctionComponent<{ to: string; exact?: boolean; className?: string }> = ({
    children,
    to,
    exact,
    className = '',
}) => (
    <NavLink to={to} exact={exact} className={`${SIDEBAR_LIST_GROUP_ITEM_ACTION_CLASS} ${className}`}>
        {children}
    </NavLink>
)

/**
 * Header of a `SideBarGroup`
 */
export const SidebarGroupHeader: React.FunctionComponent<{
    icon?: React.ComponentType<{ className?: string }>
    label: string
    children?: undefined
}> = ({ icon: Icon, label }) => (
    <div className="card-header">
        {Icon && <Icon className="icon-inline" />} {label}
    </div>
)

/**
 * A box of items in the side bar. Use `SideBarGroupHeader` and `SideBarGroupItems` as children.
 */
export const SidebarGroup: React.FunctionComponent<{}> = ({ children }) => (
    <div className={SIDEBAR_CARD_CLASS}>{children}</div>
)

/**
 * Container for all `SideBarNavItem` in a `SideBarGroup`.
 */
export const SidebarGroupItems: React.FunctionComponent<{}> = ({ children }) => (
    <div className="list-group list-group-flush">{children}</div>
)
