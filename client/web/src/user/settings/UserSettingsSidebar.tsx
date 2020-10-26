import AddIcon from 'mdi-react/AddIcon'
import ConsoleIcon from 'mdi-react/ConsoleIcon'
import LogoutIcon from 'mdi-react/LogoutIcon'
import ServerIcon from 'mdi-react/ServerIcon'
import MapSearchOutlineIcon from 'mdi-react/MapSearchOutlineIcon'
import * as React from 'react'
import { Link, RouteComponentProps } from 'react-router-dom'
import {
    SIDEBAR_BUTTON_CLASS,
    SidebarGroup,
    SidebarGroupHeader,
    SidebarGroupItems,
    SidebarNavItem,
} from '../../components/Sidebar'
import { OrgAvatar } from '../../org/OrgAvatar'
import { SiteAdminAlert } from '../../site-admin/SiteAdminAlert'
import { eventLogger } from '../../tracking/eventLogger'
import { NavItemDescriptor } from '../../util/contributions'
import { UserAreaRouteContext } from '../area/UserArea'
import { HAS_SEEN_TOUR_KEY, HAS_CANCELLED_TOUR_KEY } from '../../search/input/SearchOnboardingTour'
import { OnboardingTourProps } from '../../search'
import { AuthenticatedUser } from '../../auth'
import { UserAreaUserFields } from '../../graphql-operations'

export interface UserSettingsSidebarItemConditionContext {
    user: UserAreaUserFields
    authenticatedUser: Pick<AuthenticatedUser, 'id' | 'siteAdmin' | 'tags'>
}

export type UserSettingsSidebarItems = Record<
    'account',
    readonly NavItemDescriptor<UserSettingsSidebarItemConditionContext>[]
>

export interface UserSettingsSidebarProps extends UserAreaRouteContext, OnboardingTourProps, RouteComponentProps<{}> {
    items: UserSettingsSidebarItems
    className?: string
}

function reEnableSearchTour(): void {
    localStorage.setItem(HAS_SEEN_TOUR_KEY, 'false')
    localStorage.setItem(HAS_CANCELLED_TOUR_KEY, 'false')
}

/** Sidebar for user account pages. */
export const UserSettingsSidebar: React.FunctionComponent<UserSettingsSidebarProps> = props => {
    if (!props.authenticatedUser) {
        return null
    }

    // When the site admin is viewing another user's account.
    const siteAdminViewingOtherUser = props.user.id !== props.authenticatedUser.id
    const context = {
        user: props.user,
        authenticatedUser: props.authenticatedUser,
    }

    return (
        <div className={`user-settings-sidebar ${props.className || ''}`}>
            {/* Indicate when the site admin is viewing another user's account */}
            {siteAdminViewingOtherUser && (
                <SiteAdminAlert className="sidebar__alert">
                    Viewing account for <strong>{props.user.username}</strong>
                </SiteAdminAlert>
            )}

            <SidebarGroup>
                <SidebarGroupHeader label="User account" />
                <SidebarGroupItems>
                    {props.items.account.map(
                        ({ label, to, exact, condition = () => true }) =>
                            condition(context) && (
                                <SidebarNavItem key={label} to={props.match.path + to} exact={exact}>
                                    {label}
                                </SidebarNavItem>
                            )
                    )}
                </SidebarGroupItems>
            </SidebarGroup>

            {(props.user.organizations.nodes.length > 0 || !siteAdminViewingOtherUser) && (
                <SidebarGroup>
                    <SidebarGroupHeader label="Organizations" />
                    <SidebarGroupItems>
                        {props.user.organizations.nodes.map(org => (
                            <SidebarNavItem
                                key={org.id}
                                to={`/organizations/${org.name}/settings`}
                                className="text-truncate text-nowrap"
                            >
                                <OrgAvatar org={org.name} className="d-inline-flex" /> {org.name}
                            </SidebarNavItem>
                        ))}
                    </SidebarGroupItems>
                    {!siteAdminViewingOtherUser && (
                        <Link to="/organizations/new" className="btn btn-secondary btn-sm w-100">
                            <AddIcon className="icon-inline" /> New organization
                        </Link>
                    )}
                </SidebarGroup>
            )}
            {!siteAdminViewingOtherUser && (
                <Link to="/api/console" className={SIDEBAR_BUTTON_CLASS}>
                    <ConsoleIcon className="icon-inline" /> API console
                </Link>
            )}
            {props.authenticatedUser.siteAdmin && (
                <Link to="/site-admin" className={SIDEBAR_BUTTON_CLASS}>
                    <ServerIcon className="icon-inline list-group-item-action-icon" /> Site admin
                </Link>
            )}
            {props.showOnboardingTour && (
                <button type="button" onClick={reEnableSearchTour} className={SIDEBAR_BUTTON_CLASS}>
                    <MapSearchOutlineIcon className="icon-inline list-group-item-action-icon" /> Show search tour
                </button>
            )}
            {!siteAdminViewingOtherUser &&
                props.authenticatedUser.session &&
                props.authenticatedUser.session.canSignOut && (
                    <a href="/-/sign-out" className={SIDEBAR_BUTTON_CLASS} onClick={logTelemetryOnSignOut}>
                        <LogoutIcon className="icon-inline list-group-item-action-icon" /> Sign out
                    </a>
                )}
            <div>Version: {window.context.version}</div>
        </div>
    )
}

function logTelemetryOnSignOut(): void {
    eventLogger.log('SignOutClicked')
}
