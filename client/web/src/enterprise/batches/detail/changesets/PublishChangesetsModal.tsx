import Dialog from '@reach/dialog'
import React, { useCallback, useState } from 'react'

import { Form } from '@sourcegraph/branded/src/components/Form'
import { LoadingSpinner } from '@sourcegraph/react-loading-spinner'
import { asError, isErrorLike } from '@sourcegraph/shared/src/util/errors'

import { ErrorAlert } from '../../../../components/alerts'
import { Scalars } from '../../../../graphql-operations'
import { publishChangesets as _publishChangesets } from '../backend'

export interface PublishChangesetsModalProps {
    onCancel: () => void
    afterCreate: () => void
    batchChangeID: Scalars['ID']
    changesetIDs: () => Promise<Scalars['ID'][]>

    /** For testing only. */
    publishChangesets?: typeof _publishChangesets
}

export const PublishChangesetsModal: React.FunctionComponent<PublishChangesetsModalProps> = ({
    onCancel,
    afterCreate,
    batchChangeID,
    changesetIDs,
    publishChangesets: publishChangesets = _publishChangesets,
}) => {
    const [isLoading, setIsLoading] = useState<boolean | Error>(false)
    const [draft, setDraft] = useState(false)

    const onSubmit = useCallback<React.FormEventHandler>(async () => {
        setIsLoading(true)
        try {
            const ids = await changesetIDs()
            await publishChangesets(batchChangeID, ids, draft)
            afterCreate()
        } catch (error) {
            setIsLoading(asError(error))
        }
    }, [changesetIDs, publishChangesets, batchChangeID, draft, afterCreate])

    const onToggleDraft = useCallback<React.ChangeEventHandler<HTMLInputElement>>(event => {
        setDraft(event.target.checked)
    }, [])

    return (
        <Dialog
            className="modal-body modal-body--top-third p-4 rounded border"
            onDismiss={onCancel}
            aria-labelledby={MODAL_LABEL_ID}
        >
            <h3 id={MODAL_LABEL_ID}>Publish changesets</h3>
            <p className="mb-4">Are you sure you want to publish all the selected changesets to the code hosts?</p>
            <Form>
                <div className="form-group">
                    <div className="form-check">
                        <input
                            id={CHECKBOX_ID}
                            type="checkbox"
                            checked={draft}
                            onChange={onToggleDraft}
                            className="form-check-input"
                            disabled={isLoading === true}
                        />
                        <label className="form-check-label" htmlFor={CHECKBOX_ID}>
                            Publish as draft.
                        </label>
                    </div>
                </div>
            </Form>
            {isErrorLike(isLoading) && <ErrorAlert error={isLoading} />}
            <div className="d-flex justify-content-end">
                <button
                    type="button"
                    disabled={isLoading === true}
                    className="btn btn-outline-secondary mr-2"
                    onClick={onCancel}
                >
                    Cancel
                </button>
                <button type="button" onClick={onSubmit} disabled={isLoading === true} className="btn btn-primary">
                    {isLoading === true && <LoadingSpinner className="icon-inline" />}
                    Publish
                </button>
            </div>
        </Dialog>
    )
}

const MODAL_LABEL_ID = 'publish-changesets-modal-title'
const CHECKBOX_ID = 'publish-changesets-modal-draft-check'
