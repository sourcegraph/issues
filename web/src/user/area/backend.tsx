import { Observable } from 'rxjs'
import { map } from 'rxjs/operators'
import { gql, dataOrThrowErrors } from '../../../../shared/src/graphql/graphql'
import { queryGraphQL } from '../../backend/graphql'
import { UserResult, UserVariables } from '../../graphql-operations'

export type User = NonNullable<UserResult['user']>

export const fetchUser = (args: UserVariables): Observable<User | null> =>
    queryGraphQL<UserResult>(
        gql`
            query User($username: String!, $siteAdmin: Boolean!) {
                user(username: $username) {
                    __typename
                    id
                    username
                    displayName
                    url
                    settingsURL
                    avatarURL
                    viewerCanAdminister
                    siteAdmin
                    builtinAuth
                    createdAt
                    emails {
                        email
                        verified
                    }
                    organizations {
                        nodes {
                            id
                            displayName
                            name
                        }
                    }
                    permissionsInfo @include(if: $siteAdmin) {
                        syncedAt
                        updatedAt
                    }
                }
            }
        `,
        args
    ).pipe(
        map(dataOrThrowErrors),
        map(data => {
            if (!data.user) {
                throw new Error(`User not found: ${JSON.stringify(args.username)}`)
            }
            return data.user
        })
    )
