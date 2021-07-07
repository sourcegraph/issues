# Working with GraphQL

The webapp and browser extension interact with our backend through a strongly typed GraphQL API.
We auto-generate TypeScript types for the schema and all queries, mutations and fragments to ensure the client uses the API correctly.

## GraphQL Client
We use [Apollo Client](https://www.apollographql.com/docs/react/) to manage data-fetching and caching within our app. It provides a set of declarative interfaces which abstract away a lot of repeated code, whilst supporting a 'stale-while-revalidate' strategy through a normalized client cache of requested data.

**Writing and running your query**

We use `gql` template strings to declare our GraphQL queries.

Each query must have a globally unique name as per the [GraphQL specification](https://spec.graphql.org/June2018/#sec-Operation-Name-Uniqueness). Typically we should name our queries similarly to how we might name a function, by describing what the query will return. For mutations, we should prefix the name with a verb like `Delete` or `Update`, this will help avoid collisions between queries and mutations.

Using each unique query, we can generate specific types so you can receive autocompletion, syntax highlighting, hover tooltips and validation in your IDE.

Once you have built your query, `graphql-codegen` will generate the correct request and response types. This process should happen automatically through local development, you can also manually trigger this by running `yarn generate` or `yarn watch-generate`.

Using a `useQuery` hook, we can easily fire a request and handle the response correctly.

```ts
// ./MyComponent.tsx
import { useQuery, gql } from '@sourcegraph/shared/src/graphql/graphql'

import { UserDisplayNameResult, UserDisplayNameVariables } from '../../graphql-operations'

export const USER_DISPLAY_NAME = gql`
  query UserDisplayName($username: String!) {
      user(username: $username) {
          id
          displayName
      }
  }
`

const MyComponent = ({ username }: { username: string }) => {
  const { data, loading, error } = useQuery<UserDisplayNameResult, UserDisplayNameVariables>(USER_DISPLAY_NAME, { variables: { username } });

  if (loading) {
    // handle loading state
  }

  if (error) {
    // handle error state
  }

  if (data) {
    // display to user
  }
}
```

Equally, it is possible to create our own wrapper hooks, when we want to modify data accordingly.

```ts
const useFullName = (variables: UserDisplayNameVariables): ApolloQueryResult<{ fullName: string }> => {
    const response = useQuery<UserDisplayNameResult, UserDisplayNameVariables>(USER_DISPLAY_NAME, { variables })

    if (response.error) {
        // Handle error
    }

    return {
        ...response,
        data: {
            fullName: `${response.data.user.firstName} ${response.data.user.lastName}`,
        },
    }
}
```

## Frequently asked questions

### How do I use the Apollo cache?
Apollo uses a normalized in-memory cache to store the results of different queries, most of this happens automatically!

Apollo generates a composite key for each identifiable object in a response. This is typically done by combining the `__typeName` with the `id` that is returned in the response. For example:

When firing this request:
```
{
  user {
    __typename
    id
    displayName
  }
}
```

Assuming `__typename === User` and `id === 1234`, the response data is added to the cache under the key:
`User:1234`.

When a different query requests similar data, Apollo will merge both responses into the cache and update both parts of the UI with the latest data. This is useful to form links between components whilst still ensuring that the components are self-contained. For example, if a mutation updated `displayName` in a different component, then these components that *query* `displayName` would automatically receive the updated value and re-render.

**All queries should return an object identifier**. If this identifier is not under the `id` field, we need to inform Apollo which field to use in order to generate the correct key. See the [docs](https://www.apollographql.com/docs/react/caching/cache-configuration/#customizing-identifier-generation-by-type) for more information on this.

### How should I write tests that handle data-fetching?
Apollo lets us easily mock queries in our tests without having to actually mock out our own logic. The tests will fail if an un-mocked query fires through Apollo, so it is important to accurately build mock requests. In order to test how the UI displays a response, you can provide a mocked result. See this example:

```ts
import { MockedProvider } from '@apollo/client/testing'
import { render } from '@testing-library/react'

import { getDocumentNode } from '@sourcegraph/shared/src/graphql/utils'

import { MyComponent, USER_DISPLAY_NAME } from './MyComponent'

const mocks = [
    {
        request: {
            query: getDocumentNode(USER_DISPLAY_NAME),
            variables: {
                username: 'mock_username',
            },
        },
        result: {
            data: {
                user: {
                    displayName: 'Mock DisplayName',
                },
            },
        },
    },
]

describe('My Test', () => {
    it('works', () => {
        const { getByText } = render(
            <MockedProvider mocks={mocks}>
                <MyComponent />
            </MockedProvider>
        )
        expect(getByText('Your display name is: Mock DisplayName')).toBeVisible();
    })
})
```

### How can I run a query outside of React?
Most queries should be requested in the context of our UI and should use hooks. If there is a scenario where this is not possible, it is still possible to realise the benefits of Apollo without relying this approach. We can imperatively trigger any query using `client.query`.

```ts
import { getDocumentNode } from '@sourcegraph/shared/src/graphql/utils'

import { client } from './backend/graphql'
import {
    UserDisplayNameResult,
    UserDisplayNameVariables,
} from './graphql-operations'

const getUserDisplayName = async (username: string): Promise<UserDisplayNameResult> => {
    const { data, error } = await client.query<UserDisplayNameResult, UserDisplayNameVariables>({
        query: getDocumentNode(UserDisplayName),
        variables: { username },
    })

    if (error) {
        // handle error
    }

    return data
}
```

### I have an issue, how can I debug?
Aside from typical debugging methods, Apollo provides [Apollo Client Devtools](https://www.apollographql.com/docs/react/development-testing/developer-tooling/#apollo-client-devtools) as a browser extension to help with debugging. This extension will automatically track query requests and responses, and provide a visual representation of cached data.

### We have different ways of requesting data from GraphQL, why?
Our code isn't yet fully aligned to a single-approach, but this is something we are working towards over time.

A lot of older code uses the non-generic `queryGraphQl()` and `mutateGraphQl()` functions.
These are less type-safe, because they return schema types with _all_ fields of the schema present, no matter whether they were queried or not.

Other code uses `requestGraphQL()`, this is an improved approach which provides better types, but it doesn't scale well when requesting data across multiple areas of the application, and often leads to cross-component dependencies.

Our goal is to migrate more code to use Apollo. This should make our components more self-contained, increase perceived performance with client-side caching and make it easier to write effective tests.

## Writing a React component or function that takes an API object as input

React components are often structured to display a subfield of a query.
The best way to declare this input type is to define a _GraphQL fragment_ with the component, then using the auto-generated type for that fragment.
This ensures the parent component don't forget to query a required field and it makes it easy to hard-code stub results in tests.

```tsx
import { PersonFields } from '../graphql-operations'

export const personFields = gql`
    fragment PersonFields on Person {
        name
    }
`

export const Greeting: React.FunctionComponent<{ person: PersonFields }> = ({ person }) =>
    <div>Hello, {person.name}!</div>
```

Since the fragment is exported, parent components can use it in their queries to include the needed data:

```ts
import { personFields } from './greeting',

export const PEOPLE = gql`
    query People {
        people {
            nodes {
                ...PersonFields
            }
        }
    }
    ${personFields}
`
```

**Note**: A lot of older components still use all-fields types generated from the whole schema (as opposed to from a fragment), usually referenced from the namespace `GQL.*`.
This is less safe as fields could be missing from the actual queries and it makes testing harder as hard-coded results need to be casted to the whole type.
Some components also worked around this by redeclaring the type structure with complex `Pick<T, K>` expressions.
When you need to interface with these, consider refactoring them to use a fragment type instead.

## Migrating to Apollo

It should be relatively straightforward to refactor existing code to use Apollo. A typical migration will look like this:

1. Replace `requestGraphQL | queryGraphQL | mutateGraphQL` with `useQuery` or `useMutation` hooks. This may mean refactoring React `class` components into functional ones. 
    - Some components may be problematic to refactor right now. If this is the case, it is still possible to benefit from the Apollo cache by using `watchQuery`. This works similarly to `requestGraphQL` but routes the request through the Apollo cache. You should always handle the result as an `Observable` (i.e. not convert it to a promise) so the component can listen to updates from the Apollo cache.

2. Ensure that the query/mutation is correctly updating the Apollo cache by returning an `id` alongside the relevant data. Typically Apollo will warn in the console if this is not the case, you should also install [Apollo Client Devtools](https://www.apollographql.com/docs/react/development-testing/developer-tooling/#apollo-client-devtools) to help with debugging.
    - In some cases this might mean we can remove some code. For example, if two components are now able to use the Apollo cache as a source of truth, those components may no longer need to communicate with each other to pass data back and forth (common with mutations).
    - Mutations should return updated data. If I fire a mutation to change my `username`, the mutation should return the updated `username` alongside my user `id` so the cache can refresh correctly. If this is not possible, consider changing the backend to return the correct data, or consult the [Apollo docs to manually update the cache](https://www.apollographql.com/docs/react/caching/cache-interaction/#using-cachemodify).

3. Add/fix tests. We should be able to correctly mock each query/mutation without having to change too much of our tests. Check the [Apollo testing documentation](https://www.apollographql.com/docs/react/development-testing/testing/) for further information.

If you are running into problems, or have any questions at all, please get in touch with the [Frontend platform team](https://about.sourcegraph.com/handbook/engineering/developer-insights/frontend-platform).
