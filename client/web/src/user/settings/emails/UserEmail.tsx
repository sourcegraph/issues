import React, { FunctionComponent } from 'react'
import { UserEmailsResult } from '../../../graphql-operations'
import { useRemoveUserEmail, useResendEmailVerification, useSetUserEmailVerified } from './useUserEmail'

interface Props {
    user: string
    email: NonNullable<UserEmailsResult['node']>['emails'][number]
}

export const UserEmail: FunctionComponent<Props> = ({
    user,
    email: { email, isPrimary, verified, verificationPending, viewerCanManuallyVerify },
}) => {
    const { mutate: mutateEmailVerified, isLoading: emailVerifiedLoading } = useSetUserEmailVerified()
    const {
        mutate: mutateResendEmailVerification,
        isLoading: resendEmailVerificationLoading,
    } = useResendEmailVerification()

    const { mutate: mutateRemoveEmail, isLoading: removeEmailLoading } = useRemoveUserEmail()

    const removeEmail = (user: string, email: string): void => {
        mutateRemoveEmail({ user, email })
    }

    const updateEmailVerification = (verified: boolean): void => {
        mutateEmailVerified({ user, email, verified })
    }

    const resendEmailVerification = (user: string, email: string): void => {
        mutateResendEmailVerification({ user, email })
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
                                className="btn btn-link text-primary p-0"
                                onClick={() => resendEmailVerification(user, email)}
                                disabled={emailVerifiedLoading}
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
                            className="btn btn-link text-primary p-0"
                            onClick={() => updateEmailVerification(!verified)}
                            disabled={resendEmailVerificationLoading}
                        >
                            {verified ? 'Mark as unverified' : 'Mark as verified'}
                        </button>
                    )}{' '}
                    {!isPrimary && (
                        <button
                            type="button"
                            className="btn btn-link text-danger p-0"
                            onClick={() => removeEmail(user, email)}
                            disabled={removeEmailLoading}
                        >
                            Remove
                        </button>
                    )}
                </div>
            </div>
        </>
    )
}
