import React from 'react'
import { Link } from 'react-router-dom'
import * as GQL from '../../../../../shared/src/graphql/schema'
import { userURL } from '../../../user'

/**
 * Displays the account name as a link.
 */
export const AccountName: React.FunctionComponent<{
    account: Pick<GQL.IUser, 'username' | 'displayName'> | null
    link?: string
}> = ({ account, link }) =>
    account ? (
        <>
            <Link to={link || userURL(account.username)}>{account.username}</Link>{' '}
            {account.displayName && `(${account.displayName})`}
        </>
    ) : (
        <em>(Account deleted)</em>
    )
