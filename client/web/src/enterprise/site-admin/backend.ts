import * as GQL from '../../../../shared/src/graphql/schema'
import { dataOrThrowErrors, gql } from '../../../../shared/src/graphql/graphql'
import { map } from 'rxjs/operators'
import { Observable } from 'rxjs'
import { queryGraphQL } from '../../backend/graphql'

/**
 * Fetch a single LSIF upload by id.
 */
export function fetchLsifUpload({ id }: { id: string }): Observable<GQL.ILSIFUpload | null> {
    return queryGraphQL(
        gql`
            query SiteAdminLsifUpload($id: ID!) {
                node(id: $id) {
                    __typename
                    ... on LSIFUpload {
                        projectRoot {
                            commit {
                                repository {
                                    name
                                    url
                                }
                            }
                        }
                    }
                }
            }
        `,
        { id }
    ).pipe(
        map(dataOrThrowErrors),
        map(({ node }) => {
            if (!node) {
                return null
            }
            if (node.__typename !== 'LSIFUpload') {
                throw new Error(`The given ID is a ${node.__typename}, not an LSIFUpload`)
            }

            return node
        })
    )
}
