import * as GQL from '../../../../shared/src/graphql/schema'
import React, { FunctionComponent, useCallback, useEffect, useState, useMemo } from 'react'
import { eventLogger } from '../../tracking/eventLogger'
import {
    FilteredConnection,
    FilteredConnectionQueryArgs,
    FilteredConnectionFilter,
} from '../../components/FilteredConnection'
import { Link } from '../../../../shared/src/components/Link'
import { PageTitle } from '../../components/PageTitle'
import { RouteComponentProps } from 'react-router'
import { Timestamp } from '../../components/time/Timestamp'
import { fetchLsifUploads as defaultFetchLsifUploads, deleteLsifUpload } from './backend'
import DeleteIcon from 'mdi-react/DeleteIcon'
import { ErrorLike, isErrorLike } from '../../../../shared/src/util/errors'
import { ErrorAlert } from '../../components/alerts'
import { Subject } from 'rxjs'
import * as H from 'history'
import { LsifUploadConnectionFields } from '../../graphql-operations'

const Header: FunctionComponent<{}> = () => (
    <thead>
        <tr>
            <th>Repository</th>
            <th>Commit</th>
            <th>Indexer</th>
            <th>Root</th>
            <th>State</th>
            <th>Last Activity</th>
            <th />
        </tr>
    </thead>
)

export type GraphQlLsifUploadNode = LsifUploadConnectionFields['nodes'][number]

export interface UploadNodeProps {
    node: GraphQlLsifUploadNode
    onDelete: () => void
    history: H.History

    /** Function that returns the current time (for stability in visual tests). */
    now?: () => Date
}

const UploadNode: FunctionComponent<UploadNodeProps> = ({ node, onDelete, history, now }) => {
    const [deletionOrError, setDeletionOrError] = useState<'loading' | 'deleted' | ErrorLike>()

    const deleteUpload = async (): Promise<void> => {
        if (!window.confirm('Are you sure you want to delete this upload?')) {
            return
        }

        setDeletionOrError('loading')

        try {
            await deleteLsifUpload({ id: node.id }).toPromise()
            onDelete()
        } catch (error) {
            setDeletionOrError(error)
        }
    }

    return deletionOrError && isErrorLike(deletionOrError) ? (
        <ErrorAlert prefix="Error deleting LSIF upload" error={deletionOrError} history={history} />
    ) : (
        <tr>
            <td>
                {node.projectRoot ? (
                    <Link to={node.projectRoot.repository.url}>
                        <code>{node.projectRoot.repository.name}</code>
                    </Link>
                ) : (
                    'unknown'
                )}
            </td>
            <td>
                <code>
                    {node.projectRoot ? (
                        <Link to={node.projectRoot.commit.url}>
                            <code>{node.projectRoot.commit.abbreviatedOID}</code>
                        </Link>
                    ) : (
                        node.inputCommit.slice(0, 7)
                    )}
                </code>
            </td>
            <td>{node.inputIndexer}</td>
            <td>
                {node.projectRoot ? (
                    <Link to={node.projectRoot.url}>
                        <strong>{node.projectRoot.path || '/'}</strong>
                    </Link>
                ) : (
                    node.inputRoot || '/'
                )}
            </td>
            <td>
                <Link to={`./uploads/${node.id}`}>
                    {node.state === GQL.LSIFUploadState.UPLOADING ? (
                        <span>Uploading</span>
                    ) : node.state === GQL.LSIFUploadState.PROCESSING ? (
                        <span>Processing</span>
                    ) : node.state === GQL.LSIFUploadState.COMPLETED ? (
                        <span className="text-success">Completed</span>
                    ) : node.state === GQL.LSIFUploadState.ERRORED ? (
                        <span className="text-danger">Failed to process</span>
                    ) : (
                        <span>Waiting to process (#{node.placeInQueue} in line)</span>
                    )}
                </Link>
            </td>
            <td>
                {node.finishedAt ? (
                    <span>
                        Completed <Timestamp date={node.finishedAt} now={now} noAbout={true} />
                    </span>
                ) : node.startedAt ? (
                    <span>
                        Started <Timestamp date={node.startedAt} now={now} noAbout={true} />
                    </span>
                ) : (
                    <span>
                        Uploaded <Timestamp date={node.uploadedAt} now={now} noAbout={true} />
                    </span>
                )}
            </td>
            <td>
                <button
                    type="button"
                    className="btn btn-sm btn-danger"
                    onClick={deleteUpload}
                    disabled={deletionOrError === 'loading'}
                    data-tooltip="Delete upload"
                >
                    <DeleteIcon className="icon-inline" />
                </button>
            </td>
        </tr>
    )
}

interface Props extends RouteComponentProps<{}> {
    repo?: GQL.Repository
    fetchLsifUploads?: typeof defaultFetchLsifUploads

    /** Function that returns the current time (for stability in visual tests). */
    now?: () => Date
}

/**
 * The repository settings code intel uploads page.
 */
export const CodeIntelUploadsPage: FunctionComponent<Props> = ({
    repo,
    fetchLsifUploads = defaultFetchLsifUploads,
    now,
    ...props
}) => {
    useEffect(() => eventLogger.logViewEvent('CodeIntelUploads'), [])

    const filters: FilteredConnectionFilter[] = [
        {
            label: 'All',
            id: 'all',
            tooltip: 'Show all uploads',
            args: {},
        },
        {
            label: 'Current',
            id: 'current',
            tooltip: 'Show current uploads only',
            args: { isLatestForRepo: true },
        },
        {
            label: 'Completed',
            id: 'completed',
            tooltip: 'Show completed uploads only',
            args: { state: GQL.LSIFUploadState.COMPLETED },
        },
        {
            label: 'Errored',
            id: 'errored',
            tooltip: 'Show errored uploads only',
            args: { state: GQL.LSIFUploadState.ERRORED },
        },
        {
            label: 'Queued',
            id: 'queued',
            tooltip: 'Show queued uploads only',
            args: { state: GQL.LSIFUploadState.QUEUED },
        },
    ]

    // This observable emits values after successful deletion of an upload and
    // forces the filter connection to refresh.
    const onDeleteSubject = useMemo(() => new Subject<void>(), [])
    const onDeleteCallback = useMemo(() => onDeleteSubject.next.bind(onDeleteSubject), [onDeleteSubject])

    const queryUploads = useCallback(
        (args: FilteredConnectionQueryArgs) =>
            fetchLsifUploads(
                repo
                    ? { state: null, isLatestForRepo: null, repository: repo.id, ...args }
                    : { state: null, isLatestForRepo: null, ...args }
            ),
        [repo, fetchLsifUploads]
    )

    return (
        <div className="code-intel-uploads">
            <PageTitle title="Precise code intelligence uploads" />
            <h2>Precise code intelligence uploads</h2>
            <p>
                Enable precise code intelligence by{' '}
                <a
                    href="https://docs.sourcegraph.com/user/code_intelligence/lsif"
                    target="_blank"
                    rel="noreferrer noopener"
                >
                    uploading LSIF data
                </a>
                .
            </p>

            <p>
                Current uploads provide code intelligence for the latest commit on the default branch and are used in
                cross-repository <em>Find References</em> requests. Non-current uploads may still provide code
                intelligence for historic and branch commits.
            </p>

            <FilteredConnection<GraphQlLsifUploadNode, Omit<UploadNodeProps, 'node'>>
                className="mt-3"
                listComponent="table"
                listClassName="table"
                noun="upload"
                pluralNoun="uploads"
                headComponent={Header}
                nodeComponent={UploadNode}
                nodeComponentProps={{ onDelete: onDeleteCallback, history: props.history, now }}
                queryConnection={queryUploads}
                updates={onDeleteSubject}
                history={props.history}
                location={props.location}
                cursorPaging={true}
                filters={filters}
                defaultFilter="current"
            />
        </div>
    )
}
