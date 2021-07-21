import classNames from 'classnames'
import AlertCircleIcon from 'mdi-react/AlertCircleIcon'
import CheckCircleIcon from 'mdi-react/CheckCircleIcon'
import React, { useState, useCallback } from 'react'

import { ErrorLike } from '@sourcegraph/shared/src/util/errors'
import { useRedesignToggle } from '@sourcegraph/shared/src/util/useRedesignToggle'

import { CircleDashedIcon } from '../../../components/CircleDashedIcon'
import { LoaderButton } from '../../../components/LoaderButton'
import { ExternalServiceKind, ListExternalServiceFields } from '../../../graphql-operations'

import { RemoveCodeHostConnectionModal } from './RemoveCodeHostConnectionModal'
import { ifNotNavigated } from './UserAddCodeHostsPage'

interface CodeHostItemProps {
    kind: ExternalServiceKind
    name: string
    icon: React.ComponentType<{ className?: string }>
    navigateToAuthProvider: (kind: ExternalServiceKind) => void
    isTokenUpdateRequired: boolean | undefined
    // optional service object fields when the code host connection is active
    service?: ListExternalServiceFields

    onDidAdd?: (service: ListExternalServiceFields) => void
    onDidRemove: () => void
    onDidError: (error: ErrorLike) => void
}

export const CodeHostItem: React.FunctionComponent<CodeHostItemProps> = ({
    service,
    kind,
    name,
    isTokenUpdateRequired,
    icon: Icon,
    navigateToAuthProvider,
    onDidRemove,
    onDidError,
}) => {
    const [isRedesignEnabled] = useRedesignToggle()
    const [isRemoveConnectionModalOpen, setIsRemoveConnectionModalOpen] = useState(false)
    const toggleRemoveConnectionModal = useCallback(
        () => setIsRemoveConnectionModalOpen(!isRemoveConnectionModalOpen),
        [isRemoveConnectionModalOpen]
    )

    const [oauthInFlight, setOauthInFlight] = useState(false)

    const toAuthProvider = useCallback((): void => {
        setOauthInFlight(true)
        ifNotNavigated(() => {
            setOauthInFlight(false)
        })
        navigateToAuthProvider(kind)
    }, [kind, navigateToAuthProvider])

    return (
        <div className={classNames('d-flex align-items-start', !isRedesignEnabled && 'p-2')}>
            {service && isRemoveConnectionModalOpen && (
                <RemoveCodeHostConnectionModal
                    id={service.id}
                    kind={kind}
                    name={name}
                    repoCount={service.repoCount}
                    onDidRemove={onDidRemove}
                    onDidCancel={toggleRemoveConnectionModal}
                    onDidError={onDidError}
                />
            )}
            <div className="align-self-center">
                {service?.warning || service?.lastSyncError ? (
                    <AlertCircleIcon className="icon-inline mb-0 mr-2 text-danger" />
                ) : service?.id ? (
                    <CheckCircleIcon className="icon-inline mb-0 mr-2 text-success" />
                ) : (
                    <CircleDashedIcon className="icon-inline mb-0 mr-2 user-code-hosts-page__icon--dashed" />
                )}
                <Icon className="mb-0 mr-1" />
            </div>
            <div className="flex-1 align-self-center">
                <h3 className="m-0">{name}</h3>
            </div>
            <div className="align-self-center">
                {/* always show remove button when the service exists */}
                {service?.id && (
                    <button
                        type="button"
                        className="btn btn-link text-danger shadow-none"
                        onClick={toggleRemoveConnectionModal}
                    >
                        Remove
                    </button>
                )}

                {/* Show one of: update, updating, connect, connecting buttons */}
                {!service?.id ? (
                    oauthInFlight ? (
                        <LoaderButton
                            type="button"
                            className={classNames(
                                'btn',
                                !isRedesignEnabled && 'btn-primary',
                                isRedesignEnabled && 'btn-success'
                            )}
                            loading={true}
                            disabled={true}
                            label="Connecting..."
                            alwaysShowLabel={true}
                        />
                    ) : (
                        <button
                            type="button"
                            className={classNames(
                                'btn',
                                !isRedesignEnabled && 'btn-primary',
                                isRedesignEnabled && 'btn-success'
                            )}
                            onClick={toAuthProvider}
                        >
                            Connect
                        </button>
                    )
                ) : (
                    isTokenUpdateRequired &&
                    (oauthInFlight ? (
                        <LoaderButton
                            type="button"
                            className="btn btn-merged"
                            loading={true}
                            disabled={true}
                            label="Updating..."
                            alwaysShowLabel={true}
                        />
                    ) : (
                        <button type="button" className="btn btn-merged" onClick={toAuthProvider}>
                            Update
                        </button>
                    ))
                )}
            </div>
        </div>
    )
}
