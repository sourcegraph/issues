import * as React from 'react'

/**
 * OrgAvatar displays the avatar of an organization.
 */
export const OrgAvatar: React.FunctionComponent<{
    /** The organization's name. */
    org: string

    size?: 'md' | 'lg'

    className?: string
}> = ({ org, size = 'md', className = '' }) => (
    <div className={`org-avatar org-avatar--${size} ${className}`}>{org.slice(0, 2).toUpperCase()}</div>
)
