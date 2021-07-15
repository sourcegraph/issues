import classNames from 'classnames'
import React from 'react'

interface ConnectionContainerProps {
    className?: string
    compact?: boolean
}

export const ConnectionContainer: React.FunctionComponent<ConnectionContainerProps> = ({
    children,
    className,
    compact,
}) => {
    const compactnessClass = `filtered-connection--${compact ? 'compact' : 'noncompact'}`
    return (
        <div className={classNames('filtered-connection test-filtered-connection', compactnessClass, className)}>
            {children}
        </div>
    )
}
