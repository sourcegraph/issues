import React, { useCallback } from 'react'
import { UsernameInput } from '../../../auth/SignInSignUpCommon'
import * as GQL from '../../../../../shared/src/graphql/schema'
import { UserAvatar } from '../../UserAvatar'
import { USER_DISPLAY_NAME_MAX_LENGTH } from '../..'

export type UserProfileFormFieldsValue = Pick<GQL.IUser, 'username' | 'displayName' | 'avatarURL'>

interface Props {
    value: UserProfileFormFieldsValue
    onChange: (newValue: UserProfileFormFieldsValue) => void
    usernameFieldDisabled?: boolean
    disabled?: boolean
}

export const UserProfileFormFields: React.FunctionComponent<Props> = ({
    value,
    onChange,
    usernameFieldDisabled,
    disabled,
}) => {
    const onUsernameChange = useCallback<React.ChangeEventHandler<HTMLInputElement>>(
        event => onChange({ ...value, username: event.target.value }),
        [onChange, value]
    )
    const onDisplayNameChange = useCallback<React.ChangeEventHandler<HTMLInputElement>>(
        event => onChange({ ...value, displayName: event.target.value }),
        [onChange, value]
    )
    const onAvatarURLChange = useCallback<React.ChangeEventHandler<HTMLInputElement>>(
        event => onChange({ ...value, avatarURL: event.target.value }),
        [onChange, value]
    )

    return (
        <div className="user-profile-form-fields">
            <div className="form-group">
                <label htmlFor="UserProfileFormFields__username">Username</label>
                <UsernameInput
                    id="UserProfileFormFields__username"
                    className="test-UserProfileFormFields-username"
                    value={value.username}
                    onChange={onUsernameChange}
                    required={true}
                    disabled={usernameFieldDisabled || disabled}
                    aria-describedby="UserProfileFormFields__username-help"
                />
                <small id="UserProfileFormFields__username-help" className="form-text text-muted">
                    A username consists of letters, numbers, hyphens (-), dots (.) and may not begin or end with a dot,
                    nor begin with a hyphen.
                </small>
            </div>
            <div className="form-group">
                <label htmlFor="UserProfileFormFields__displayName">Display name</label>
                <input
                    id="UserProfileFormFields__displayName"
                    type="text"
                    className="form-control test-UserProfileFormFields__displayName"
                    value={value.displayName || ''}
                    onChange={onDisplayNameChange}
                    disabled={disabled}
                    spellCheck={false}
                    placeholder="Display name"
                    maxLength={USER_DISPLAY_NAME_MAX_LENGTH}
                />
            </div>
            <div className="d-flex align-items-center">
                <div className="form-group w-100">
                    <label htmlFor="UserProfileFormFields__avatarURL">Avatar URL</label>
                    <input
                        id="UserProfileFormFields__avatarURL"
                        type="url"
                        className="form-control test-UserProfileFormFields__avatarURL"
                        value={value.avatarURL || ''}
                        onChange={onAvatarURLChange}
                        disabled={disabled}
                        spellCheck={false}
                        placeholder="URL to avatar photo"
                    />
                </div>
                {value.avatarURL && <UserAvatar user={value} className="user-profile-form-fields__avatar ml-2" />}
            </div>
        </div>
    )
}
