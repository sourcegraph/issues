import classNames from 'classnames'
import React from 'react'

interface ConnectionListProps {
    /** list HTML element type. Default is <ul>. */
    as?: 'ul' | 'table' | 'div'

    /** CSS class name for the list element (<ul>, <table>, or <div>). */
    className?: string
}

export const ConnectionList: React.FunctionComponent<ConnectionListProps> = ({
    as: ListComponent = 'ul',
    className,
    children,
}) => (
    <ListComponent className={classNames('filtered-connection__nodes', className)} data-testid="nodes">
        {children}
    </ListComponent>
)
