import React, { useCallback } from 'react'
import { useHistory } from 'react-router'

import styles from './LinkWithSelectableText.module.scss'

interface LinkWithSelectableTextProps {
    to: string
    className: string
    onClick?: () => void
}

export const LinkWithSelectableText: React.FunctionComponent<LinkWithSelectableTextProps> = ({
    to,
    className,
    children,
    onClick,
}) => {
    const history = useHistory()

    const onMouseUp = useCallback((): void => {
        const selection = window.getSelection()
        if (!selection || selection.toString().length === 0) {
            onClick?.()
            history.push(to)
        }
    }, [to, history, onClick])

    return (
        <div
            onMouseUp={onMouseUp}
            className={className}
            role="link"
            aria-label="Link with selectable text"
            tabIndex={0}
        >
            <div className={styles.cursorText}>{children}</div>
        </div>
    )
}
