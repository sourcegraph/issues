import Dialog from '@reach/dialog'
import { VisuallyHidden } from '@reach/visually-hidden'
import classnames from 'classnames'
import CloseIcon from 'mdi-react/CloseIcon'
import React from 'react'
import { useHistory } from 'react-router-dom'

import { isErrorLike } from '@sourcegraph/codeintellify/lib/errors'
import { PlatformContextProps } from '@sourcegraph/shared/src/platform/context'

import { ErrorAlert } from '../../../../../../components/alerts'
import { LoaderButton } from '../../../../../../components/LoaderButton'
import { SettingsBasedInsightDashboard } from '../../../../../core/types'

import styles from './DeleteDashobardModal.module.scss'
import { useDeleteDashboardHandler } from './hooks/use-delete-dashboard-handler'

export interface DeleteDashboardModalProps extends PlatformContextProps<'updateSettings'> {
    dashboard: SettingsBasedInsightDashboard
    onClose: () => void
}

export const DeleteDashboardModal: React.FunctionComponent<DeleteDashboardModalProps> = props => {
    const { dashboard, platformContext, onClose } = props
    const history = useHistory()

    const handleDeleteSuccess = (): void => {
        history.push(`/insights/dashboards/${dashboard.owner.id}`)
        onClose()
    }

    const { loadingOrError, handler } = useDeleteDashboardHandler({
        platformContext,
        dashboard,
        onSuccess: handleDeleteSuccess,
    })

    const isDeleting = !isErrorLike(loadingOrError) && loadingOrError

    return (
        <Dialog className={styles.modal} onDismiss={onClose} aria-label="Delete code insight dashboard modal">
            <button type="button" className={classnames('btn btn-icon', styles.closeButton)} onClick={onClose}>
                <VisuallyHidden>Close</VisuallyHidden>
                <CloseIcon />
            </button>

            <h2 className="text-danger">Delete ”{dashboard.title}”</h2>

            <span className="d-block mb-4">
                This can't be undone. You will still be able to access insights from this dashboard in ”All insights”.
            </span>

            {isErrorLike(loadingOrError) && <ErrorAlert className='className="mt-3"' error={loadingOrError} />}

            <div className="d-flex justify-content-end mt-4">
                <button type="button" className="btn btn-secondary mr-2" onClick={onClose}>
                    Cancel
                </button>

                <LoaderButton
                    type="button"
                    alwaysShowLabel={true}
                    loading={isDeleting}
                    label={isDeleting ? 'Deleting' : 'Delete forever'}
                    disabled={isDeleting}
                    spinnerClassName="mr-2"
                    className="btn btn-danger"
                    onClick={handler}
                />
            </div>
        </Dialog>
    )
}
