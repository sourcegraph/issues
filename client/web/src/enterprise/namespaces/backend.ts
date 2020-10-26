import { gql, dataOrThrowErrors } from '../../../../shared/src/graphql/graphql'
import { queryGraphQL } from '../../backend/graphql'
import { Observable } from 'rxjs'
import { Namespace } from '../../../../shared/src/graphql/schema'
import { map } from 'rxjs/operators'

export const queryNamespaces = (): Observable<Pick<Namespace, '__typename' | 'id' | 'namespaceName' | 'url'>[]> =>
    queryGraphQL(
        gql`
            query ViewerNamespaces {
                # TODO expose combined namespaces field
                currentUser {
                    __typename
                    id
                    namespaceName
                    url
                    organizations {
                        nodes {
                            __typename
                            id
                            namespaceName
                            url
                        }
                    }
                }
            }
        `
    ).pipe(
        map(dataOrThrowErrors),
        map(data => {
            if (!data.currentUser) {
                return []
            }
            return [data.currentUser, ...data.currentUser.organizations.nodes]
        })
    )
