import * as H from 'history'
import React, { useCallback, AnchorHTMLAttributes } from 'react'
import { Key } from 'ts-key-enum'
import { Link } from './Link'
import classNames from 'classnames'
import { noop } from 'lodash'

const isSelectKeyPress = (event: React.KeyboardEvent): boolean =>
    event.key === Key.Enter && !event.ctrlKey && !event.shiftKey && !event.metaKey && !event.altKey

interface Props extends Pick<AnchorHTMLAttributes<never>, 'target' | 'rel'> {
    /** The link destination URL. */
    to?: H.LocationDescriptor

    /**
     * Called when the user clicks or presses enter on this element.
     */
    onSelect?: (event: React.MouseEvent<HTMLElement> | React.KeyboardEvent<HTMLElement>) => void

    /** A tooltip to display when the user hovers or focuses this element. */
    ['data-tooltip']?: string

    /**
     * If given, the element is treated as a toggle with the boolean indicating its state.
     * Applies `aria-pressed`.
     */
    pressed?: boolean

    /**
     * The component's CSS class name
     *
     * @default "nav-link"
     */
    className?: string

    disabled?: boolean

    id?: string
}

/**
 * A component that is displayed in the same way, regardless of whether it's a link (with a
 * destination URL) or a button (with a click handler).
 *
 * It is keyboard accessible: unlike `<Link>` or `<a>`, pressing the enter key triggers it.
 */
export const ButtonLink: React.FunctionComponent<Props> = ({
    className = 'nav-link',
    to,
    target,
    rel,
    disabled,
    pressed,
    'data-tooltip': tooltip,
    onSelect = noop,
    children,
    id,
}) => {
    // We need to set up a keypress listener because <a onclick> doesn't get
    // triggered by enter.
    const onAnchorKeyPress: React.KeyboardEventHandler<HTMLAnchorElement> = useCallback(
        event => {
            if (isSelectKeyPress(event)) {
                onSelect(event)
            }
        },
        [onSelect]
    )

    const commonProps: React.AnchorHTMLAttributes<HTMLAnchorElement> & {
        'data-tooltip': string | undefined
    } = {
        className: classNames(className, disabled && 'disabled'),
        'data-tooltip': tooltip,
        'aria-label': tooltip,
        role: typeof pressed === 'boolean' ? 'button' : undefined,
        'aria-pressed': pressed,
        tabIndex: 0,
        onClick: onSelect,
        onKeyPress: onAnchorKeyPress,
        id,
    }

    const onClickPreventDefault: React.MouseEventHandler<HTMLAnchorElement> = useCallback(
        event => {
            // Prevent default action of reloading page because of empty href
            event.preventDefault()
            onSelect(event)
        },
        [onSelect]
    )

    if (!to) {
        return (
            // Need empty href for styling reasons
            // Use onAuxClick so that middle-clicks are caught.
            <a href="" {...commonProps} onClick={onClickPreventDefault} onAuxClick={onClickPreventDefault}>
                {children}
            </a>
        )
    }

    return (
        <Link {...commonProps} to={to} target={target} rel={rel}>
            {children}
        </Link>
    )
}
