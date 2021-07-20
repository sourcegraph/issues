import classnames from 'classnames'
import * as H from 'history'
import ChevronDownIcon from 'mdi-react/ChevronDownIcon'
import ChevronRightIcon from 'mdi-react/ChevronRightIcon'
import prettyBytes from 'pretty-bytes'
import React, { useState, useCallback } from 'react'
import { Observable } from 'rxjs'

import { ViewerId } from '@sourcegraph/shared/src/api/viewerTypes'
import { Link } from '@sourcegraph/shared/src/components/Link'
import { ThemeProps } from '@sourcegraph/shared/src/theme'
import { useRedesignToggle } from '@sourcegraph/shared/src/util/useRedesignToggle'

import { FileDiffFields } from '../../graphql-operations'
import { DiffMode } from '../../repo/commit/RepositoryCommitPage'
import { dirname } from '../../util/path'

import { DiffStat, DiffStatSquares } from './DiffStat'
import { ExtensionInfo } from './FileDiffConnection'
import { FileDiffHunks } from './FileDiffHunks'

export interface FileDiffNodeProps extends ThemeProps {
    node: FileDiffFields
    lineNumbers: boolean
    className?: string
    location: H.Location
    history: H.History

    extensionInfo?: ExtensionInfo<{
        observeViewerId?: (uri: string) => Observable<ViewerId | undefined>
    }>

    /** Reflect selected line in url */
    persistLines?: boolean
    diffMode?: DiffMode
}

/** A file diff. */
export const FileDiffNode: React.FunctionComponent<FileDiffNodeProps> = ({
    history,
    isLightTheme,
    lineNumbers,
    location,
    node,
    className,
    extensionInfo,
    persistLines,
    diffMode = 'unified',
}) => {
    const [expanded, setExpanded] = useState<boolean>(true)
    const [renderDeleted, setRenderDeleted] = useState<boolean>(false)
    const [isRedesignEnabled] = useRedesignToggle()

    const toggleExpand = useCallback((): void => {
        setExpanded(!expanded)
    }, [expanded])

    const onClickToViewDeleted = useCallback((): void => {
        setRenderDeleted(true)
    }, [])

    let path: React.ReactFragment
    if (node.newPath && (node.newPath === node.oldPath || !node.oldPath)) {
        path = <span title={node.newPath}>{node.newPath}</span>
    } else if (node.newPath && node.oldPath && node.newPath !== node.oldPath) {
        path = (
            <span title={`${node.oldPath} ⟶ ${node.newPath}`}>
                {node.oldPath} ⟶ {node.newPath}
            </span>
        )
    } else {
        // By process of elimination (that TypeScript is unfortunately unable to infer, except
        // by reorganizing this code in a way that's much more complex to humans), node.oldPath
        // is non-null.
        // eslint-disable-next-line @typescript-eslint/no-non-null-assertion
        path = <span title={node.oldPath!}>{node.oldPath}</span>
    }

    let stat: React.ReactFragment
    // If one of the files was binary, display file size change instead of DiffStat.
    if (node.oldFile?.binary || node.newFile?.binary) {
        const sizeChange = (node.newFile?.byteSize ?? 0) - (node.oldFile?.byteSize ?? 0)
        const className = sizeChange >= 0 ? 'text-success' : 'text-danger'
        stat = <strong className={classnames(className, 'code')}>{prettyBytes(sizeChange)}</strong>
    } else {
        stat = (
            <>
                <DiffStat className="mr-1" {...node.stat} />
                <DiffStatSquares {...node.stat} />
            </>
        )
    }

    const anchor = `diff-${node.internalID}`

    return (
        <>
            {/* The empty <a> tag is to allow users to anchor links to the top of this file diff node */}
            {/* eslint-disable-next-line jsx-a11y/anchor-has-content, jsx-a11y/anchor-is-valid */}
            <a id={anchor} aria-hidden={true} />
            <div className={classnames('file-diff-node test-file-diff-node', { card: !isRedesignEnabled }, className)}>
                <div className={classnames('file-diff-node__header', { 'card-header': !isRedesignEnabled })}>
                    <button type="button" className="btn btn-sm btn-icon mr-2" onClick={toggleExpand}>
                        {expanded ? (
                            <ChevronDownIcon className="icon-inline" />
                        ) : (
                            <ChevronRightIcon className="icon-inline" />
                        )}
                    </button>
                    <div className="file-diff-node__header-path-stat align-items-baseline">
                        {!node.oldPath && <span className="badge badge-success text-uppercase mr-2">Added</span>}
                        {!node.newPath && <span className="badge badge-danger text-uppercase mr-2">Deleted</span>}
                        {node.newPath && node.oldPath && node.newPath !== node.oldPath && (
                            <span className="badge badge-warning text-uppercase mr-2">
                                {dirname(node.newPath) !== dirname(node.oldPath) ? 'Moved' : 'Renamed'}
                            </span>
                        )}
                        {stat}
                        <Link to={{ ...location, hash: anchor }} className="file-diff-node__header-path ml-2">
                            {path}
                        </Link>
                    </div>
                    <div className="file-diff-node__header-actions">
                        {/* We only have a 'view' component for GitBlobs, but not for `VirtualFile`s. */}
                        {node.mostRelevantFile.__typename === 'GitBlob' && (
                            <Link
                                to={node.mostRelevantFile.url}
                                className="btn btn-sm btn-link"
                                data-tooltip="View file at revision"
                            >
                                View
                            </Link>
                        )}
                    </div>
                </div>
                {expanded &&
                    (node.oldFile?.binary || node.newFile?.binary ? (
                        <div className="text-muted m-2">Binary files can't be rendered.</div>
                    ) : !node.newPath && !renderDeleted ? (
                        <div className="text-muted m-2">
                            <p className="mb-0">Deleted files aren't rendered by default.</p>
                            <button type="button" className="btn btn-link m-0 p-0" onClick={onClickToViewDeleted}>
                                Click here to view.
                            </button>
                        </div>
                    ) : (
                        <FileDiffHunks
                            className="file-diff-node__hunks"
                            fileDiffAnchor={anchor}
                            history={history}
                            isLightTheme={isLightTheme}
                            location={location}
                            persistLines={persistLines}
                            extensionInfo={
                                extensionInfo && {
                                    extensionsController: extensionInfo.extensionsController,
                                    observeViewerId: extensionInfo.observeViewerId,
                                    hoverifier: extensionInfo.hoverifier,
                                    base: {
                                        ...extensionInfo.base,
                                        filePath: node.oldPath,
                                    },
                                    head: {
                                        ...extensionInfo.head,
                                        filePath: node.newPath,
                                    },
                                }
                            }
                            hunks={node.hunks}
                            lineNumbers={lineNumbers}
                            diffMode={diffMode}
                        />
                    ))}
            </div>
        </>
    )
}
