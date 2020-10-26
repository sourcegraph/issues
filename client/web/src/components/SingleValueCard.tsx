import * as React from 'react'
import { LinkOrSpan } from '../../../shared/src/components/LinkOrSpan'

/**
 * A card that displays a large single value.
 */
export const SingleValueCard: React.FunctionComponent<{
    title: string
    subTitle?: string
    value: string | number
    link?: string
    className?: string
    valueClassName?: string
    valueTooltip?: string
    subText?: string
}> = ({ title, value, subTitle, link, className, valueClassName, valueTooltip, subText }) => (
    <div className={`card single-value-card ${className || ''}`}>
        <div className="card-body text-center">
            <h4 className="card-title mb-0">{title}</h4>
            <small className="card-text">{subTitle || ''}</small>
            <p
                data-tooltip={valueTooltip}
                className={`card-text font-weight-bold text-nowrap single-value-card__value ${valueClassName || ''}`}
            >
                <LinkOrSpan to={link}>{value}</LinkOrSpan>
            </p>
            {subText && <small>{subText}</small>}
        </div>
    </div>
)
