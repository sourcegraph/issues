import classNames from 'classnames'
import * as React from 'react'
import { NavLink } from 'react-router-dom'

import { PageHeader } from '@sourcegraph/wildcard'

import { BatchChangesProps } from '../../batches/batches'
import { NavItemWithIconDescriptor } from '../../util/contributions'
import { UserAvatar } from '../UserAvatar'

import { UserAreaRouteContext } from './UserArea'

interface Props extends UserAreaRouteContext {
    navItems: readonly UserAreaHeaderNavItem[]
    className?: string
}

export interface UserAreaHeaderContext extends BatchChangesProps, Pick<Props, 'user'> {
    isSourcegraphDotCom: boolean
}

export interface UserAreaHeaderNavItem extends NavItemWithIconDescriptor<UserAreaHeaderContext> {}

/**
 * Header for the user area.
 */
export const UserAreaHeader: React.FunctionComponent<Props> = ({ url, navItems, className = '', ...props }) => (
    <div className={classNames('user-area-header', className)}>
        <div className="container">
            <PageHeader
                path={[
                    {
                        text: (
                            <span className="align-middle">
                                {props.user.displayName ? (
                                    <>
                                        {props.user.displayName} ({props.user.username})
                                    </>
                                ) : (
                                    props.user.username
                                )}
                            </span>
                        ),
                        icon: () => <UserAvatar className="user-area-header__avatar" user={props.user} />,
                    },
                ]}
                className="mb-3"
            />
            <div className="d-flex align-items-end justify-content-between">
                <ul className="nav nav-tabs w-100">
                    {navItems.map(
                        ({ to, label, exact, icon: Icon, condition = () => true }) =>
                            condition(props) && (
                                <li key={label} className="nav-item">
                                    <NavLink to={url + to} className="nav-link" activeClassName="active" exact={exact}>
                                        <span>
                                            {Icon && <Icon className="icon-inline" />}{' '}
                                            <span className="text-content" data-tab-content={label}>
                                                {label}
                                            </span>
                                        </span>
                                    </NavLink>
                                </li>
                            )
                    )}
                </ul>
            </div>
        </div>
    </div>
)
