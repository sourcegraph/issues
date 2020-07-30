import * as GQL from '../../../../shared/src/graphql/schema'
import AlertCircleIcon from 'mdi-react/AlertCircleIcon'
import CheckIcon from 'mdi-react/CheckIcon'
import ClockOutlineIcon from 'mdi-react/ClockOutlineIcon'
import React, { FunctionComponent, useEffect, useMemo, useState } from 'react'
import { asError, ErrorLike, isErrorLike } from '../../../../shared/src/util/errors'
import { catchError, takeWhile, concatMap, repeatWhen, delay } from 'rxjs/operators'
import { ErrorAlert } from '../../components/alerts'
import { eventLogger } from '../../tracking/eventLogger'
import { fetchLsifUpload as defaultFetchUpload, deleteLsifUpload, Upload } from './backend'
import { Link } from '../../../../shared/src/components/Link'
import { LoadingSpinner } from '@sourcegraph/react-loading-spinner'
import { PageTitle } from '../../components/PageTitle'
import { RouteComponentProps, Redirect } from 'react-router'
import { Timestamp } from '../../components/time/Timestamp'
import { useObservable } from '../../../../shared/src/util/useObservable'
import DeleteIcon from 'mdi-react/DeleteIcon'
import { SchedulerLike, timer } from 'rxjs'
import * as H from 'history'

const REFRESH_INTERVAL_MS = 5000

interface Props extends RouteComponentProps<{ id: string }> {
    repo?: GQL.Repository
    fetchLsifUpload?: typeof defaultFetchUpload

    /** Scheduler for the refresh timer */
    scheduler?: SchedulerLike
    history: H.History

    /** Function that returns the current time (for stability in visual tests). */
    now?: () => Date
}

const terminalStates = new Set([GQL.LSIFUploadState.COMPLETED, GQL.LSIFUploadState.ERRORED])

function shouldReload(upload: Upload | ErrorLike | null | undefined): boolean {
    return !isErrorLike(upload) && !(upload && terminalStates.has(upload.state))
}

/**
 * A page displaying metadata about an LSIF upload.
 */
export const CodeIntelUploadPage: FunctionComponent<Props> = ({
    repo,
    scheduler,
    match: {
        params: { id },
    },
    history,
    fetchLsifUpload = defaultFetchUpload,
    now,
}) => {
    useEffect(() => eventLogger.logViewEvent('CodeIntelUpload'))

    const [deletionOrError, setDeletionOrError] = useState<'loading' | 'deleted' | ErrorLike>()

    const uploadOrError = useObservable(
        useMemo(
            () =>
                timer(0, REFRESH_INTERVAL_MS, scheduler).pipe(
                    concatMap(() =>
                        fetchLsifUpload({ id }).pipe(
                            catchError((error): [ErrorLike] => [asError(error)]),
                            repeatWhen(observable => observable.pipe(delay(REFRESH_INTERVAL_MS)))
                        )
                    ),
                    takeWhile(shouldReload, true)
                ),
            [id, scheduler, fetchLsifUpload]
        )
    )

    const deleteUpload = async (): Promise<void> => {
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
    }

    return deletionOrError === 'deleted' ? (
        <Redirect to=".." />
    ) : isErrorLike(deletionOrError) ? (
        <ErrorAlert prefix="Error deleting LSIF upload" error={deletionOrError} history={history} />
    ) : (
        <div className="site-admin-lsif-upload-page w-100">
            <PageTitle title="Code intelligence - uploads" />
            {isErrorLike(uploadOrError) ? (
                <ErrorAlert prefix="Error loading LSIF upload" error={uploadOrError} history={history} />
            ) : !uploadOrError ? (
                <LoadingSpinner className="icon-inline" />
            ) : (
                <>
                    <div className="mb-1">
                        <h2 className="mb-0">
                            Upload for commit{' '}
                            {uploadOrError.projectRoot
                                ? uploadOrError.projectRoot.commit.abbreviatedOID
                                : uploadOrError.inputCommit.slice(0, 7)}{' '}
                            indexed by {uploadOrError.inputIndexer} rooted at{' '}
                            {uploadOrError.projectRoot
                                ? uploadOrError.projectRoot.path
                                : uploadOrError.inputRoot || '/'}
                        </h2>
                    </div>

                    {uploadOrError.state === GQL.LSIFUploadState.UPLOADING ? (
                        <div className="alert alert-primary mb-4 mt-3">
                            <LoadingSpinner className="icon-inline" />{' '}
                            <span className="test-upload-state">Still uploading...</span>
                        </div>
                    ) : uploadOrError.state === GQL.LSIFUploadState.PROCESSING ? (
                        <div className="alert alert-primary mb-4 mt-3">
                            <LoadingSpinner className="icon-inline" />{' '}
                            <span className="test-upload-state">Upload is currently being processed...</span>
                        </div>
                    ) : uploadOrError.state === GQL.LSIFUploadState.COMPLETED ? (
                        <div className="alert alert-success mb-4 mt-3">
                            <CheckIcon className="icon-inline" />{' '}
                            <span className="test-upload-state">Upload processed successfully.</span>
                        </div>
                    ) : uploadOrError.state === GQL.LSIFUploadState.ERRORED ? (
                        <div className="alert alert-danger mb-4 mt-3">
                            <AlertCircleIcon className="icon-inline" />{' '}
                            <span className="test-upload-state">Upload failed to complete:</span>{' '}
                            <code>{uploadOrError.failure}</code>
                        </div>
                    ) : (
                        <div className="alert alert-primary mb-4 mt-3">
                            <ClockOutlineIcon className="icon-inline" />{' '}
                            <span className="test-upload-state">
                                Upload is queued. There are {uploadOrError.placeInQueue} uploads ahead of this one.
                            </span>
                        </div>
                    )}

                    <table className="table">
                        <tbody>
                            <tr>
                                <td>Repository</td>
                                <td>
                                    {uploadOrError.projectRoot ? (
                                        <Link to={uploadOrError.projectRoot.repository.url}>
                                            {uploadOrError.projectRoot.repository.name}
                                        </Link>
                                    ) : (
                                        repo?.name || 'unknown'
                                    )}
                                </td>
                            </tr>

                            <tr>
                                <td>Commit</td>
                                <td>
                                    {uploadOrError.projectRoot ? (
                                        <Link to={uploadOrError.projectRoot.commit.url}>
                                            <code>{uploadOrError.projectRoot.commit.oid}</code>
                                        </Link>
                                    ) : (
                                        uploadOrError.inputCommit
                                    )}
                                </td>
                            </tr>

                            <tr>
                                <td>Root</td>
                                <td>
                                    {uploadOrError.projectRoot ? (
                                        <Link to={uploadOrError.projectRoot.url}>
                                            <strong>{uploadOrError.projectRoot.path || '/'}</strong>
                                        </Link>
                                    ) : (
                                        uploadOrError.inputRoot || '/'
                                    )}
                                </td>
                            </tr>

                            <tr>
                                <td>Indexer</td>
                                <td>{uploadOrError.inputIndexer}</td>
                            </tr>

                            <tr>
                                <td>Is latest for repo</td>
                                <td>
                                    {uploadOrError.finishedAt ? (
                                        <span className="test-is-latest-for-repo">
                                            {uploadOrError.isLatestForRepo ? 'yes' : 'no'}
                                        </span>
                                    ) : (
                                        <span className="text-muted">Upload has not yet completed.</span>
                                    )}
                                </td>
                            </tr>

                            <tr>
                                <td>Uploaded</td>
                                <td>
                                    <Timestamp date={uploadOrError.uploadedAt} now={now} noAbout={true} />
                                </td>
                            </tr>

                            <tr>
                                <td>Began processing</td>
                                <td>
                                    {uploadOrError.startedAt ? (
                                        <Timestamp date={uploadOrError.startedAt} now={now} noAbout={true} />
                                    ) : (
                                        <span className="text-muted">Upload has not yet started.</span>
                                    )}
                                </td>
                            </tr>

                            <tr>
                                <td>
                                    {uploadOrError.state === GQL.LSIFUploadState.ERRORED && uploadOrError.finishedAt
                                        ? 'Failed'
                                        : 'Finished'}{' '}
                                    processing
                                </td>
                                <td>
                                    {uploadOrError.finishedAt ? (
                                        <Timestamp date={uploadOrError.finishedAt} now={now} noAbout={true} />
                                    ) : (
                                        <span className="text-muted">Upload has not yet completed.</span>
                                    )}
                                </td>
                            </tr>
                        </tbody>
                    </table>

                    <div className="mt-4 p-2 pt-2">
                        <button
                            type="button"
                            className="btn btn-danger"
                            onClick={deleteUpload}
                            disabled={deletionOrError === 'loading'}
                            aria-describedby="upload-delete-button-help"
                        >
                            <DeleteIcon className="icon-inline" /> Delete upload
                        </button>
                        <small id="upload-delete-button-help" className="form-text text-muted">
                            Deleting this upload makes it immediately unavailable to answer code intelligence queries.
                        </small>
                    </div>
                </>
            )}
        </div>
    )
}
