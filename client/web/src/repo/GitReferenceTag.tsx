import GithubIcon from 'mdi-react/GithubIcon'
import SourceBranchIcon from 'mdi-react/SourceBranchIcon'
import TagIcon from 'mdi-react/TagIcon'
import React from 'react'
import * as GQL from '../../../shared/src/graphql/schema'

interface Props {
    gitReference: GQL.IGitRef

    /**
     * Called when the mousedown event is triggered on the element.
     */
    onMouseDown?: () => void
}

export const GitReferenceTag: React.FunctionComponent<Props> = ({ gitReference, onMouseDown }: Props) => {
    // TODO(sqs): make not github specific
    const githubRepoURL = gitReference.repository.name.startsWith('github.com/')
        ? `https://${gitReference.repository.name}`
        : undefined

    const abbrevName = gitReference.name.slice(gitReference.prefix.length)
    let kind = ''
    let url = githubRepoURL || ''
    let Icon: React.ComponentType<{ className?: string }> | undefined
    switch (gitReference.prefix) {
        case 'refs/heads/':
            kind = 'branch'
            url = url && `${url}/compare/${encodeURIComponent(abbrevName)}`
            Icon = SourceBranchIcon
            break

        case 'refs/tags/':
            kind = 'tag'
            url = url && `${url}/releases/tag/${encodeURIComponent(abbrevName)}`
            Icon = TagIcon
            break

        case 'refs/pull/':
            if (gitReference.name.endsWith('/head')) {
                kind = 'pull request'
                url = url && `${url}/pull/${abbrevName.split('/')[0]}`
                Icon = GithubIcon
            } else if (gitReference.name.endsWith('/merge')) {
                kind = 'pull request merge'
                url = url && `${url}/pull/${abbrevName.split('/')[0]}`
                Icon = GithubIcon
            }
            break
    }

    // Render <a> if there's a canonical URL for the Git ref. Otherwise render <span>.
    const children: (React.ReactChild | undefined)[] = [
        Icon && <Icon key={1} className="icon-inline" />,
        <span key={2} className="git-ref-tag__display-name">
            {gitReference.displayName}
        </span>,
    ]
    const props = {
        title: `${gitReference.name} ${kind ? `(${kind})` : ''}`,
        className: 'git-ref-tag',
    }

    return url ? (
        <a href={url} {...props} onMouseDown={onMouseDown}>
            {children}
        </a>
    ) : (
        <span {...props}>{children}</span>
    )
}
