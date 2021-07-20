import { ApolloError, ApolloQueryResult } from '@apollo/client'

import { gql, useQuery } from '@sourcegraph/shared/src/graphql/graphql'

import { Maybe, AffiliatedRepositoriesResult, AffiliatedRepositoriesVariables, Exact } from '../graphql-operations'

interface UseAffiliatedReposResult {
    affiliatedRepos: AffiliatedRepositoriesResult['affiliatedRepositories']['nodes'] | undefined
    loadingAffiliatedRepos: boolean
    errorAffiliatedRepos: ApolloError | undefined
    refetchAffiliatedRepos:
        | ((
              variables?:
                  | Partial<
                        Exact<{
                            user: string
                            codeHost: Maybe<string>
                            query: Maybe<string>
                        }>
                    >
                  | undefined
          ) => Promise<ApolloQueryResult<AffiliatedRepositoriesResult>>)
        | undefined
}

const AFFILIATED_REPOS = gql`
    query UserAffiliatedRepositories($user: ID!, $codeHost: ID, $query: String) {
        affiliatedRepositories(user: $user, codeHost: $codeHost, query: $query) {
            nodes {
                name
                codeHost {
                    kind
                    id
                    displayName
                }
                private
            }
        }
    }
`

export const useAffiliatedRepos = (userId: string): UseAffiliatedReposResult => {
    const { data, loading, error, refetch } = useQuery<AffiliatedRepositoriesResult, AffiliatedRepositoriesVariables>(
        AFFILIATED_REPOS,
        {
            variables: {
                user: userId,
                codeHost: null,
                query: null,
            },
        }
    )

    return {
        affiliatedRepos: data?.affiliatedRepositories.nodes,
        loadingAffiliatedRepos: loading,
        errorAffiliatedRepos: error,
        refetchAffiliatedRepos: refetch,
    }
}
