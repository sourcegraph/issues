import Dialog from '@reach/dialog'
import classNames from 'classnames'
import React, { useCallback, useState } from 'react'

import { Form } from '@sourcegraph/branded/src/components/Form'
import { LoadingSpinner } from '@sourcegraph/react-loading-spinner'
import { useMutation } from '@sourcegraph/shared/src/graphql/graphql'

import { ErrorAlert } from '../../../components/alerts'
import {
    CreateBatchChangesCredentialResult,
    CreateBatchChangesCredentialVariables,
    ExternalServiceKind,
    Scalars,
} from '../../../graphql-operations'

import styles from './AddCredentialModal.module.scss'
import { CREATE_BATCH_CHANGES_CREDENTIAL } from './backend'
import { CodeHostSshPublicKey } from './CodeHostSshPublicKey'
import { ModalHeader } from './ModalHeader'

export interface AddCredentialModalProps {
    onCancel: () => void
    afterCreate: () => void
    userID: Scalars['ID'] | null
    externalServiceKind: ExternalServiceKind
    externalServiceURL: string
    requiresSSH: boolean

    /** For testing only. */
    initialStep?: Step
}

const HELP_TEXT_LINK_URL = 'https://docs.sourcegraph.com/batch_changes/quickstart#configure-code-host-credentials'

const helpTexts: Record<ExternalServiceKind, JSX.Element> = {
    [ExternalServiceKind.GITHUB]: (
        <>
            <a href={HELP_TEXT_LINK_URL} rel="noreferrer noopener" target="_blank">
                Create a new access token
            </a>{' '}
            with <code>repo</code>, <code>read:org</code>, <code>user:email</code>, and <code>read:discussion</code>{' '}
            scopes.
        </>
    ),
    [ExternalServiceKind.GITLAB]: (
        <>
            <a href={HELP_TEXT_LINK_URL} rel="noreferrer noopener" target="_blank">
                Create a new access token
            </a>{' '}
            with <code>api</code>, <code>read_repository</code>, and <code>write_repository</code> scopes.
        </>
    ),
    [ExternalServiceKind.BITBUCKETSERVER]: (
        <>
            <a href={HELP_TEXT_LINK_URL} rel="noreferrer noopener" target="_blank">
                Create a new access token
            </a>{' '}
            with <code>write</code> permissions on the project and repository level.
        </>
    ),

    // These are just for type completeness and serve as placeholders for a bright future.
    [ExternalServiceKind.BITBUCKETCLOUD]: <span>Unsupported</span>,
    [ExternalServiceKind.GITOLITE]: <span>Unsupported</span>,
    [ExternalServiceKind.PERFORCE]: <span>Unsupported</span>,
    [ExternalServiceKind.PHABRICATOR]: <span>Unsupported</span>,
    [ExternalServiceKind.AWSCODECOMMIT]: <span>Unsupported</span>,
    [ExternalServiceKind.OTHER]: <span>Unsupported</span>,
}

type Step = 'add-token' | 'get-ssh-key'

export const AddCredentialModal: React.FunctionComponent<AddCredentialModalProps> = ({
    onCancel,
    afterCreate,
    userID,
    externalServiceKind,
    externalServiceURL,
    requiresSSH,
    initialStep = 'add-token',
}) => {
    const labelId = 'addCredential'
    const [credential, setCredential] = useState<string>('')
    const [step, setStep] = useState<Step>(initialStep)
    const [createBatchChangesCredential, { data, error, loading }] = useMutation<
        CreateBatchChangesCredentialResult,
        CreateBatchChangesCredentialVariables
    >(CREATE_BATCH_CHANGES_CREDENTIAL, {
        onCompleted: ({ createBatchChangesCredential: createdCredential }) => {
            if (requiresSSH && createdCredential.sshPublicKey) {
                setStep('get-ssh-key')
            } else {
                afterCreate()
            }
        },
    })

    const onChangeCredential = useCallback<React.ChangeEventHandler<HTMLInputElement>>(event => {
        setCredential(event.target.value)
    }, [])

    const onSubmit = useCallback<React.FormEventHandler>(
        event => {
            event.preventDefault()
            return createBatchChangesCredential({
                variables: {
                    user: userID,
                    credential,
                    externalServiceKind,
                    externalServiceURL,
                },
            })
        },
        [userID, credential, externalServiceKind, externalServiceURL, createBatchChangesCredential]
    )

    const sshPublicKey = data?.createBatchChangesCredential.sshPublicKey

    return (
        <Dialog
            className="modal-body modal-body--top-third p-4 rounded border"
            onDismiss={onCancel}
            aria-labelledby={labelId}
        >
            <div className="test-add-credential-modal">
                <ModalHeader
                    id={labelId}
                    externalServiceKind={externalServiceKind}
                    externalServiceURL={externalServiceURL}
                />
                {requiresSSH && (
                    <div className="d-flex w-100 justify-content-between mb-4">
                        <div className="flex-grow-1 mr-2">
                            <p className={classNames('mb-0 py-2', step === 'get-ssh-key' && 'text-muted')}>
                                1. Add token
                            </p>
                            <div
                                className={classNames(
                                    styles.addCredentialModalModalStepRuler,
                                    styles.addCredentialModalModalStepRulerPurple
                                )}
                            />
                        </div>
                        <div className="flex-grow-1 ml-2">
                            <p className={classNames('mb-0 py-2', step === 'add-token' && 'text-muted')}>
                                2. Get SSH Key
                            </p>
                            <div
                                className={classNames(
                                    styles.addCredentialModalModalStepRuler,
                                    step === 'add-token' && styles.addCredentialModalModalStepRulerGray,
                                    step === 'get-ssh-key' && styles.addCredentialModalModalStepRulerBlue
                                )}
                            />
                        </div>
                    </div>
                )}
                {step === 'add-token' && (
                    <>
                        {error && <ErrorAlert error={error} />}
                        <Form onSubmit={onSubmit}>
                            <div className="form-group">
                                <label htmlFor="token">Personal access token</label>
                                <input
                                    id="token"
                                    name="token"
                                    type="text"
                                    className="form-control test-add-credential-modal-input"
                                    required={true}
                                    spellCheck="false"
                                    minLength={1}
                                    value={credential}
                                    onChange={onChangeCredential}
                                />
                                <p className="form-text">{helpTexts[externalServiceKind]}</p>
                            </div>
                            <div className="d-flex justify-content-end">
                                <button
                                    type="button"
                                    disabled={loading}
                                    className="btn btn-outline-secondary mr-2"
                                    onClick={onCancel}
                                >
                                    Cancel
                                </button>
                                <button
                                    type="submit"
                                    disabled={loading || credential.length === 0}
                                    className="btn btn-primary test-add-credential-modal-submit"
                                >
                                    {loading && <LoadingSpinner className="icon-inline" />}
                                    {requiresSSH ? 'Next' : 'Add credential'}
                                </button>
                            </div>
                        </Form>
                    </>
                )}
                {step === 'get-ssh-key' && (
                    <>
                        <p>
                            An SSH key has been generated for your batch changes code host connection. Copy the public
                            key below and enter it on your code host.
                        </p>
                        <CodeHostSshPublicKey externalServiceKind={externalServiceKind} sshPublicKey={sshPublicKey!} />
                        <div className="d-flex justify-content-end">
                            <button type="button" className="btn btn-outline-secondary mr-2" onClick={afterCreate}>
                                Close
                            </button>
                            <button
                                type="button"
                                className="btn btn-primary test-add-credential-modal-submit"
                                onClick={afterCreate}
                            >
                                Add credential
                            </button>
                        </div>
                    </>
                )}
            </div>
        </Dialog>
    )
}
