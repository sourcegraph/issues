import * as React from 'react'
import classNames from 'classnames'

interface Props {
    className?: string
    icon: React.ReactNode
    title: string
    bodyText: string
    href: string
    linkText: string
    googleAnalytics?: boolean
}

export const CtaBanner: React.FunctionComponent<Props> = ({
    icon,
    className,
    title,
    bodyText,
    href,
    linkText,
    googleAnalytics,
}) => (
    <div className={classNames('web-content cta-banner shadow d-flex flex-row card py-4 pr-4 pl-3', className)}>
        <div className="mr-4 d-flex flex-column align-items-center">{icon}</div>
        <div>
            <h3>{title}</h3>
            <p>{bodyText}</p>
            <a
                href={href}
                // eslint-disable-next-line react/jsx-no-target-blank
                target="_blank"
                rel="noreferrer"
                className={classNames('btn btn-primary', { 'ga-cta-install-now': googleAnalytics })}
            >
                {linkText}
            </a>
        </div>
    </div>
)
