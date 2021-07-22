import DeleteIcon from 'mdi-react/DeleteIcon'
import React, { FunctionComponent, useCallback, useEffect, useMemo, useState } from 'react'
import { Redirect, RouteComponentProps } from 'react-router'
import { timer } from 'rxjs'
import { catchError, concatMap, delay, repeatWhen, takeWhile } from 'rxjs/operators'

import { LoadingSpinner } from '@sourcegraph/react-loading-spinner'
import { LSIFIndexState } from '@sourcegraph/shared/src/graphql-operations'
import { TelemetryProps } from '@sourcegraph/shared/src/telemetry/telemetryService'
import { asError, ErrorLike, isErrorLike } from '@sourcegraph/shared/src/util/errors'
import { useObservable } from '@sourcegraph/shared/src/util/useObservable'
import { Container, PageHeader } from '@sourcegraph/wildcard'

import { ErrorAlert } from '../../../components/alerts'
import { PageTitle } from '../../../components/PageTitle'
import { LsifIndexFields } from '../../../graphql-operations'
import { CodeIntelStateBanner } from '../shared/CodeIntelStateBanner'

import { deleteLsifIndex, fetchLsifIndex as defaultFetchLsifIndex } from './backend'
import { CodeIntelAssociatedUpload } from './CodeIntelAssociatedUpload'
import { CodeIntelIndexMeta } from './CodeIntelIndexMeta'
import { CodeIntelIndexTimeline } from './CodeIntelIndexTimeline'

export interface CodeIntelIndexPageProps extends RouteComponentProps<{ id: string }>, TelemetryProps {
    fetchLsifIndex?: typeof defaultFetchLsifIndex
    now?: () => Date
}

const REFRESH_INTERVAL_MS = 5000

const classNamesByState = new Map([
    [LSIFIndexState.COMPLETED, 'alert-success'],
    [LSIFIndexState.ERRORED, 'alert-danger'],
])

export const CodeIntelIndexPage: FunctionComponent<CodeIntelIndexPageProps> = ({
    match: {
        params: { id },
    },
    fetchLsifIndex = defaultFetchLsifIndex,
    telemetryService,
    now,
}) => {
    useEffect(() => telemetryService.logViewEvent('CodeIntelIndex'), [telemetryService])

    const [deletionOrError, setDeletionOrError] = useState<'loading' | 'deleted' | ErrorLike>()

    const indexOrError = useObservable(
        useMemo(
            () =>
                timer(0, REFRESH_INTERVAL_MS, undefined).pipe(
                    concatMap(() =>
                        fetchLsifIndex({ id }).pipe(
                            catchError((error): [ErrorLike] => [asError(error)]),
                            repeatWhen(observable => observable.pipe(delay(REFRESH_INTERVAL_MS)))
                        )
                    ),
                    takeWhile(shouldReload, true)
                ),
            [id, fetchLsifIndex]
        )
    )

    const deleteIndex = useCallback(async (): Promise<void> => {
        if (!indexOrError || isErrorLike(indexOrError)) {
            return
        }

        if (!window.confirm(`Delete auto-index record for commit ${indexOrError.inputCommit.slice(0, 7)}?`)) {
            return
        }

        setDeletionOrError('loading')

        try {
            await deleteLsifIndex({ id }).toPromise()
            setDeletionOrError('deleted')
        } catch (error) {
            setDeletionOrError(error)
        }
    }, [id, indexOrError])

    return deletionOrError === 'deleted' ? (
        <Redirect to="." />
    ) : isErrorLike(deletionOrError) ? (
        <ErrorAlert prefix="Error deleting LSIF index record" error={deletionOrError} />
    ) : (
        <div className="site-admin-lsif-index-page w-100">
            <PageTitle title="Code intelligence - auto-indexing" />
            {isErrorLike(indexOrError) ? (
                <ErrorAlert prefix="Error loading LSIF index" error={indexOrError} />
            ) : !indexOrError ? (
                <LoadingSpinner className="icon-inline" />
            ) : (
                <>
                    <PageHeader
                        headingElement="h2"
                        path={[
                            {
                                text: (
                                    <>
                                        <span className="text-muted">Auto-index record for commit</span>
                                        <span className="ml-2">
                                            {indexOrError.projectRoot
                                                ? indexOrError.projectRoot.commit.abbreviatedOID
                                                : indexOrError.inputCommit.slice(0, 7)}
                                        </span>
                                    </>
                                ),
                            },
                        ]}
                        actions={<CodeIntelDeleteIndex deleteIndex={deleteIndex} deletionOrError={deletionOrError} />}
                        className="mb-3"
                    />

                    <Container>
                        <CodeIntelStateBanner
                            state={indexOrError.state}
                            placeInQueue={indexOrError.placeInQueue}
                            failure={indexOrError.failure}
                            typeName="index"
                            pluralTypeName="indexes"
                            className={classNamesByState.get(indexOrError.state)}
                        />
                        <CodeIntelIndexMeta node={indexOrError} now={now} />
                        <CodeIntelAssociatedUpload node={indexOrError} now={now} />

                        <h3>Timeline</h3>
                        <CodeIntelIndexTimeline index={indexOrError} now={now} className="mb-3" />
                    </Container>
                </>
            )}
        </div>
    )
}

const terminalStates = new Set([LSIFIndexState.COMPLETED, LSIFIndexState.ERRORED])

function shouldReload(index: LsifIndexFields | ErrorLike | null | undefined): boolean {
    return !isErrorLike(index) && !(index && terminalStates.has(index.state))
}

interface CodeIntelDeleteIndexProps {
    deleteIndex: () => Promise<void>
    deletionOrError?: 'loading' | 'deleted' | ErrorLike
}

const CodeIntelDeleteIndex: FunctionComponent<CodeIntelDeleteIndexProps> = ({ deleteIndex, deletionOrError }) => (
    <button
        type="button"
        className="btn btn-outline-danger"
        onClick={deleteIndex}
        disabled={deletionOrError === 'loading'}
        aria-describedby="upload-delete-button-help"
        data-tooltip="Deleting this index will remove it from the index queue."
    >
        <DeleteIcon className="icon-inline" /> Delete index
    </button>
)
