import ConsoleIcon from 'mdi-react/ConsoleIcon'
import * as React from 'react'
import { Link, NavLink, RouteComponentProps } from 'react-router-dom'
import * as GQL from '../../../../shared/src/graphql/schema'
import {
    SIDEBAR_BUTTON_CLASS,
    SIDEBAR_LIST_GROUP_ITEM_ACTION_CLASS,
    SidebarGroupItems,
    SidebarGroupHeader,
    SidebarGroup,
} from '../../components/Sidebar'
import { NavGroupDescriptor } from '../../util/contributions'

export interface RepoSettingsSideBarGroup extends NavGroupDescriptor {}

export type RepoSettingsSideBarGroups = readonly RepoSettingsSideBarGroup[]

interface Props extends RouteComponentProps<{}> {
    repoSettingsSidebarGroups: RepoSettingsSideBarGroups
    className?: string
    repo?: GQL.IRepository
}

/**
 * Sidebar for repository settings pages.
 */
export const RepoSettingsSidebar: React.FunctionComponent<Props> = ({
    repoSettingsSidebarGroups,
    className,
    repo,
}: Props) =>
    repo ? (
        <div className={`repo-settings-sidebar ${className || ''}`}>
            {repoSettingsSidebarGroups.map(
                ({ header, items, condition = () => true }, index) =>
                    condition({}) && (
                        <SidebarGroup key={index}>
                            {header && <SidebarGroupHeader icon={header.icon} label={header.label} />}
                            <SidebarGroupItems>
                                {items.map(
                                    ({ label, to, exact, condition = () => true }) =>
                                        condition({}) && (
                                            <NavLink
                                                to={`/${repo?.name}/-/settings${to}`}
                                                exact={exact}
                                                key={label}
                                                className={SIDEBAR_LIST_GROUP_ITEM_ACTION_CLASS}
                                            >
                                                {label}
                                            </NavLink>
                                        )
                                )}
                            </SidebarGroupItems>
                        </SidebarGroup>
                    )
            )}

            <Link to="/api/console" className={SIDEBAR_BUTTON_CLASS}>
                <ConsoleIcon className="icon-inline" />
                API console
            </Link>
        </div>
    ) : (
        <div className={`repo-settings-sidebar ${className || ''}`} />
    )
