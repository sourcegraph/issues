import { Observable } from 'rxjs'
import { map } from 'rxjs/operators'
import { RepoNotFoundError } from '../../../../shared/src/backend/errors'
import { dataOrThrowErrors, gql } from '../../../../shared/src/graphql/graphql'
import * as GQL from '../../../../shared/src/graphql/schema'
import { queryGraphQL } from '../../backend/graphql'

/**
 * Fetches a repository.
 */
export function fetchRepository(name: string): Observable<GQL.IRepository> {
    return queryGraphQL(
        gql`
            query Repository($name: String!) {
                repository(name: $name) {
                    id
                    name
                    isPrivate
                    viewerCanAdminister
                    mirrorInfo {
                        remoteURL
                        cloneInProgress
                        cloneProgress
                        cloned
                        updatedAt
                        updateSchedule {
                            due
                            index
                            total
                        }
                        updateQueue {
                            updating
                            index
                            total
                        }
                    }
                    externalServices {
                        nodes {
                            id
                            kind
                            displayName
                        }
                    }
                    permissionsInfo {
                        syncedAt
                        updatedAt
                    }
                }
            }
        `,
        { name }
    ).pipe(
        map(dataOrThrowErrors),
        map(data => {
            if (!data.repository) {
                throw new RepoNotFoundError(name)
            }
            return data.repository
        })
    )
}
