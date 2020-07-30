import { Hoverifier } from '@sourcegraph/codeintellify'
import * as H from 'history'
import prettyBytes from 'pretty-bytes'
import ChevronDownIcon from 'mdi-react/ChevronDownIcon'
import * as React from 'react'
import { Link } from 'react-router-dom'
import { ActionItemAction } from '../../../../shared/src/actions/ActionItem'
import { HoverMerged } from '../../../../shared/src/api/client/types/hover'
import * as GQL from '../../../../shared/src/graphql/schema'
import { FileSpec, RepoSpec, ResolvedRevisionSpec, RevisionSpec } from '../../../../shared/src/util/url'
import { DiffStat } from './DiffStat'
import { FileDiffHunks } from './FileDiffHunks'
import { ThemeProps } from '../../../../shared/src/theme'
import { ExtensionsControllerProps } from '../../../../shared/src/extensions/controller'
import ChevronRightIcon from 'mdi-react/ChevronRightIcon'
import classNames from 'classnames'
import { FileDiffFields } from '../../graphql-operations'

export interface FileDiffNodeProps extends ThemeProps {
    node: FileDiffFields
    lineNumbers: boolean
    className?: string
    location: H.Location
    history: H.History

    extensionInfo?: {
        /** The base repository and revision. */
        base: RepoSpec & RevisionSpec & ResolvedRevisionSpec & { repoID: GQL.Scalars['ID'] }

        /** The head repository and revision. */
        head: RepoSpec & RevisionSpec & ResolvedRevisionSpec & { repoID: GQL.Scalars['ID'] }

        hoverifier: Hoverifier<RepoSpec & RevisionSpec & FileSpec & ResolvedRevisionSpec, HoverMerged, ActionItemAction>
    } & ExtensionsControllerProps

    /** Reflect selected line in url */
    persistLines?: boolean
}

interface State {
    expanded: boolean
    renderDeleted: boolean
}

/** A file diff. */
export class FileDiffNode extends React.PureComponent<FileDiffNodeProps, State> {
    public state: State = { expanded: true, renderDeleted: false }

    public render(): JSX.Element | null {
        const node = this.props.node

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
            stat = <strong className={classNames(className, 'mr-2 code')}>{prettyBytes(sizeChange)}</strong>
        } else {
            stat = (
                <DiffStat
                    added={node.stat.added}
                    changed={node.stat.changed}
                    deleted={node.stat.deleted}
                    className="file-diff-node__header-stat"
                />
            )
        }

        const anchor = `diff-${node.internalID}`

        return (
            <>
                <a id={anchor} />
                <div className={`file-diff-node card ${this.props.className || ''}`}>
                    <div className="card-header file-diff-node__header">
                        <button type="button" className="btn btn-sm btn-icon mr-2" onClick={this.toggleExpand}>
                            {this.state.expanded ? (
                                <ChevronDownIcon className="icon-inline" />
                            ) : (
                                <ChevronRightIcon className="icon-inline" />
                            )}
                        </button>
                        <div className="file-diff-node__header-path-stat align-items-baseline">
                            {!node.oldPath && <span className="badge badge-success mr-2">Added file</span>}
                            {!node.newPath && <span className="badge badge-danger mr-2">Deleted file</span>}
                            {node.newPath && node.oldPath && node.newPath !== node.oldPath && (
                                <span className="badge badge-merged mr-2">Moved file</span>
                            )}
                            {stat}
                            <Link to={{ ...this.props.location, hash: anchor }} className="file-diff-node__header-path">
                                {path}
                            </Link>
                        </div>
                        <div className="file-diff-node__header-actions">
                            {/* We only have a 'view' component for GitBlobs, but not for `VirtualFile`s. */}
                            {node.mostRelevantFile.__typename === 'GitBlob' && (
                                <Link
                                    to={node.mostRelevantFile.url}
                                    className="btn btn-sm"
                                    data-tooltip="View file at revision"
                                >
                                    View
                                </Link>
                            )}
                        </div>
                    </div>
                    {this.state.expanded &&
                        (node.oldFile?.binary || node.newFile?.binary ? (
                            <div className="text-muted m-2">Binary files can't be rendered.</div>
                        ) : !node.newPath && !this.state.renderDeleted ? (
                            <div className="text-muted m-2">
                                <p className="mb-0">Deleted files aren't rendered by default.</p>
                                <button type="button" className="btn btn-link m-0 p-0" onClick={this.setRenderDeleted}>
                                    Click here to view.
                                </button>
                            </div>
                        ) : (
                            <FileDiffHunks
                                {...this.props}
                                className="file-diff-node__hunks"
                                fileDiffAnchor={anchor}
                                extensionInfo={
                                    this.props.extensionInfo && {
                                        ...this.props.extensionInfo,
                                        base: {
                                            ...this.props.extensionInfo.base,
                                            filePath: node.oldPath,
                                        },
                                        head: {
                                            ...this.props.extensionInfo.head,
                                            filePath: node.newPath,
                                        },
                                    }
                                }
                                hunks={node.hunks}
                                lineNumbers={this.props.lineNumbers}
                            />
                        ))}
                </div>
            </>
        )
    }

    private toggleExpand = (): void => this.setState(previousState => ({ expanded: !previousState.expanded }))

    private setRenderDeleted = (): void => this.setState(() => ({ renderDeleted: true }))
}
