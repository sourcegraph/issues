import * as React from 'react'
import { displayRepoName, splitPath } from './RepoFileLink'
import { Link } from './Link'

interface Props {
    repoName: string

    /**
     * The link's destination. If null, the element is text, not a link.
     */
    to: string | null

    className?: string

    onClick?: React.MouseEventHandler<HTMLElement>
}

export const RepoLink: React.FunctionComponent<Props> = ({ repoName: fullRepoName, to, className, onClick }) => {
    const [repoBase, repoName] = splitPath(displayRepoName(fullRepoName))
    const children = (
        <>
            {' '}
            {repoBase ? `${repoBase}/` : null}
            <strong>{repoName}</strong>
        </>
    )
    if (to === null) {
        return children
    }
    return (
        <Link className={className || ''} to={to} onClick={onClick}>
            {children}
        </Link>
    )
}
