import LinkIcon from 'mdi-react/LinkIcon'
import React from 'react'
import { Link } from '../../../shared/src/components/Link'
import { QuickLink } from '../schema/settings.schema'

interface Props {
    quickLinks: QuickLink[] | undefined

    className?: string
}

export const QuickLinks: React.FunctionComponent<Props> = ({ quickLinks, className = '' }) =>
    quickLinks && quickLinks.length > 0 ? (
        <div className={className}>
            {quickLinks.map((quickLink, index) => (
                <small className="quicklink text-nowrap mr-2" key={index}>
                    <Link to={quickLink.url} data-tooltip={quickLink.description}>
                        <LinkIcon className="icon-inline pr-1" />
                        {quickLink.name}
                    </Link>
                </small>
            ))}
        </div>
    ) : null
