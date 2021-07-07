import Dialog from '@reach/dialog'
import React, { useCallback } from 'react'

import { LoadingSpinner } from '@sourcegraph/react-loading-spinner'
import { useMutation } from '@sourcegraph/shared/src/graphql/graphql'

import { ErrorAlert } from '../../../components/alerts'
import {
    BatchChangesCodeHostFields,
    BatchChangesCredentialFields,
    DeleteBatchChangesCredentialResult,
    DeleteBatchChangesCredentialVariables,
} from '../../../graphql-operations'

import { DELETE_BATCH_CHANGES_CREDENTIAL } from './backend'
import { CodeHostSshPublicKey } from './CodeHostSshPublicKey'
import { ModalHeader } from './ModalHeader'

export interface RemoveCredentialModalProps {
    codeHost: BatchChangesCodeHostFields
    credential: BatchChangesCredentialFields

    onCancel: () => void
    afterDelete: () => void
}

export const RemoveCredentialModal: React.FunctionComponent<RemoveCredentialModalProps> = ({
    codeHost,
    credential,
    onCancel,
    afterDelete,
}) => {
    const labelId = 'removeCredential'
    const [deleteBatchChangesCredential, { loading, error }] = useMutation<
        DeleteBatchChangesCredentialResult,
        DeleteBatchChangesCredentialVariables
    >(DELETE_BATCH_CHANGES_CREDENTIAL, {
        variables: {
            id: credential.id,
        },
        onCompleted: () => afterDelete(),
        update: cache => {
            cache.evict({
                id: cache.identify({
                    __typename: 'BatchChangesCredential',
                    id: credential.id,
                }),
            })
            cache.gc()
        },
    })

    const onDelete = useCallback<React.MouseEventHandler>(() => deleteBatchChangesCredential(), [
        deleteBatchChangesCredential,
    ])

    return (
        <Dialog
            className="modal-body modal-body--top-third p-4 rounded border"
            onDismiss={onCancel}
            aria-labelledby={labelId}
        >
            <div className="test-remove-credential-modal">
                <ModalHeader
                    id={labelId}
                    externalServiceKind={codeHost.externalServiceKind}
                    externalServiceURL={codeHost.externalServiceURL}
                />

                <h3 className="text-danger mb-4">Removing credentials is irreversible</h3>

                {error && <ErrorAlert error={error} />}

                <p>
                    To create changesets on this code host after removing credentials, you will need to repeat the 'Add
                    credentials' process.
                </p>

                {codeHost.requiresSSH && (
                    <CodeHostSshPublicKey
                        externalServiceKind={codeHost.externalServiceKind}
                        sshPublicKey={credential.sshPublicKey!}
                        showInstructionsLink={false}
                        showCopyButton={false}
                        label="Public key to remove"
                    />
                )}

                <div className="d-flex justify-content-end pt-1">
                    <button
                        type="button"
                        disabled={loading}
                        className="btn btn-outline-secondary mr-2"
                        onClick={onCancel}
                    >
                        Cancel
                    </button>
                    <button
                        type="button"
                        disabled={loading}
                        className="btn btn-danger test-remove-credential-modal-submit"
                        onClick={onDelete}
                    >
                        {loading && <LoadingSpinner className="icon-inline" />}
                        Remove credentials
                    </button>
                </div>
            </div>
        </Dialog>
    )
}
