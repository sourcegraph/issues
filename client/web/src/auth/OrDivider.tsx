import React from 'react'
import classNames from 'classnames'

interface Props {
    className?: string
}

export const OrDivider: React.FunctionComponent<Props> = ({ className }) => (
    <div className={classNames(className, 'd-flex align-items-center')}>
        <div className="w-100 or-divider__border" />
        <small className="px-2 text-muted ">OR</small>
        <div className="w-100 or-divider__border" />
    </div>
)
