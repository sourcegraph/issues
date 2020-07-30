import { Observable } from 'rxjs'
import { map } from 'rxjs/operators'
import { gql, dataOrThrowErrors } from '../../../shared/src/graphql/graphql'
import * as GQL from '../../../shared/src/graphql/schema'
import { queryGraphQL } from '../backend/graphql'
import { SymbolsResult } from '../graphql-operations'

/**
 * Fetches symbols.
 */
export function fetchSymbols(
    repo: GQL.Scalars['ID'],
    revision: string,
    args: { first?: number; query?: string; includePatterns?: string[] }
): Observable<NonNullable<(SymbolsResult['node'] & { __typename: 'Repository' })['commit']>['symbols']> {
    return queryGraphQL<SymbolsResult>(
        gql`
            query Symbols($repo: ID!, $revision: String!, $first: Int, $query: String, $includePatterns: [String!]) {
                node(id: $repo) {
                    __typename
                    ... on Repository {
                        commit(rev: $revision) {
                            symbols(first: $first, query: $query, includePatterns: $includePatterns) {
                                pageInfo {
                                    hasNextPage
                                }
                                nodes {
                                    name
                                    containerName
                                    kind
                                    language
                                    location {
                                        resource {
                                            path
                                        }
                                        range {
                                            start {
                                                line
                                                character
                                            }
                                            end {
                                                line
                                                character
                                            }
                                        }
                                    }
                                    url
                                }
                            }
                        }
                    }
                }
            }
        `,
        { ...args, repo, revision }
    ).pipe(
        map(dataOrThrowErrors),
        map(({ node }) => {
            if (!node) {
                throw new Error(`Node ${repo} not found`)
            }
            if (node.__typename !== 'Repository') {
                throw new Error(`Node is a ${node.__typename}, not a Repository`)
            }
            if (!node.commit?.symbols?.nodes) {
                throw new Error('Could not resolve commit symbols for repository')
            }
            return node.commit.symbols
        })
    )
}
