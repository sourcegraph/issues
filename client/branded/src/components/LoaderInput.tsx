import React from 'react'
import { LoadingSpinner } from '@sourcegraph/react-loading-spinner'
import classNames from 'classnames'

/** Takes loading prop, input component as child */

interface Props {
    loading: boolean
    children: React.ReactNode
    className?: string
}

export const LoaderInput: React.FunctionComponent<Props> = ({ loading, children, className }) => (
    <div className={classNames('loader-input__container', className)}>
        {children}
        {loading && <LoadingSpinner className="loader-input__spinner" />}
    </div>
)
