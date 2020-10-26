import React from 'react'
import classNames from 'classnames'

interface Props {
    title: React.ReactNode
    icon: React.ComponentType<{ className?: string }>
    actions?: React.ReactNode
    className?: string
}

export const PageHeader: React.FunctionComponent<Props> = ({ title, icon: Icon, actions, className }) => (
    <div className={classNames('page-header d-flex flex-wrap align-items-center', className)}>
        <h1 className="flex-grow-1">
            <Icon className="icon-inline page-header__icon" /> {title}
        </h1>
        {actions}
    </div>
)
