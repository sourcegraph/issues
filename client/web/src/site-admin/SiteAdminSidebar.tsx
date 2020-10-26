import ConsoleIcon from 'mdi-react/ConsoleIcon'
import * as React from 'react'
import { Link } from 'react-router-dom'
import {
    SIDEBAR_BUTTON_CLASS,
    SidebarGroup,
    SidebarGroupHeader,
    SidebarGroupItems,
    SidebarNavItem,
} from '../components/Sidebar'
import { NavGroupDescriptor } from '../util/contributions'

export interface SiteAdminSideBarGroup extends NavGroupDescriptor {}

export type SiteAdminSideBarGroups = readonly SiteAdminSideBarGroup[]

export interface SiteAdminSidebarProps {
    /** The items for the side bar, by group */
    groups: SiteAdminSideBarGroups
    className: string
}

/**
 * Sidebar for the site admin area.
 */
export const SiteAdminSidebar: React.FunctionComponent<SiteAdminSidebarProps> = ({ className, groups }) => (
    <div className={`site-admin-sidebar ${className}`}>
        {groups.map(
            ({ header, items, condition = () => true }, index) =>
                condition({}) && (
                    <SidebarGroup key={index}>
                        {header && <SidebarGroupHeader icon={header.icon} label={header.label} />}
                        <SidebarGroupItems>
                            {items.map(
                                ({ label, to, exact, condition = () => true }) =>
                                    condition({}) && (
                                        <SidebarNavItem to={to} exact={exact} key={label}>
                                            {label}
                                        </SidebarNavItem>
                                    )
                            )}
                        </SidebarGroupItems>
                    </SidebarGroup>
                )
        )}

        <Link to="/api/console" className={SIDEBAR_BUTTON_CLASS}>
            <ConsoleIcon className="icon-inline" /> API console
        </Link>
        <a href="/-/debug/" className={SIDEBAR_BUTTON_CLASS}>
            Instrumentation
        </a>
        <a href="/-/debug/grafana" className={SIDEBAR_BUTTON_CLASS}>
            Monitoring
        </a>
        <a href="/-/debug/jaeger" className={SIDEBAR_BUTTON_CLASS}>
            Tracing
        </a>
    </div>
)
