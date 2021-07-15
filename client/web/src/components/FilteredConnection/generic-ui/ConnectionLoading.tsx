import classNames from 'classnames'
import React from 'react'

import { LoadingSpinner } from '@sourcegraph/react-loading-spinner'

interface ConnectionLoadingProps {
    className?: string
}

export const ConnectionLoading: React.FunctionComponent<ConnectionLoadingProps> = ({ className }) => (
    <span className={classNames('filtered-connection__loader test-filtered-connection__loader', className)}>
        <LoadingSpinner className="icon-inline" />
    </span>
)
