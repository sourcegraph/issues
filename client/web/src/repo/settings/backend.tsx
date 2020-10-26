import { Observable } from 'rxjs'
import { map } from 'rxjs/operators'
import { gql } from '../../../../shared/src/graphql/graphql'
import * as GQL from '../../../../shared/src/graphql/schema'
import { createAggregateError } from '../../../../shared/src/util/errors'
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
        map(({ data, errors }) => {
            if (!data || !data.repository || !data.repository.externalServices) {
                throw createAggregateError(errors)
            }
            return data.repository
        })
    )
}
