import * as React from 'react'
import { LinkOrSpan } from '../../../shared/src/components/LinkOrSpan'
import { SignatureFields } from '../graphql-operations'

/**
 * Formats a person name to: "username (Display Name)" or "Display Name"
 */
export const formatPersonName = ({ user, displayName }: SignatureFields['person']): string =>
    user ? user.username : displayName

/**
 * A person's name, with a link to their Sourcegraph user profile if an associated user account is
 * found.
 */
export const PersonLink: React.FunctionComponent<{
    /** The person to link to. */
    person: SignatureFields['person']

    /** A class name that is always applied. */
    className?: string

    /** A class name applied when there is an associated user account. */
    userClassName?: string
}> = ({ person, className = '', userClassName = '' }) => (
    <LinkOrSpan
        to={person.user?.url}
        className={`${className} ${person.user ? userClassName : ''}`}
        data-tooltip={
            person.user && (person.user.displayName || person.displayName)
                ? `${person.user.displayName || person.displayName} <${person.email}>`
                : person.email
        }
    >
        {formatPersonName(person)}
    </LinkOrSpan>
)
