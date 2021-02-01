import React from 'react'
import classNames from 'classnames'

export const Page: React.FunctionComponent<{ className?: string }> = ({ className, children }) => (
    <div className={classNames('container web-content mt-4', className)}>{children}</div>
)
