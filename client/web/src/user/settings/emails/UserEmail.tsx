import React, { useState, FunctionComponent } from 'react'

import { dataOrThrowErrors, gql } from '@sourcegraph/shared/src/graphql/graphql'
import { asError, ErrorLike } from '@sourcegraph/shared/src/util/errors'

import { requestGraphQL } from '../../../backend/graphql'
import {
    UserEmailsResult,
    RemoveUserEmailResult,
    RemoveUserEmailVariables,
    SetUserEmailVerifiedResult,
    SetUserEmailVerifiedVariables,
    ResendVerificationEmailResult,
    ResendVerificationEmailVariables,
} from '../../../graphql-operations'
import { eventLogger } from '../../../tracking/eventLogger'

interface Props {
    user: string
    email: NonNullable<UserEmailsResult['node']>['emails'][number]
    onError: (error: ErrorLike) => void

    onDidRemove?: (email: string) => void
    onEmailVerify?: () => void
    onEmailResendVerification?: () => void
}

export const UserEmail: FunctionComponent<Props> = ({
    user,
    email: { email, isPrimary, verified, verificationPending, viewerCanManuallyVerify },
    onError,
    onDidRemove,
    onEmailVerify,
    onEmailResendVerification,
}) => {
    const [isLoading, setIsLoading] = useState(false)

    const handleError = (error: ErrorLike): void => {
        onError(asError(error))
        setIsLoading(false)
    }

    const removeEmail = async (): Promise<void> => {
        setIsLoading(true)

        try {
            dataOrThrowErrors(
                await requestGraphQL<RemoveUserEmailResult, RemoveUserEmailVariables>(
                    gql`
                        mutation RemoveUserEmail($user: ID!, $email: String!) {
                            removeUserEmail(user: $user, email: $email) {
                                id
                            }
                        }
                    `,
                    { user, email }
                ).toPromise()
            )

            setIsLoading(false)
            eventLogger.log('UserEmailAddressDeleted')

            if (onDidRemove) {
                onDidRemove(email)
            }
        } catch (error) {
            handleError(error)
        }
    }

    const updateEmailVerification = async (verified: boolean): Promise<void> => {
        setIsLoading(true)

        try {
            dataOrThrowErrors(
                await requestGraphQL<SetUserEmailVerifiedResult, SetUserEmailVerifiedVariables>(
                    gql`
                        mutation SetUserEmailVerified($user: ID!, $email: String!, $verified: Boolean!) {
                            setUserEmailVerified(user: $user, email: $email, verified: $verified) {
                                id
                            }
                        }
                    `,
                    { user, email, verified }
                ).toPromise()
            )

            setIsLoading(false)

            if (verified) {
                eventLogger.log('UserEmailAddressMarkedVerified')
            } else {
                eventLogger.log('UserEmailAddressMarkedUnverified')
            }

            if (onEmailVerify) {
                onEmailVerify()
            }
        } catch (error) {
            handleError(error)
        }
    }

    const resendEmailVerification = async (email: string): Promise<void> => {
        setIsLoading(true)

        try {
            dataOrThrowErrors(
                await requestGraphQL<ResendVerificationEmailResult, ResendVerificationEmailVariables>(
                    gql`
                        mutation ResendVerificationEmail($user: ID!, $email: String!) {
                            resendVerificationEmail(user: $user, email: $email) {
                                alwaysNil
                            }
                        }
                    `,
                    { user, email }
                ).toPromise()
            )

            setIsLoading(false)
            eventLogger.log('UserEmailAddressVerificationResent')

            onEmailResendVerification?.()
        } catch (error) {
            handleError(error)
        }
    }

    return (
        <>
            <div className="d-flex align-items-center justify-content-between">
                <div>
                    <span className="mr-2">{email}</span>
                    {verified && <span className="badge badge-success mr-1">Verified</span>}
                    {!verified && !verificationPending && (
                        <span className="badge badge-secondary mr-1">Not verified</span>
                    )}
                    {isPrimary && <span className="badge badge-primary mr-1">Primary</span>}
                    {!verified && verificationPending && (
                        <span>
                            <span className="user-settings-emails-page__dot">&bull;&nbsp;</span>
                            <button
                                type="button"
                                className="btn btn-link p-0"
                                onClick={() => resendEmailVerification(email)}
                                disabled={isLoading}
                            >
                                Resend verification email
                            </button>
                        </span>
                    )}
                </div>
                <div>
                    {viewerCanManuallyVerify && (
                        <button
                            type="button"
                            className="btn btn-link p-0"
                            onClick={() => updateEmailVerification(!verified)}
                            disabled={isLoading}
                        >
                            {verified ? 'Mark as unverified' : 'Mark as verified'}
                        </button>
                    )}{' '}
                    {!isPrimary && (
                        <button
                            type="button"
                            className="btn btn-link text-danger p-0"
                            onClick={removeEmail}
                            disabled={isLoading}
                        >
                            Remove
                        </button>
                    )}
                </div>
            </div>
        </>
    )
}
