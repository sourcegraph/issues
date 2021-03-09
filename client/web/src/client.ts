import { GraphQLClient } from 'graphql-request'
import { GraphQLResponse, Variables } from 'graphql-request/dist/types'
import { QueryFunction } from 'react-query'

const uri = 'https://sourcegraph.test:3443/.api/graphql'

interface Parameters_<TQuery, TVariables> {
    queryKey: [TQuery, TVariables]
}

const fetcher = async <TData, TVariables extends Variables>(
    context: Parameters_<string, TVariables>
): Promise<TData> => {
    const [query, variables] = context.queryKey

    const graphQLClient = new GraphQLClient(uri, {
        headers: {
            ...window.context.xhrHeaders,
            Accept: 'application/json',
            'Content-Type': 'application/json',
            'X-Sourcegraph-Should-Trace': new URLSearchParams(window.location.search).get('trace') || 'false',
        },
    })

    const data = await graphQLClient.request(query, variables)
    return data
}

export default fetcher
