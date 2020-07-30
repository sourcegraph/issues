import { parseISO } from 'date-fns'
import formatDistance from 'date-fns/formatDistance'
import * as H from 'history'
import SourceCommitIcon from 'mdi-react/SourceCommitIcon'
import * as React from 'react'
import { Link } from 'react-router-dom'
import { RepoLink } from '../../../../shared/src/components/RepoLink'
import { ResultContainer } from '../../../../shared/src/components/ResultContainer'
import * as GQL from '../../../../shared/src/graphql/schema'
import { AbsoluteRepoFilePosition, RepoSpec, toPrettyBlobURL } from '../../../../shared/src/util/url'
import { DecoratedTextLines } from '../../components/DecoratedTextLines'
import { GitReferenceTag } from '../../repo/GitReferenceTag'
import { eventLogger } from '../../tracking/eventLogger'
import { UserAvatar } from '../../user/UserAvatar'

interface Props {
    location: H.Location

    /**
     * The commit search result.
     */
    result: GQL.CommitSearchResult

    /**
     * Called when the search result is selected.
     */
    onSelect: () => void

    /**
     * Whether this diff should be rendered as expanded.
     */
    expanded: boolean

    allExpanded?: boolean
}

const logClick = (): void => {
    eventLogger.log('CommitSearchResultClicked')
}

export const CommitSearchResult: React.FunctionComponent<Props> = (props: Props) => {
    const title: React.ReactChild = (
        <div className="commit-search-result__title">
            <RepoLink
                repoName={props.result.commit.repository.name}
                to={
                    props.result.commit.tree
                        ? props.result.commit.tree.canonicalURL
                        : props.result.commit.repository.url
                }
            />
            <Link
                to={props.result.commit.url}
                className="commit-search-result__title-person"
                onClick={stopPropagationToCollapseOrExpand}
                onMouseDown={logClick}
            >
                <UserAvatar user={props.result.commit.author.person} size={32} className="mr-1 icon-inline" />
                {props.result.commit.author.person.displayName}
            </Link>
            <Link
                to={props.result.commit.url}
                className="commit-search-result__title-message"
                onClick={stopPropagationToCollapseOrExpand}
                onMouseDown={logClick}
            >
                {commitMessageSubject(props.result.commit.message) || '(empty commit message)'}
            </Link>
            <span className="commit-search-result__title-signature">
                {uniqueReferences([...props.result.refs, ...props.result.sourceRefs]).map((reference, index) => (
                    <GitReferenceTag key={index} gitReference={reference} onMouseDown={logClick} />
                ))}
                <code>
                    <Link
                        to={props.result.commit.url}
                        onClick={stopPropagationToCollapseOrExpand}
                        onMouseDown={logClick}
                    >
                        {props.result.commit.abbreviatedOID}
                    </Link>
                </code>{' '}
                <Link to={props.result.commit.url} onClick={stopPropagationToCollapseOrExpand} onMouseDown={logClick}>
                    {formatDistance(parseISO(props.result.commit.author.date), new Date(), {
                        addSuffix: true,
                    })}
                </Link>
            </span>
        </div>
    )

    const expandedChildren: JSX.Element[] = []

    if (props.result.messagePreview) {
        expandedChildren.push(
            <DecoratedTextLines
                key="messagePreview"
                className="file-match__item commit-search-result__item commit-search-result__item-message-preview"
                value={props.result.messagePreview.value.split('\n')}
                highlights={props.result.messagePreview.highlights}
                lineClasses={[{ line: 1, className: 'strong' }]}
                onMouseDown={logClick}
            />
        )
    }

    if (props.result.diffPreview) {
        const commonContext: RepoSpec = {
            repoName: props.result.commit.repository.name,
        }

        interface AbsoluteRepoFilePositionNonReadonly
            extends Pick<AbsoluteRepoFilePosition, Exclude<keyof AbsoluteRepoFilePosition, 'position'>> {
            position: { line: number; character: number }
        }

        // lhsCtx and rhsCtx need the cast because their values at const init time lack
        // the filePath field, which is assigned as we iterate over the lines below.
        const lhsContext = {
            ...commonContext,
            commitID: props.result.commit.oid + '~',
            revision: props.result.commit.oid + '~',
        } as AbsoluteRepoFilePositionNonReadonly
        const rhsContext = {
            ...commonContext,
            commitID: props.result.commit.oid,
            revision: props.result.commit.oid,
        } as AbsoluteRepoFilePositionNonReadonly

        // Omit "index ", "--- file", and "+++ file" lines.
        const lines = props.result.diffPreview.value.split('\n')
        const lineClasses: { line: number; className: string; url?: string }[] = []
        let ignoreUntilAtAt = false
        for (const [index, line] of lines.entries()) {
            if (ignoreUntilAtAt && !line.startsWith('@@')) {
                lineClasses.push({ line: index + 1, className: 'hidden' })
                continue
            }
            ignoreUntilAtAt = false
            if (line.startsWith('diff ')) {
                ignoreUntilAtAt = true

                // Simplify from "diff --git origname newname".
                const [origName, newName] = line.replace(/^diff --git /, '').split(' ')
                let simplerLine: string
                if (origName === newName) {
                    simplerLine = origName
                } else {
                    simplerLine = `${origName} -> ${newName} (renamed)`
                }
                lines[index] = simplerLine

                lhsContext.filePath = origName
                rhsContext.filePath = newName
                lineClasses.push({ line: index + 1, className: 'file-header', url: toPrettyBlobURL(rhsContext) })
            } else if (line.startsWith('@@')) {
                // TODO(sqs): a bit hacky getting the position
                try {
                    const match = line.match(/^@@ -(\d+),.*\+(\d+)/)
                    if (match) {
                        if (match[1]) {
                            const leftLine = parseInt(match[1], 10)
                            lhsContext.position = { line: leftLine - 1, character: 0 }
                        }
                        if (match[2]) {
                            const rightLine = parseInt(match[2], 10)
                            rhsContext.position = { line: rightLine - 1, character: 0 }
                        }
                    }
                } catch (error) {
                    // TODO(sqs)
                    console.error(error)
                }

                lineClasses.push({ line: index + 1, className: 'hunk-header', url: toPrettyBlobURL(rhsContext) })
            } else {
                if (rhsContext.position?.line) {
                    if (!line.startsWith('+')) {
                        lhsContext.position.line++
                    }
                    if (!line.startsWith('-')) {
                        rhsContext.position.line++
                    }
                }

                if (line.startsWith('+')) {
                    lineClasses.push({ line: index + 1, className: 'added', url: toPrettyBlobURL(rhsContext) })
                } else if (line.startsWith('-')) {
                    lineClasses.push({
                        line: index + 1,
                        className: 'deleted',
                        url: toPrettyBlobURL(lhsContext),
                    })
                }
            }
        }

        expandedChildren.push(
            <DecoratedTextLines
                key="diffPreview"
                className="file-match__item commit-search-result__item commit-search-result__item-diff-preview"
                value={lines}
                highlights={props.result.diffPreview.highlights}
                lineClasses={lineClasses}
                onMouseDown={logClick}
            />
        )
    }

    return (
        <ResultContainer
            collapsible={true}
            defaultExpanded={true}
            icon={SourceCommitIcon}
            title={title}
            expandedChildren={expandedChildren}
        />
    )
}

function commitMessageSubject(message: string): string {
    const eol = message.indexOf('\n')
    return eol === -1 ? message : message.slice(0, eol)
}

function stopPropagationToCollapseOrExpand(event: React.MouseEvent): void {
    event.stopPropagation()
}

function uniqueReferences(references: GQL.GitRef[]): GQL.GitRef[] {
    const seenName = new Set<string>()
    const uniq: GQL.GitRef[] = []
    for (const reference of references) {
        if (!seenName.has(reference.name)) {
            uniq.push(reference)
            seenName.add(reference.name)
        }
    }
    return uniq
}
