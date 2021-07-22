import React from 'react'
import { Link } from 'react-router-dom'
import { ListGroup, ListGroupItem } from 'reactstrap'

import { BatchChangesProps } from '../batches/batches'
import { SidebarCollapseItems, SidebarGroupItems, SidebarNavItem } from '../components/Sidebar'
import { NavGroupDescriptor } from '../util/contributions'

export interface SiteAdminSideBarGroupContext extends BatchChangesProps {
    isSourcegraphDotCom: boolean
}

export interface SiteAdminSideBarGroup extends NavGroupDescriptor<SiteAdminSideBarGroupContext> {}

export type SiteAdminSideBarGroups = readonly SiteAdminSideBarGroup[]

export interface SiteAdminSidebarProps extends BatchChangesProps {
    isSourcegraphDotCom: boolean
    /** The items for the side bar, by group */
    groups: SiteAdminSideBarGroups
    className: string
}

/**
 * Sidebar for the site admin area.
 */
export const SiteAdminSidebar: React.FunctionComponent<SiteAdminSidebarProps> = ({ className, groups, ...props }) => (
    <div className={`site-admin-sidebar ${className}`}>
        <ListGroup>
            {groups.map(
                ({ header, items, condition = () => true }, index) =>
                    condition(props) &&
                    (items.length > 1 ? (
                        <ListGroupItem className="p-0" key={index}>
                            <SidebarCollapseItems icon={header?.icon} label={header?.label} openByDefault={true}>
                                <SidebarGroupItems>
                                    {items.map(
                                        ({ label, to, source = 'client', condition = () => true }) =>
                                            condition(props) && (
                                                <SidebarNavItem to={to} exact={true} key={label} source={source}>
                                                    {label}
                                                </SidebarNavItem>
                                            )
                                    )}
                                </SidebarGroupItems>
                            </SidebarCollapseItems>
                        </ListGroupItem>
                    ) : (
                        <ListGroupItem className="p-0" key={items[0].label}>
                            <Link to={items[0].to} className="bg-2 border-0 d-flex list-group-item-action p-2 w-100">
                                <span>
                                    {header?.icon && <header.icon className="sidebar__icon icon-inline mr-1" />}{' '}
                                    {items[0].label}
                                </span>
                            </Link>
                        </ListGroupItem>
                    ))
            )}
        </ListGroup>
    </div>
)
