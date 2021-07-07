import { Observable as ApolloObservable, gql as apolloGql } from '@apollo/client'
import type { DocumentNode } from 'graphql'
import { Observable, observable } from 'rxjs'

import type { GraphQLRequestDocument } from './graphql'

// Apollo's QueryObservable is not compatible with RxJS
// https://github.com/kamilkisiela/apollo-angular/blob/ed3bd18c7f0a514676fc0300e68fd249a441dffb/packages/apollo-angular/src/utils.ts#L73-L85
export function fixApolloObservable<T>(apolloObservable: ApolloObservable<T>): Observable<T> {
    ;(apolloObservable as any)[observable] = () => apolloObservable
    return apolloObservable as any
}

/**
 * Returns a `DocumentNode` value to support integrations with GraphQL clients that require this.
 *
 * @param document The GraphQL operation payload
 * @returns The created `DocumentNode`
 */
export const getDocumentNode = (document: GraphQLRequestDocument): DocumentNode => {
    if (typeof document === 'string') {
        return apolloGql(document)
    }
    return document
}
