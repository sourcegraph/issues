import React, { useCallback, useState } from 'react'
import { map } from 'rxjs/operators'
import { dataOrThrowErrors, gql } from '../../../../../shared/src/graphql/graphql'
import { isErrorLike } from '../../../../../shared/src/util/errors'
import { requestGraphQL } from '../../../backend/graphql'
import { UpdateUserResult, UpdateUserVariables, UserAreaUserFields } from '../../../graphql-operations'
import { eventLogger } from '../../../tracking/eventLogger'
import { UserProfileFormFields, UserProfileFormFieldsValue } from './UserProfileFormFields'
import * as GQL from '../../../../../shared/src/graphql/schema'
import { UserAreaGQLFragment } from '../../area/UserArea'
import { Form } from '../../../../../branded/src/components/Form'

interface Props {
    user: Pick<GQL.IUser, 'id' | 'viewerCanChangeUsername' | 'siteAdmin'>

    initialValue: UserProfileFormFieldsValue

    /** Called when the user is successfully updated. */
    onUpdate: (newValue: UserAreaUserFields) => void

    after?: React.ReactFragment
}

/**
 * A form to edit a user's profile.
 */
export const EditUserProfileForm: React.FunctionComponent<Props> = ({ user, initialValue, onUpdate, after }) => {
    const [value, setValue] = useState<UserProfileFormFieldsValue>(initialValue)
    const onChange = useCallback<React.ComponentProps<typeof UserProfileFormFields>['onChange']>(
        newValue => setValue(previous => ({ ...previous, ...newValue })),
        []
    )

    /** Operation state: false (initial state), true (loading), Error, or 'success'. */
    const [opState, setOpState] = useState<boolean | Error | 'success'>(false)
    const onSubmit = useCallback<React.FormEventHandler>(
        async event => {
            event.preventDefault()
            setOpState(true)
            eventLogger.log('UpdateUserClicked')
            try {
                const updatedUser = await requestGraphQL<UpdateUserResult, UpdateUserVariables>(
                    gql`
                        mutation UpdateUser(
                            $user: ID!
                            $username: String!
                            $displayName: String
                            $avatarURL: String
                            $siteAdmin: Boolean!
                        ) {
                            updateUser(
                                user: $user
                                username: $username
                                displayName: $displayName
                                avatarURL: $avatarURL
                            ) {
                                ...UserAreaUserFields
                            }
                        }
                        ${UserAreaGQLFragment}
                    `,
                    { ...value, user: user.id, siteAdmin: user.siteAdmin }
                )
                    .pipe(
                        map(dataOrThrowErrors),
                        map(data => data.updateUser)
                    )
                    .toPromise()
                eventLogger.log('UserProfileUpdated')
                setOpState('success')
                onUpdate(updatedUser)
            } catch (error) {
                eventLogger.log('UpdateUserFailed')
                setOpState(error)
            }
        },
        [onUpdate, user.id, user.siteAdmin, value]
    )

    return (
        <Form className="w-100" onSubmit={onSubmit}>
            <UserProfileFormFields
                value={value}
                onChange={onChange}
                usernameFieldDisabled={!user.viewerCanChangeUsername}
                disabled={opState === true}
            />
            <button
                type="submit"
                className="btn btn-primary"
                disabled={opState === true}
                id="test-EditUserProfileForm__save"
            >
                Save
            </button>
            {isErrorLike(opState) && <div className="mt-3 alert alert-danger">Error: {opState.message}</div>}
            {opState === 'success' && (
                <div className="mt-3 alert alert-success test-EditUserProfileForm__success">User profile updated.</div>
            )}
            {after && (
                <>
                    <hr className="my-4" />
                    {after}
                </>
            )}
        </Form>
    )
}
