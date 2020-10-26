import React from 'react'
import * as GQL from '../../../../../shared/src/graphql/schema'

/**
 * Displays an inline list of email addresses for an account.
 */
export const AccountEmailAddresses: React.FunctionComponent<{
    emails: Pick<GQL.IUserEmail, 'email' | 'verified'>[]
}> = ({ emails }) => (
    <>
        {emails.map(({ email, verified }, index) => (
            <span key={index} className={`text-nowrap d-inline-block mr-2 ${verified ? '' : 'text-muted font-italic'}`}>
                <a href={`mailto:${email}`}>{email}</a> {verified ? '(verified)' : '(unverified)'}
            </span>
        ))}
    </>
)
