import classNames from 'classnames'
import React, { useMemo } from 'react'
import * as H from 'history'
import { createLinkClickHandler } from './linkClickHandler'

interface Props {
    wrapper?: 'div' | 'span'
    dangerousInnerHTML: string
    history: H.History
    className?: string
    /** A function to attain a reference to the top-level div from a parent component. */
    refFn?: (reference: HTMLElement | null) => void
}

export const Markdown: React.FunctionComponent<Props> = ({
    wrapper: RootComponent = 'div',
    refFn,
    className,
    history,
    dangerousInnerHTML,
}: Props) => {
    // Links in markdown cannot use react-router's <Link>.
    // Prevent hitting the backend (full page reloads) for links that stay inside the app.
    const onClick = useMemo(() => createLinkClickHandler(history), [history])
    return (
        <RootComponent
            onClick={onClick}
            ref={refFn}
            className={classNames(className, 'markdown')}
            dangerouslySetInnerHTML={{ __html: dangerousInnerHTML }}
        />
    )
}
