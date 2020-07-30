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
import { fetchLsifIndexes as defaultFetchLsifIndexes, deleteLsifIndex, Index } from './backend'
import DeleteIcon from 'mdi-react/DeleteIcon'
import { ErrorLike, isErrorLike } from '../../../../shared/src/util/errors'
import { ErrorAlert } from '../../components/alerts'
import { Subject } from 'rxjs'
import * as H from 'history'

const Header: FunctionComponent<{}> = () => (
    <thead>
        <tr>
            <th>Repository</th>
            <th>Commit</th>
            <th>State</th>
            <th>Last Activity</th>
            <th />
        </tr>
    </thead>
)

export interface IndexNodeProps {
    node: Index
    onDelete: () => void
    history: H.History

    /** Function that returns the current time (for stability in visual tests). */
    now?: () => Date
}

const IndexNode: FunctionComponent<IndexNodeProps> = ({ node, onDelete, history, now }) => {
    const [deletionOrError, setDeletionOrError] = useState<'loading' | 'deleted' | ErrorLike>()

    const deleteIndex = async (): Promise<void> => {
        if (!window.confirm('Are you sure you want to delete this index record?')) {
            return
        }

        setDeletionOrError('loading')

        try {
            await deleteLsifIndex({ id: node.id }).toPromise()
            onDelete()
        } catch (error) {
            setDeletionOrError(error)
        }
    }

    return deletionOrError && isErrorLike(deletionOrError) ? (
        <ErrorAlert prefix="Error deleting LSIF index" error={deletionOrError} history={history} />
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
            <td>
                <Link to={`./indexes/${node.id}`}>
                    {node.state === GQL.LSIFIndexState.PROCESSING ? (
                        <span>Processing</span>
                    ) : node.state === GQL.LSIFIndexState.COMPLETED ? (
                        <span className="text-success">Completed</span>
                    ) : node.state === GQL.LSIFIndexState.ERRORED ? (
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
                        Queued <Timestamp date={node.queuedAt} now={now} noAbout={true} />
                    </span>
                )}
            </td>
            <td>
                <button
                    type="button"
                    className="btn btn-sm btn-danger"
                    onClick={deleteIndex}
                    disabled={deletionOrError === 'loading'}
                    data-tooltip="Delete index"
                >
                    <DeleteIcon className="icon-inline" />
                </button>
            </td>
        </tr>
    )
}

interface Props extends RouteComponentProps<{}> {
    repo?: GQL.Repository
    fetchLsifIndexes?: typeof defaultFetchLsifIndexes

    /** Function that returns the current time (for stability in visual tests). */
    now?: () => Date
}

/**
 * The repository settings code intelligence page.
 */
export const CodeIntelIndexesPage: FunctionComponent<Props> = ({
    repo,
    fetchLsifIndexes = defaultFetchLsifIndexes,
    now,
    ...props
}) => {
    useEffect(() => eventLogger.logViewEvent('CodeIntelIndexes'), [])

    const filters: FilteredConnectionFilter[] = [
        {
            label: 'All',
            id: 'all',
            tooltip: 'Show all uploads',
            args: {},
        },
        {
            label: 'Completed',
            id: 'completed',
            tooltip: 'Show completed indexes only',
            args: { state: GQL.LSIFIndexState.COMPLETED },
        },
        {
            label: 'Errored',
            id: 'errored',
            tooltip: 'Show errored indexes only',
            args: { state: GQL.LSIFIndexState.ERRORED },
        },
        {
            label: 'Queued',
            id: 'queued',
            tooltip: 'Show queued indexes only',
            args: { state: GQL.LSIFIndexState.QUEUED },
        },
    ]

    // This observable emits values after successful deletion of an index and
    // forces the filter connection to refresh.
    const onDeleteSubject = useMemo(() => new Subject<void>(), [])
    const onDeleteCallback = useMemo(() => onDeleteSubject.next.bind(onDeleteSubject), [onDeleteSubject])

    const queryIndexes = useCallback(
        (args: FilteredConnectionQueryArgs) =>
            fetchLsifIndexes(repo ? { repository: repo.id, state: null, ...args } : { state: null, ...args }),
        [repo, fetchLsifIndexes]
    )

    return (
        <div className="code-intel-indexes">
            <PageTitle title="Precise code intelligence auto-index records" />
            <h2>Precise code intelligence auto-index records</h2>
            <p>
                Popular Go repositories are indexed automatically via{' '}
                <a href="https://github.com/sourcegraph/lsif-go" target="_blank" rel="noreferrer noopener">
                    lsif-go
                </a>{' '}
                on{' '}
                <a href="https://sourcegraph.com" target="_blank" rel="noreferrer noopener">
                    Sourcegraph.com
                </a>
                . Enable precise code intelligence for non-Go code by{' '}
                <a
                    href="https://docs.sourcegraph.com/user/code_intelligence/lsif"
                    target="_blank"
                    rel="noreferrer noopener"
                >
                    uploading LSIF data
                </a>
                .
            </p>

            <FilteredConnection<Index, Omit<IndexNodeProps, 'node'>>
                className="mt-3"
                listComponent="table"
                listClassName="table"
                noun="index"
                pluralNoun="indexes"
                headComponent={Header}
                nodeComponent={IndexNode}
                nodeComponentProps={{ onDelete: onDeleteCallback, history: props.history, now }}
                queryConnection={queryIndexes}
                updates={onDeleteSubject}
                history={props.history}
                location={props.location}
                cursorPaging={true}
                filters={filters}
            />
        </div>
    )
}
