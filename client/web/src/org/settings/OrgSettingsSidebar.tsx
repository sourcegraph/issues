import * as React from 'react'
import { NavLink, RouteComponentProps } from 'react-router-dom'
import {
    SIDEBAR_LIST_GROUP_ITEM_ACTION_CLASS,
    SidebarGroup,
    SidebarGroupHeader,
    SidebarGroupItems,
} from '../../components/Sidebar'
import { SiteAdminAlert } from '../../site-admin/SiteAdminAlert'
import { OrgAreaPageProps } from '../area/OrgArea'

interface Props extends OrgAreaPageProps, RouteComponentProps<{}> {
    className?: string
}

/**
 * Sidebar for org settings pages.
 */
export const OrgSettingsSidebar: React.FunctionComponent<Props> = ({ org, authenticatedUser, className, match }) => {
    if (!org) {
        return null
    }

    const siteAdminViewingOtherOrg = authenticatedUser && org.viewerCanAdminister && !org.viewerIsMember

    return (
        <div className={`org-settings-sidebar ${className || ''}`}>
            {/* Indicate when the site admin is viewing another org's settings */}
            {siteAdminViewingOtherOrg && (
                <SiteAdminAlert className="sidebar__alert">
                    Viewing settings for <strong>{org.name}</strong>
                </SiteAdminAlert>
            )}

            <SidebarGroup>
                <SidebarGroupHeader label="Organization" />
                <SidebarGroupItems>
                    <NavLink to={match.url} exact={true} className={SIDEBAR_LIST_GROUP_ITEM_ACTION_CLASS}>
                        Settings
                    </NavLink>
                    <NavLink to={`${match.url}/profile`} exact={true} className={SIDEBAR_LIST_GROUP_ITEM_ACTION_CLASS}>
                        Profile
                    </NavLink>
                </SidebarGroupItems>
            </SidebarGroup>
        </div>
    )
}
