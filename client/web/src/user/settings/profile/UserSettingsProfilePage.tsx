import React, { useCallback, useEffect } from 'react'
import H from 'history'
import { percentageDone } from '../../../../../shared/src/components/activation/Activation'
import { ActivationChecklist } from '../../../../../shared/src/components/activation/ActivationChecklist'
import { gql } from '../../../../../shared/src/graphql/graphql'
import { isErrorLike } from '../../../../../shared/src/util/errors'
import { refreshAuthenticatedUser } from '../../../auth'
import { PageTitle } from '../../../components/PageTitle'
import { eventLogger } from '../../../tracking/eventLogger'
import { UserAreaRouteContext } from '../../area/UserArea'
import { EditUserProfilePage as EditUserProfilePageFragment } from '../../../graphql-operations'
import { EditUserProfileForm } from './EditUserProfileForm'

export const EditUserProfilePageGQLFragment = gql`
    fragment EditUserProfilePage on User {
        id
        username
        displayName
        avatarURL
        viewerCanChangeUsername
        siteAdmin
    }
`

interface Props extends Pick<UserAreaRouteContext, 'onUserUpdate' | 'activation'> {
    user: EditUserProfilePageFragment

    history: H.History
    location: H.Location
}

export const UserSettingsProfilePage: React.FunctionComponent<Props> = ({
    user,
    onUserUpdate: parentOnUpdate,
    ...props
}) => {
    useEffect(() => eventLogger.logViewEvent('UserProfile'), [])

    const onUpdate = useCallback<React.ComponentProps<typeof EditUserProfileForm>['onUpdate']>(
        newValue => {
            // Handle when username changes.
            if (newValue.username !== user.username) {
                props.history.push(`/users/${newValue.username}/settings/profile`)
            }

            parentOnUpdate(newValue)

            // In case the edited user is the current user, immediately reflect the changes in the
            // UI.
            refreshAuthenticatedUser()
                .toPromise()
                .finally(() => {})
        },
        [parentOnUpdate, props.history, user.username]
    )

    return (
        <div className="user-settings-profile-page">
            <PageTitle title="Profile" />
            <h2>Profile</h2>

            {props.activation?.completed && percentageDone(props.activation.completed) < 100 && (
                <div className="card mb-3">
                    <div className="card-body">
                        <h3 className="mb-0">Almost there!</h3>
                        <p className="mb-0">Complete the steps below to finish onboarding to Sourcegraph.</p>
                    </div>
                    <ActivationChecklist
                        history={props.history}
                        steps={props.activation.steps}
                        completed={props.activation.completed}
                    />
                </div>
            )}
            {user && !isErrorLike(user) && (
                <EditUserProfileForm
                    user={user}
                    initialValue={user}
                    onUpdate={onUpdate}
                    after={
                        window.context.sourcegraphDotComMode && (
                            <p className="mt-4">
                                <a href="https://about.sourcegraph.com/contact">Contact support</a> to delete your
                                account.
                            </p>
                        )
                    }
                />
            )}
        </div>
    )
}
