import { ApolloError, ApolloQueryResult } from '@apollo/client'

import { useQuery } from '@sourcegraph/shared/src/graphql/graphql'
import { EXTERNAL_SERVICES } from '@sourcegraph/web/src/components/externalServices/backend'

import {
    Exact,
    ExternalServicesResult,
    ExternalServicesVariables,
    ListExternalServiceFields,
    Maybe,
} from '../graphql-operations'

interface UseExternalServicesResult {
    externalServices: ListExternalServiceFields[] | undefined
    loadingServices: boolean
    errorServices: ApolloError | undefined
    refetchExternalServices: (
        variables?:
            | Partial<
                  Exact<{
                      first: Maybe<number>
                      after: Maybe<string>
                      namespace: Maybe<string>
                  }>
              >
            | undefined
    ) => Promise<ApolloQueryResult<ExternalServicesResult>>
}

export const useExternalServices = (userId: string | null): UseExternalServicesResult => {
    const { data, loading, error, refetch } = useQuery<ExternalServicesResult, ExternalServicesVariables>(
        EXTERNAL_SERVICES,
        {
            variables: {
                namespace: userId,
                first: null,
                after: null,
            },
        }
    )

    return {
        externalServices: data?.externalServices.nodes,
        loadingServices: loading,
        errorServices: error,
        refetchExternalServices: refetch,
    }
}
