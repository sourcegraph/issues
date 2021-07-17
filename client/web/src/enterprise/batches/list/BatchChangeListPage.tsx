import classNames from 'classnames'
import ChevronDownIcon from 'mdi-react/ChevronDownIcon'
import ChevronRightIcon from 'mdi-react/ChevronRightIcon'
import PlusIcon from 'mdi-react/PlusIcon'
import React, { useEffect, useCallback, useState, useMemo } from 'react'
import { RouteComponentProps } from 'react-router'
import { Observable, ReplaySubject } from 'rxjs'
import { filter, map, tap, withLatestFrom } from 'rxjs/operators'

import { Link } from '@sourcegraph/shared/src/components/Link'
import { TelemetryProps } from '@sourcegraph/shared/src/telemetry/telemetryService'
import { pluralize } from '@sourcegraph/shared/src/util/strings'
import { useObservable } from '@sourcegraph/shared/src/util/useObservable'
import { Container, PageHeader } from '@sourcegraph/wildcard'

import { BatchChangesIcon } from '../../../batches/icons'
import { FilteredConnection, FilteredConnectionFilter } from '../../../components/FilteredConnection'
import {
    ListBatchChange,
    Scalars,
    BatchChangeState,
    BatchChangesVariables,
    BatchChangesResult,
    BatchChangesByNamespaceVariables,
} from '../../../graphql-operations'

import {
    areBatchChangesLicensed as _areBatchChangesLicensed,
    queryBatchChanges as _queryBatchChanges,
    queryBatchChangesByNamespace,
} from './backend'
import styles from './BatchChangeListPage.module.scss'
import { BatchChangeNode, BatchChangeNodeProps } from './BatchChangeNode'
import { BatchChangesListEmpty } from './BatchChangesListEmpty'
import { BatchChangesListIntro } from './BatchChangesListIntro'

export interface BatchChangeListPageProps extends TelemetryProps, Pick<RouteComponentProps, 'location'> {
    headingElement: 'h1' | 'h2'
    displayNamespace?: boolean
    /** For testing only. */
    queryBatchChanges?: typeof _queryBatchChanges
    /** For testing only. */
    areBatchChangesLicensed?: typeof _areBatchChangesLicensed
    /** For testing only. */
    openTab?: SelectedTab
}

const FILTERS: FilteredConnectionFilter[] = [
    {
        id: 'status',
        label: 'Status',
        type: 'radio',
        values: [
            {
                label: 'Open',
                value: 'open',
                tooltip: 'Show only batch changes that are open',
                args: { state: BatchChangeState.OPEN },
            },
            {
                label: 'Closed',
                value: 'closed',
                tooltip: 'Show only batch changes that are closed',
                args: { state: BatchChangeState.CLOSED },
            },
            {
                label: 'All',
                value: 'all',
                tooltip: 'Show all batch changes',
                args: {},
            },
        ],
    },
]

type SelectedTab = 'batchChanges' | 'pendingSpecs' | 'gettingStarted'

/**
 * A list of all batch changes on the Sourcegraph instance.
 */
export const BatchChangeListPage: React.FunctionComponent<BatchChangeListPageProps> = ({
    queryBatchChanges = _queryBatchChanges,
    areBatchChangesLicensed = _areBatchChangesLicensed,
    displayNamespace = true,
    headingElement,
    location,
    openTab,
    ...props
}) => {
    useEffect(() => props.telemetryService.logViewEvent('BatchChangesListPage'), [props.telemetryService])

    /*
     * Tracks whether this is the first fetch since this page has been rendered the first time.
     * Used to only switch to the "Getting started" tab if the user didn't select the tab manually.
     */
    const isFirstFetch = useMemo(() => {
        const subject = new ReplaySubject(1)
        subject.next(true)
        return subject
    }, [])
    const [selectedTab, setSelectedTab] = useState<SelectedTab>(openTab ?? 'batchChanges')
    const query = useCallback<(args: Partial<BatchChangesVariables>) => Observable<BatchChangesResult['batchChanges']>>(
        args =>
            queryBatchChanges(args).pipe(
                withLatestFrom(isFirstFetch),
                tap(([response, isFirst]) => {
                    if (isFirst) {
                        isFirstFetch.next(false)
                        if (!openTab && response.totalCount === 0) {
                            setSelectedTab('gettingStarted')
                        }
                    }
                }),
                // Don't emit when we are switching to the getting started tab right away to prevent a costly render.
                // Only if:
                //  - We don't fetch for the first time (the user clicked a tab) OR
                //  - There are more than 0 changesets in the namespace OR
                //  - A test forces us to display a specific tab
                filter(([response, isFirst]) => !isFirst || openTab !== undefined || response.totalCount > 0),
                map(([response]) => response.batchChanges)
            ),
        [queryBatchChanges, isFirstFetch, openTab]
    )
    const licensed: boolean | undefined = useObservable(
        useMemo(() => areBatchChangesLicensed(), [areBatchChangesLicensed])
    )

    return (
        <>
            <PageHeader
                path={[{ icon: BatchChangesIcon, text: 'Batch Changes' }]}
                className="test-batches-list-page mb-3"
                actions={<NewBatchChangeButton location={location} />}
                headingElement={headingElement}
                description="Run custom code over hundreds of repositories and manage the resulting changesets."
            />
            <BatchChangesListIntro licensed={licensed} />
            <BatchChangeListTabHeader selectedTab={selectedTab} setSelectedTab={setSelectedTab} />
            {selectedTab === 'gettingStarted' && <BatchChangesListEmpty />}
            {selectedTab === 'pendingSpecs' && <BatchChangesPending />}
            {selectedTab === 'batchChanges' && (
                <Container className="mb-4">
                    <FilteredConnection<ListBatchChange, Omit<BatchChangeNodeProps, 'node'>>
                        {...props}
                        location={location}
                        nodeComponent={BatchChangeNode}
                        nodeComponentProps={{ displayNamespace }}
                        queryConnection={query}
                        hideSearch={true}
                        defaultFirst={15}
                        filters={FILTERS}
                        noun="batch change"
                        pluralNoun="batch changes"
                        listComponent="div"
                        listClassName={styles.batchChangeListPageGrid}
                        className="filtered-connection__centered-summary"
                        cursorPaging={true}
                        noSummaryIfAllNodesVisible={true}
                        emptyElement={<BatchChangeListEmptyElement location={location} />}
                    />
                </Container>
            )}
        </>
    )
}

export interface NamespaceBatchChangeListPageProps extends BatchChangeListPageProps {
    namespaceID: Scalars['ID']
}

/**
 * A list of all batch changes in a namespace.
 */
export const NamespaceBatchChangeListPage: React.FunctionComponent<NamespaceBatchChangeListPageProps> = ({
    namespaceID,
    ...props
}) => {
    const queryConnection = useCallback(
        (args: Partial<BatchChangesByNamespaceVariables>) =>
            queryBatchChangesByNamespace({
                namespaceID,
                first: args.first ?? null,
                after: args.after ?? null,
                // The types for FilteredConnectionQueryArguments don't allow access to the filter arguments.
                state: (args as { state: BatchChangeState | undefined }).state ?? null,
                viewerCanAdminister: null,
            }),
        [namespaceID]
    )
    return <BatchChangeListPage {...props} displayNamespace={false} queryBatchChanges={queryConnection} />
}

interface BatchChangeListEmptyElementProps extends Pick<RouteComponentProps, 'location'> {}

const BatchChangeListEmptyElement: React.FunctionComponent<BatchChangeListEmptyElementProps> = ({ location }) => (
    <div className="w-100 py-5 text-center">
        <p>
            <strong>No batch changes have been created</strong>
        </p>
        <NewBatchChangeButton location={location} />
    </div>
)

interface NewBatchChangeButtonProps extends Pick<RouteComponentProps, 'location'> {}

const NewBatchChangeButton: React.FunctionComponent<NewBatchChangeButtonProps> = ({ location }) => (
    <Link to={`${location.pathname}/create`} className="btn btn-primary">
        <PlusIcon className="icon-inline" /> Create
    </Link>
)

const BatchChangeListTabHeader: React.FunctionComponent<{
    selectedTab: SelectedTab
    setSelectedTab: (selectedTab: SelectedTab) => void
}> = ({ selectedTab, setSelectedTab }) => {
    const onSelectBatchChanges = useCallback<React.MouseEventHandler>(
        event => {
            event.preventDefault()
            setSelectedTab('batchChanges')
        },
        [setSelectedTab]
    )
    const onSelectPendingSpecs = useCallback<React.MouseEventHandler>(
        event => {
            event.preventDefault()
            setSelectedTab('pendingSpecs')
        },
        [setSelectedTab]
    )
    const onSelectGettingStarted = useCallback<React.MouseEventHandler>(
        event => {
            event.preventDefault()
            setSelectedTab('gettingStarted')
        },
        [setSelectedTab]
    )
    return (
        <div className="overflow-auto mb-2">
            <ul className="nav nav-tabs d-inline-flex d-sm-flex flex-nowrap text-nowrap">
                <li className="nav-item">
                    {/* eslint-disable-next-line jsx-a11y/anchor-is-valid */}
                    <a
                        href=""
                        onClick={onSelectBatchChanges}
                        className={classNames('nav-link', selectedTab === 'batchChanges' && 'active')}
                        role="button"
                    >
                        <span className="text-content" data-tab-content="All batch changes">
                            All batch changes
                        </span>
                    </a>
                </li>
                <li className="nav-item">
                    {/* eslint-disable-next-line jsx-a11y/anchor-is-valid */}
                    <a
                        href=""
                        onClick={onSelectPendingSpecs}
                        className={classNames('nav-link', selectedTab === 'pendingSpecs' && 'active')}
                        role="button"
                    >
                        <span className="text-content" data-tab-content="Executed batch specs">
                            Executed batch specs
                        </span>
                    </a>
                </li>
                <li className="nav-item">
                    {/* eslint-disable-next-line jsx-a11y/anchor-is-valid */}
                    <a
                        href=""
                        onClick={onSelectGettingStarted}
                        className={classNames('nav-link', selectedTab === 'gettingStarted' && 'active')}
                        role="button"
                    >
                        <span className="text-content" data-tab-content="Getting started">
                            Getting started
                        </span>
                    </a>
                </li>
            </ul>
        </div>
    )
}

const BatchChangesPending: React.FunctionComponent = () => (
    <Container className="mb-4">
        <div className={styles.batchChangeJobList}>
            <BatchChangesJob
                state="complete"
                title="old-batch-change"
                when="1 day ago"
                progress={{ complete: 20, failed: 0, executing: 0, queued: 0 }}
            />
            <BatchChangesJob
                state="failed"
                title="errored-batch-change"
                when="1 hour ago"
                progress={{ complete: 12, failed: 3, executing: 0, queued: 0 }}
                failed={[
                    { workspace: 'github.com/sourcegraph/sourcegraph (client/web)', steps: { complete: 3, total: 5 } },
                    {
                        workspace: 'github.com/sourcegraph/sourcegraph (enterprise/batches)',
                        steps: { complete: 3, total: 5 },
                    },
                    { workspace: 'github.com/sourcegraph/src-cli', steps: { complete: 2, total: 5 } },
                ]}
            />{' '}
            <BatchChangesJob
                state="active"
                title="active-batch-change"
                when="10 minutes ago"
                progress={{ complete: 10, failed: 0, executing: 5, queued: 5 }}
            />{' '}
            <BatchChangesJob
                state="active"
                title="pending-batch-change"
                when="1 minute ago"
                progress={{ complete: 0, failed: 0, executing: 0, queued: 10 }}
            />
        </div>
    </Container>
)

interface BatchChangesJobProps {
    state: 'complete' | 'active' | 'failed'
    title: string
    when: string
    progress: {
        complete: number
        failed: number
        executing: number
        queued: number
    }
    failed?: {
        workspace: string
        steps: {
            complete: number
            total: number
        }
    }[]
}

const BatchChangesJob: React.FunctionComponent<BatchChangesJobProps> = ({ state, title, when, progress, failed }) => {
    const total = progress.complete + progress.failed + progress.executing + progress.queued
    const [isExpanded, setIsExpanded] = useState<boolean>(false)
    const toggleIsExpanded = useCallback<React.MouseEventHandler<HTMLButtonElement>>(
        event => {
            event.preventDefault()
            setIsExpanded(!isExpanded)
        },
        [isExpanded]
    )

    return (
        <>
            <div className={styles.batchChangeJobSeparator} />
            <button
                type="button"
                className={classNames(styles.batchChangeJobChevron, 'btn btn-icon d-none d-sm-block')}
                aria-label={isExpanded ? 'Collapse section' : 'Expand section'}
                onClick={toggleIsExpanded}
            >
                {isExpanded ? (
                    <ChevronDownIcon className="icon-inline" aria-label="Close section" />
                ) : (
                    <ChevronRightIcon className="icon-inline" aria-label="Expand section" />
                )}
            </button>{' '}
            <div className={styles.batchChangeJobState}>
                {state === 'complete' && (
                    <span className={classNames('badge badge-success text-uppercase')}>Complete</span>
                )}
                {state === 'active' && <span className={classNames('badge badge-warning text-uppercase')}>Active</span>}
                {state === 'failed' && (
                    <span className={classNames('badge badge-danger text-uppercase')}>Failed</span>
                )}{' '}
            </div>
            <div className={styles.batchChangeJobDetails}>
                <a href="#">
                    <h3 className="m-0">{title}</h3>
                </a>
                <small className="text-muted d-sm-block">submitted {when}</small>
            </div>
            <div className={styles.batchChangeJobProgress}>
                <small
                    className={classNames(styles.batchChangeJobProgressLabel, 'text-muted')}
                >{`${progress.complete}/${total} complete`}</small>
                <div className={styles.batchChangeJobProgressBar}>
                    <div
                        data-tooltip={`${progress.complete} complete`}
                        className={styles.batchChangeJobProgressBarComplete}
                        style={{ flex: progress.complete }}
                    />
                    <div
                        data-tooltip={`${progress.complete} failed`}
                        className={styles.batchChangeJobProgressBarFailed}
                        style={{ flex: progress.failed }}
                    />
                    <div
                        data-tooltip={`${progress.complete} executing`}
                        className={styles.batchChangeJobProgressBarExecuting}
                        style={{ flex: progress.executing }}
                    />
                    <div
                        data-tooltip={`${progress.complete} queued`}
                        className={styles.batchChangeJobProgressBarPending}
                        style={{ flex: progress.pending }}
                    />
                </div>
            </div>
            {isExpanded && failed && failed.length > 0 && (
                <div className={classNames(styles.batchChangeJobFailed, 'p-2')}>
                    <h3>Failed jobs</h3>
                    {failed.map(({ workspace, steps }) => (
                        <>
                            <div className={styles.batchChangeJobFailedWorkspace}>{workspace}</div>
                            <div className={styles.batchChangeJobFailedProgress}>
                                <small
                                    className={classNames(styles.batchChangeJobFailedProgressLabel, 'text-muted')}
                                >{`${steps.complete}/${steps.total} ${pluralize('step', steps.total)}`}</small>
                                <div className={styles.batchChangeJobFailedProgressBar}>
                                    <div
                                        data-tooltip={`${steps.complete} complete`}
                                        className={styles.batchChangeJobFailedProgressBarComplete}
                                        style={{ flex: steps.complete }}
                                    />
                                    <div
                                        data-tooltip={`${steps.total - steps.complete} incomplete`}
                                        className={styles.batchChangeJobFailedProgressBarIncomplete}
                                        style={{ flex: steps.total - steps.complete }}
                                    />
                                </div>
                            </div>
                            <div className={styles.batchChangeJobFailedActions}>
                                <button type="button" className="btn btn-outline-info">
                                    View log
                                </button>
                            </div>
                        </>
                    ))}
                </div>
            )}
        </>
    )
}
