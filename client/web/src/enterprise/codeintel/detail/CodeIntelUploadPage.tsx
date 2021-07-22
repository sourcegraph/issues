import DeleteIcon from 'mdi-react/DeleteIcon'
import InformationOutlineIcon from 'mdi-react/InformationOutlineIcon'
import React, { FunctionComponent, useCallback, useEffect, useMemo, useState } from 'react'
import { Redirect, RouteComponentProps } from 'react-router'
import { timer } from 'rxjs'
import { catchError, concatMap, delay, repeatWhen, takeWhile } from 'rxjs/operators'

import { LoadingSpinner } from '@sourcegraph/react-loading-spinner'
import { LSIFUploadState } from '@sourcegraph/shared/src/graphql-operations'
import { TelemetryProps } from '@sourcegraph/shared/src/telemetry/telemetryService'
import { asError, ErrorLike, isErrorLike } from '@sourcegraph/shared/src/util/errors'
import { useObservable } from '@sourcegraph/shared/src/util/useObservable'
import { Container, PageHeader } from '@sourcegraph/wildcard'

import { ErrorAlert } from '../../../components/alerts'
import { PageTitle } from '../../../components/PageTitle'
import { LsifUploadFields } from '../../../graphql-operations'
import { CodeIntelStateBanner } from '../shared/CodeIntelStateBanner'

import { deleteLsifUpload, fetchLsifUpload as defaultFetchUpload } from './backend'
import { CodeIntelAssociatedIndex } from './CodeIntelAssociatedIndex'
import { CodeIntelUploadMeta } from './CodeIntelUploadMeta'
import { CodeIntelUploadTimeline } from './CodeIntelUploadTimeline'

export interface CodeIntelUploadPageProps extends RouteComponentProps<{ id: string }>, TelemetryProps {
    fetchLsifUpload?: typeof defaultFetchUpload
    now?: () => Date
}

const REFRESH_INTERVAL_MS = 5000

const classNamesByState = new Map([
    [LSIFUploadState.COMPLETED, 'alert-success'],
    [LSIFUploadState.ERRORED, 'alert-danger'],
])

export const CodeIntelUploadPage: FunctionComponent<CodeIntelUploadPageProps> = ({
    match: {
        params: { id },
    },
    fetchLsifUpload = defaultFetchUpload,
    telemetryService,
    now,
}) => {
    useEffect(() => telemetryService.logViewEvent('CodeIntelUpload'), [telemetryService])

    const [deletionOrError, setDeletionOrError] = useState<'loading' | 'deleted' | ErrorLike>()

    const uploadOrError = useObservable(
        useMemo(
            () =>
                timer(0, REFRESH_INTERVAL_MS, undefined).pipe(
                    concatMap(() =>
                        fetchLsifUpload({ id }).pipe(
                            catchError((error): [ErrorLike] => [asError(error)]),
                            repeatWhen(observable => observable.pipe(delay(REFRESH_INTERVAL_MS)))
                        )
                    ),
                    takeWhile(shouldReload, true)
                ),
            [id, fetchLsifUpload]
        )
    )

    const deleteUpload = useCallback(async (): Promise<void> => {
        if (!uploadOrError || isErrorLike(uploadOrError)) {
            return
        }

        let description = `${uploadOrError.inputCommit.slice(0, 7)}`
        if (uploadOrError.inputRoot) {
            description += ` rooted at ${uploadOrError.inputRoot}`
        }

        if (!window.confirm(`Delete upload for commit ${description}?`)) {
            return
        }

        setDeletionOrError('loading')

        try {
            await deleteLsifUpload({ id }).toPromise()
            setDeletionOrError('deleted')
        } catch (error) {
            setDeletionOrError(error)
        }
    }, [id, uploadOrError])

    return deletionOrError === 'deleted' ? (
        <Redirect to="." />
    ) : isErrorLike(deletionOrError) ? (
        <ErrorAlert prefix="Error deleting LSIF upload" error={deletionOrError} />
    ) : (
        <div className="site-admin-lsif-upload-page w-100">
            <PageTitle title="Code intelligence - uploads" />
            {isErrorLike(uploadOrError) ? (
                <ErrorAlert prefix="Error loading LSIF upload" error={uploadOrError} />
            ) : !uploadOrError ? (
                <LoadingSpinner className="icon-inline" />
            ) : (
                <>
                    <PageHeader
                        headingElement="h2"
                        path={[
                            {
                                text: (
                                    <>
                                        <span className="text-muted">Upload for commit</span>
                                        <span className="ml-2">
                                            {uploadOrError.projectRoot
                                                ? uploadOrError.projectRoot.commit.abbreviatedOID
                                                : uploadOrError.inputCommit.slice(0, 7)}
                                        </span>
                                        <span className="ml-2 text-muted">indexed by</span>
                                        <span className="ml-2">{uploadOrError.inputIndexer}</span>
                                        <span className="ml-2 text-muted">rooted at</span>
                                        <span className="ml-2">
                                            {(uploadOrError.projectRoot
                                                ? uploadOrError.projectRoot.path
                                                : uploadOrError.inputRoot) || '/'}
                                        </span>
                                    </>
                                ),
                            },
                        ]}
                        actions={
                            <CodeIntelDeleteUpload
                                state={uploadOrError.state}
                                deleteUpload={deleteUpload}
                                deletionOrError={deletionOrError}
                            />
                        }
                        className="mb-3"
                    />

                    <Container>
                        <CodeIntelStateBanner
                            state={uploadOrError.state}
                            placeInQueue={uploadOrError.placeInQueue}
                            failure={uploadOrError.failure}
                            typeName="upload"
                            pluralTypeName="uploads"
                            className={classNamesByState.get(uploadOrError.state)}
                        />
                        {uploadOrError.isLatestForRepo && (
                            <div className="mb-3">
                                <InformationOutlineIcon className="icon-inline" /> This upload can answer queries for
                                the tip of the default branch and are targets of cross-repository find reference
                                operations.
                            </div>
                        )}
                        <CodeIntelUploadMeta node={uploadOrError} now={now} />
                        <CodeIntelAssociatedIndex node={uploadOrError} now={now} />

                        <h3>Timeline</h3>
                        <CodeIntelUploadTimeline now={now} upload={uploadOrError} className="mb-3" />
                    </Container>
                </>
            )}
        </div>
    )
}

const terminalStates = new Set([LSIFUploadState.COMPLETED, LSIFUploadState.ERRORED])

function shouldReload(upload: LsifUploadFields | ErrorLike | null | undefined): boolean {
    return !isErrorLike(upload) && !(upload && terminalStates.has(upload.state))
}

interface CodeIntelDeleteUploadProps {
    state: LSIFUploadState
    deleteUpload: () => Promise<void>
    deletionOrError?: 'loading' | 'deleted' | ErrorLike
}

const CodeIntelDeleteUpload: FunctionComponent<CodeIntelDeleteUploadProps> = ({
    state,
    deleteUpload,
    deletionOrError,
}) =>
    state === LSIFUploadState.DELETING ? (
        <></>
    ) : (
        <button
            type="button"
            className="btn btn-outline-danger"
            onClick={deleteUpload}
            disabled={deletionOrError === 'loading'}
            aria-describedby="upload-delete-button-help"
            data-tooltip={
                state === LSIFUploadState.COMPLETED
                    ? 'Deleting this upload will make it unavailable to answer code intelligence queries the next time the repository commit graph is refreshed.'
                    : 'Delete this upload immediately'
            }
        >
            <DeleteIcon className="icon-inline" /> Delete upload
        </button>
    )
