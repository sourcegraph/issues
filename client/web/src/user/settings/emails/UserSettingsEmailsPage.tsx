import { LoadingSpinner } from '@sourcegraph/react-loading-spinner'
import React, { FunctionComponent, useEffect } from 'react'
import { useObservable } from '../../../../../shared/src/util/useObservable'
import { ErrorAlert } from '../../../components/alerts'
import { PageTitle } from '../../../components/PageTitle'
import { UserAreaUserFields, UserEmailsResult } from '../../../graphql-operations'
import { siteFlags } from '../../../site/backend'
import { eventLogger } from '../../../tracking/eventLogger'
import { AddUserEmailForm } from './AddUserEmailForm'
import { SetUserPrimaryEmailForm } from './SetUserPrimaryEmailForm'
import { UserEmail } from './UserEmail'
import { useGetUserEmail } from './useUserEmail'

interface Props {
    user: UserAreaUserFields
}

type UserEmail = NonNullable<UserEmailsResult['node']>['emails'][number]

export const UserSettingsEmailsPage: FunctionComponent<Props> = ({ user }) => {
    const { data, isLoading, error } = useGetUserEmail(user.id)

    const flags = useObservable(siteFlags)

    useEffect(() => {
        eventLogger.logViewEvent('UserSettingsEmails')
    }, [])

    return (
        <div className="user-settings-emails-page">
            <PageTitle title="Emails" />
            {flags && !flags.sendsEmailVerificationEmails && (
                <div className="alert alert-warning mt-2">
                    Sourcegraph is not configured to send email verifications. Newly added email addresses must be
                    manually verified by a site admin.
                </div>
            )}
            {error && <ErrorAlert className="mt-2" error={error} />}
            <h2>Emails</h2>
            {isLoading ? (
                <div className="d-flex justify-content-center">
                    <LoadingSpinner className="icon-inline" />
                </div>
            ) : (
                <div className="mt-4">
                    <ul className="list-group">
                        {data?.map((email: NonNullable<UserEmailsResult['node']>['emails'][number]) => (
                            <li key={email.email} className="list-group-item p-3">
                                <UserEmail user={user.id} email={email} />
                            </li>
                        ))}
                    </ul>
                    <AddUserEmailForm className="mt-4" user={user.id} />
                    <hr className="my-4" />
                    <SetUserPrimaryEmailForm user={user.id} emails={data} />
                </div>
            )}
        </div>
    )
}
