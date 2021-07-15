import classNames from 'classnames'
import React from 'react'

interface SummaryContainerProps {
    className?: string
}

export const SummaryContainer: React.FunctionComponent<SummaryContainerProps> = ({ children, className }) => (
    <div className={classNames('filtered-connection__summary-container', className)}>{children}</div>
)
