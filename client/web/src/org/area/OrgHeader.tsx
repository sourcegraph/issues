import * as React from 'react'
import { Link, NavLink, RouteComponentProps } from 'react-router-dom'
import { NavItemWithIconDescriptor } from '../../util/contributions'
import { OrgAvatar } from '../OrgAvatar'
import { OrgAreaPageProps } from './OrgArea'

interface Props extends OrgAreaPageProps, RouteComponentProps<{}> {
    isSourcegraphDotCom: boolean
    navItems: readonly OrgAreaHeaderNavItem[]
    className?: string
}

export type OrgAreaHeaderContext = Pick<Props, 'org'> & { isSourcegraphDotCom: boolean }

export interface OrgAreaHeaderNavItem extends NavItemWithIconDescriptor<OrgAreaHeaderContext> {}

/**
 * Header for the organization area.
 */
export const OrgHeader: React.FunctionComponent<Props> = ({
    org,
    navItems,
    match,
    className = '',
    isSourcegraphDotCom,
}) => (
    <div className={`org-header ${className}`}>
        <div className="container">
            {org && (
                <>
                    <h2 className="org-header__title">
                        <OrgAvatar org={org.name} />{' '}
                        <span className="org-header__title-text">{org.displayName || org.name}</span>
                    </h2>
                    <div className="d-flex align-items-end justify-content-between">
                        <ul className="nav nav-tabs border-bottom-0">
                            {navItems.map(
                                ({ to, label, exact, icon: Icon, condition = () => true }) =>
                                    condition({ org, isSourcegraphDotCom }) && (
                                        <li key={label} className="nav-item">
                                            <NavLink
                                                to={match.url + to}
                                                className="nav-link"
                                                activeClassName="active"
                                                exact={exact}
                                            >
                                                {Icon && <Icon className="icon-inline" />} {label}
                                            </NavLink>
                                        </li>
                                    )
                            )}
                        </ul>
                        <div className="flex-1" />
                        {org.viewerPendingInvitation?.respondURL && (
                            <div className="pb-1">
                                <small className="mr-2">Join organization:</small>
                                <Link to={org.viewerPendingInvitation.respondURL} className="btn btn-success btn-sm">
                                    View invitation
                                </Link>
                            </div>
                        )}
                    </div>
                </>
            )}
        </div>
    </div>
)
