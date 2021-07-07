import { InMemoryCache } from '@apollo/client'

import { TypedTypePolicies } from '../graphql-operations'

const typePolicies: TypedTypePolicies = {
    BatchChangesCodeHost: {
        keyFields: ['externalServiceKind', 'externalServiceURL'],
    },
}

export const cache = new InMemoryCache({
    typePolicies,
})
