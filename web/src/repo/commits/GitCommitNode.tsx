import copy from 'copy-to-clipboard'
import ContentCopyIcon from 'mdi-react/ContentCopyIcon'
import DotsHorizontalIcon from 'mdi-react/DotsHorizontalIcon'
import FileDocumentIcon from 'mdi-react/FileDocumentIcon'
import * as React from 'react'
import { Link } from 'react-router-dom'
import { pluralize } from '../../../../shared/src/util/strings'
import { Timestamp } from '../../components/time/Timestamp'
import { Tooltip } from '../../components/tooltip/Tooltip'
import { eventLogger } from '../../tracking/eventLogger'
import { GitCommitNodeByline } from './GitCommitNodeByline'
import { GitCommitFields } from '../../graphql-operations'

export interface GitCommitNodeProps {
    node: GitCommitFields

    /** An optional additional CSS class name to apply to this element. */
    className?: string

    /** Display in a single line (more compactly). */
    compact?: boolean

    /** Expand the commit message body. */
    expandCommitMessageBody?: boolean

    /** Hide the button to expand the commit message body. */
    hideExpandCommitMessageBody?: boolean

    /** Show the full 40-character SHA and parents on their own row. */
    showSHAAndParentsRow?: boolean

    /** Fragment to show at the end to the right of the SHA. */
    afterElement?: React.ReactFragment
}

interface State {
    showCommitMessageBody: boolean
    flashCopiedToClipboardMessage: boolean
}

/** Displays a Git commit. */
export class GitCommitNode extends React.PureComponent<GitCommitNodeProps, State> {
    public state: State = {
        showCommitMessageBody: false,
        flashCopiedToClipboardMessage: false,
    }

    public render(): JSX.Element | null {
        const bylineElement = (
            <GitCommitNodeByline
                className="text-muted git-commit-node__byline"
                author={this.props.node.author}
                committer={this.props.node.committer}
                compact={Boolean(this.props.compact)}
            />
        )
        const messageElement = (
            <div className="git-commit-node__message">
                <Link
                    to={this.props.node.canonicalURL}
                    className="git-commit-node__message-subject"
                    title={this.props.node.message}
                >
                    {this.props.node.subject}
                </Link>
                {this.props.node.body &&
                    !this.props.hideExpandCommitMessageBody &&
                    !this.props.expandCommitMessageBody && (
                        <button
                            type="button"
                            className="btn btn-secondary btn-sm git-commit-node__message-toggle"
                            onClick={this.toggleShowCommitMessageBody}
                        >
                            <DotsHorizontalIcon className="icon-inline" />
                        </button>
                    )}
                {this.props.compact && (
                    <small className="text-muted git-commit-node__message-timestamp">
                        <Timestamp
                            noAbout={true}
                            date={
                                this.props.node.committer ? this.props.node.committer.date : this.props.node.author.date
                            }
                        />
                    </small>
                )}
            </div>
        )
        const oidElement = <code className="git-commit-node__oid">{this.props.node.abbreviatedOID}</code>
        return (
            <div
                key={this.props.node.id}
                className={`git-commit-node ${this.props.compact ? 'git-commit-node--compact' : ''} ${
                    this.props.className || ''
                }`}
            >
                <div className="git-commit-node__row git-commit-node__main">
                    {!this.props.compact ? (
                        <>
                            <div className="git-commit-node__signature">
                                {messageElement}
                                {bylineElement}
                            </div>
                            <div className="git-commit-node__actions">
                                {!this.props.showSHAAndParentsRow && (
                                    <div className="btn-group btn-group-sm mr-2" role="group">
                                        <Link
                                            className="btn btn-secondary"
                                            to={this.props.node.canonicalURL}
                                            data-tooltip="View this commit"
                                        >
                                            <strong>{oidElement}</strong>
                                        </Link>
                                        <button
                                            type="button"
                                            className="btn btn-secondary"
                                            onClick={this.copyToClipboard}
                                            data-tooltip={
                                                this.state.flashCopiedToClipboardMessage ? 'Copied!' : 'Copy full SHA'
                                            }
                                        >
                                            <ContentCopyIcon className="icon-inline small" />
                                        </button>
                                    </div>
                                )}
                                {this.props.node.tree && (
                                    <Link
                                        className="btn btn-secondary btn-sm"
                                        to={this.props.node.tree.canonicalURL}
                                        data-tooltip="View files at this commit"
                                    >
                                        <FileDocumentIcon className="icon-inline small" />
                                    </Link>
                                )}
                            </div>
                        </>
                    ) : (
                        <>
                            {bylineElement}
                            {messageElement}
                            <Link to={this.props.node.canonicalURL}>{oidElement}</Link>
                        </>
                    )}
                    {this.props.afterElement}
                </div>
                {(this.props.expandCommitMessageBody || this.state.showCommitMessageBody) && (
                    <div className="git-commit-node__row">
                        <pre className="git-commit-node__message-body">{this.props.node.body}</pre>
                    </div>
                )}
                {this.props.showSHAAndParentsRow && (
                    <div className="git-commit-node__row git-commit-node__sha-and-parents">
                        <code className="git-ref-tag-2 git-commit-node__sha-and-parents-sha">
                            {this.props.node.oid}{' '}
                            <button
                                type="button"
                                className="btn btn-icon git-commit-node__sha-and-parents-copy"
                                onClick={this.copyToClipboard}
                                data-tooltip={this.state.flashCopiedToClipboardMessage ? 'Copied!' : 'Copy full SHA'}
                            >
                                <ContentCopyIcon className="icon-inline" />
                            </button>
                        </code>
                        <div className="git-commit-node__sha-and-parents-parents">
                            {this.props.node.parents.length > 0 ? (
                                <>
                                    <span className="git-commit-node__sha-and-parents-label">
                                        {this.props.node.parents.length === 1
                                            ? 'Parent'
                                            : `${this.props.node.parents.length} ${pluralize(
                                                  'parent',
                                                  this.props.node.parents.length
                                              )}`}
                                        :
                                    </span>{' '}
                                    {this.props.node.parents.map((parent, index) => (
                                        <Link
                                            key={index}
                                            className="git-ref-tag-2 git-commit-node__sha-and-parents-parent"
                                            to={parent.url}
                                        >
                                            <code>{parent.abbreviatedOID}</code>
                                        </Link>
                                    ))}
                                </>
                            ) : (
                                '(root commit)'
                            )}
                        </div>
                    </div>
                )}
            </div>
        )
    }

    private toggleShowCommitMessageBody = (): void => {
        eventLogger.log('CommitBodyToggled')
        this.setState(previousState => ({ showCommitMessageBody: !previousState.showCommitMessageBody }))
    }

    private copyToClipboard = (): void => {
        eventLogger.log('CommitSHACopiedToClipboard')
        copy(this.props.node.oid)
        this.setState({ flashCopiedToClipboardMessage: true }, () => {
            Tooltip.forceUpdate()
            setTimeout(() => {
                this.setState({ flashCopiedToClipboardMessage: false }, () => Tooltip.forceUpdate())
            }, 1500)
        })
    }
}
