import * as React from 'react'
import { SourcegraphIcon } from './SourcegraphIcon'

export interface SourcegraphIconButtonProps
    extends Pick<JSX.IntrinsicElements['a'], 'href' | 'title' | 'rel' | 'className' | 'onClick' | 'target'> {
    /** CSS class applied to the icon */
    iconClassName?: string
    /** Text label shown next to the button */
    label?: string
    /** aria-label attribute */
    ariaLabel?: string
}

export const SourcegraphIconButton: React.FunctionComponent<SourcegraphIconButtonProps> = ({
    iconClassName,
    label,
    ariaLabel,
    className,
    href,
    onClick,
    rel,
    target,
    title,
}) => (
    <a
        href={href}
        className={className}
        target={target ?? '_blank'}
        rel={rel ?? 'noopener noreferrer'}
        title={title}
        aria-label={ariaLabel}
        onClick={onClick}
    >
        <SourcegraphIcon className={iconClassName} /> {label}
    </a>
)
