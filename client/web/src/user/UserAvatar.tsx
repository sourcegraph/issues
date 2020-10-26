import AccountCircleIcon from 'mdi-react/AccountCircleIcon'
import React from 'react'
import classNames from 'classnames'

interface Props {
    size?: number
    user: {
        avatarURL: string | null
    }
    className?: string
    ['data-tooltip']?: string
    targetID?: string
}

/**
 * UserAvatar displays the avatar of a user.
 */
export const UserAvatar: React.FunctionComponent<Props> = ({
    size,
    user,
    className,
    targetID,
    // Exclude children since neither <img /> nor mdi-react icons receive them
    children,
    ...otherProps
}) => {
    if (user?.avatarURL) {
        let url = user.avatarURL
        try {
            const urlObject = new URL(user.avatarURL)
            if (size) {
                urlObject.searchParams.set('s', size.toString())
            }
            url = urlObject.href
        } catch {
            // noop
        }
        return <img className={classNames('user-avatar', className)} src={url} id={targetID} {...otherProps} />
    }
    return <AccountCircleIcon className={classNames('user-avatar', className)} id={targetID} {...otherProps} />
}
