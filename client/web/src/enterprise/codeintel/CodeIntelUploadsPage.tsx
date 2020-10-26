import * as GQL from '../../../../shared/src/graphql/schema'
import React, { FunctionComponent, useCallback, useEffect, useState, useMemo } from 'react'
import {
    FilteredConnection,
    FilteredConnectionQueryArguments,
    FilteredConnectionFilter,
} from '../../components/FilteredConnection'
import { Link } from '../../../../shared/src/components/Link'
import { PageTitle } from '../../components/PageTitle'
import { RouteComponentProps } from 'react-router'
import { Timestamp } from '../../components/time/Timestamp'
import { fetchLsifUploads as defaultFetchLsifUploads, deleteLsifUpload, Upload } from './backend'
import DeleteIcon from 'mdi-react/DeleteIcon'
import { ErrorLike, isErrorLike } from '../../../../shared/src/util/errors'
import { ErrorAlert } from '../../components/alerts'
import { Subject } from 'rxjs'
import * as H from 'history'
import { LSIFUploadState } from '../../../../shared/src/graphql-operations'
import { TelemetryProps } from '../../../../shared/src/telemetry/telemetryService'

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

export interface UploadNodeProps {
    node: Upload
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
                    {node.state === LSIFUploadState.UPLOADING ? (
                        <span>Uploading</span>
                    ) : node.state === LSIFUploadState.PROCESSING ? (
                        <span>Processing</span>
                    ) : node.state === LSIFUploadState.COMPLETED ? (
                        <span className="text-success">Completed</span>
                    ) : node.state === LSIFUploadState.ERRORED ? (
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

export interface CodeIntelUploadsPageProps extends RouteComponentProps<{}>, TelemetryProps {
    repo?: GQL.IRepository
    fetchLsifUploads?: typeof defaultFetchLsifUploads

    /** Function that returns the current time (for stability in visual tests). */
    now?: () => Date
}

/**
 * The repository settings code intel uploads page.
 */
export const CodeIntelUploadsPage: FunctionComponent<CodeIntelUploadsPageProps> = ({
    repo,
    fetchLsifUploads = defaultFetchLsifUploads,
    now,
    telemetryService,
    ...props
}) => {
    useEffect(() => telemetryService.logViewEvent('CodeIntelUploads'), [telemetryService])

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
            args: { state: LSIFUploadState.COMPLETED },
        },
        {
            label: 'Errored',
            id: 'errored',
            tooltip: 'Show errored uploads only',
            args: { state: LSIFUploadState.ERRORED },
        },
        {
            label: 'Queued',
            id: 'queued',
            tooltip: 'Show queued uploads only',
            args: { state: LSIFUploadState.QUEUED },
        },
    ]

    // This observable emits values after successful deletion of an upload and
    // forces the filter connection to refresh.
    const onDeleteSubject = useMemo(() => new Subject<void>(), [])
    const onDeleteCallback = useMemo(() => onDeleteSubject.next.bind(onDeleteSubject), [onDeleteSubject])

    const queryUploads = useCallback(
        (args: FilteredConnectionQueryArguments) => fetchLsifUploads({ repository: repo?.id, ...args }),
        [repo?.id, fetchLsifUploads]
    )

    return (
        <div className="code-intel-uploads">
            <PageTitle title="Precise code intelligence uploads" />
            <h2>Precise code intelligence uploads</h2>
            <p>
                Enable precise code intelligence by{' '}
                <a
                    href="https://docs.sourcegraph.com/code_intelligence/explanations/precise_code_intelligence"
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

            <FilteredConnection<Upload, Omit<UploadNodeProps, 'node'>>
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
