import * as React from 'react'
import { RouteComponentProps } from 'react-router-dom'

import { SidebarGroupItems, SidebarGroupHeader, SidebarGroup, SidebarNavItem } from '../../../components/Sidebar'
import { NavGroupDescriptor } from '../../../util/contributions'

export interface CodeIntelSideBarGroup extends NavGroupDescriptor {}

export type CodeIntelSideBarGroups = readonly CodeIntelSideBarGroup[]

interface Props extends RouteComponentProps<{}> {
    codeIntelSidebarGroups: CodeIntelSideBarGroups
    className?: string
    repo: { url: string }
}

/** Sidebar for code intelligence pages. */
export const CodeIntelSidebar: React.FunctionComponent<Props> = ({
    codeIntelSidebarGroups,
    className,
    repo,
}: Props) => (
    <div className={className}>
        {codeIntelSidebarGroups.map(
            ({ header, items, condition = () => true }, index) =>
                condition({}) && (
                    <SidebarGroup key={index}>
                        {header && <SidebarGroupHeader icon={header.icon} label={header.label} />}
                        <SidebarGroupItems>
                            {items.map(
                                ({ label, to, exact, condition = () => true }) =>
                                    condition({}) && (
                                        <SidebarNavItem
                                            to={`${repo.url}/-/code-intelligence${to}`}
                                            exact={exact}
                                            key={label}
                                        >
                                            {label}
                                        </SidebarNavItem>
                                    )
                            )}
                        </SidebarGroupItems>
                    </SidebarGroup>
                )
        )}
    </div>
)
