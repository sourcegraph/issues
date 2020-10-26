# Working with GraphQL queries

The webapp and browser extension interact with our backend through a strongly typed GraphQL API.
We auto-generate TypeScript types for the schema and all queries, mutations and fragments to ensure the client uses the API correctly.

## Writing a type-safe GraphQL query

Write GraphQL queries in plain template strings and tag them with the `gql` template string tag.
Each query must also have a globally unique name.
This makes sure the query gets picked up by the type generation and you receive autocompletion, syntax highlighting, hover tooltips and validation.

The preferred way to get a typed result from a GraphQL query or mutation is to pass the auto-generated interface for the query result and variables as type parameters to `requestGraphQL()`:

```ts
import { DemoResult, DemoVariables} from '../graphql-operations'

requestGraphQl<DemoResult, DemoVariables>(gql`
    query Demo($input: String) {
        foo(input: $input) {
            bar
        }
    }
`, {
    input: 'Hello world'
})
```

**Note**: A lot of older code uses the non-generic `queryGraphQl()` and `mutateGraphQl()` functions.
These are less type-safe, because they return schema types with _all_ fields of the schema present, no matter whether they were queried or not.

## Writing a React component or function that takes an API object as input

React components are often structured to display a subfield of a query.
The best way to declare this input type is to define a _GraphQL fragment_ with the component, then using the auto-generated type for that fragment.
This ensures the parent component don't forget to query a required field and it makes it easy to hard-code stub results in tests.

```tsx
import { PersionFields } from '../graphql-operations'

export const personFields = gql`
    fragment PersonFields on Person {
        name
    }
`

export const Greeting: React.FunctionComponent<{ person: PersionFields }> = ({ person }) =>
    <div>Hello, {person.name}!</div>
```

Since the fragment is exported, parent components can use it in their queries to include the needed data:

```ts
import { personFields } from './greeting',

requestGraphQl<PeopleResult>(gql`
    query People {
        people {
            nodes {
                ...PersonFields
            }
        }
    }
    ${personFields}
`)
```

**Note**: A lot of older components still use all-fields types generated from the whole schema (as opposed to from a fragment), usually referenced from the namespace `GQL.*`.
This is less safe as fields could be missing from the actual queries and it makes testing harder as hard-coded results need to be casted to the whole type.
Some components also worked around this by redeclaring the type structure with complex `Pick<T, K>` expressions.
When you need to interface with these, consider refactoring them to use a fragment type instead.
