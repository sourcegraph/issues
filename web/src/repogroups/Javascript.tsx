import { RepogroupMetadata } from './types'
import * as React from 'react'
import { SearchPatternType } from '../../../shared/src/graphql-operations'
export const javascript: RepogroupMetadata = {
    title: 'Javascript',
    name: 'javascript',
    url: '/javascript',
    description:
        "Search popular Javascript repositories on GitHub. The search examples show usage of a React framework concept—a useState Hook, with various data types.",
    examples: [
        {

            title: 'Find imports of the useState with regex search',
            exampleQuery: <>import [^;]+useState[^;]+ from 'react'</>,
            rawQuery: "import [^;]+useState[^;]+ from 'react'",
            patternType: SearchPatternType.regexp,
        },
        {
            title: 'Examples of useState with an object as the input parameter',
            description: `useState takes a single argument and can accept a primitive, an array or an object.
            Syntax for usage is [state_value, function_to_update_state_value] = useState (initial_state_value).`,
            exampleQuery: <>{'useState({:[string]})'}</>,
            rawQuery: 'useState({:[string]}) count:1000',
            patternType: SearchPatternType.structural,
        },
        {
            title: 'Examples of useState with any JavaScript data type as the input parameter',
            exampleQuery: <>useState(:[string])</>,
            rawQuery: 'useState(:[string]) count:1000',
            patternType: SearchPatternType.structural,
        },
        {
            title: 'Examples of useState with any JavaScript data type as the input parameter in only TypeScript files',
            exampleQuery: (
                <>
                    useState(:[string])
                    <span className="search-keyword">lang:</span>typescript
                </>
            ),
            rawQuery: 'useState(:[string]) count:1000 lang:typescript',
            patternType: SearchPatternType.structural,
        },
    ],
    homepageDescription: 'Learn Javascript with code search examples.',
    homepageIcon: 'data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iMzIiIGhlaWdodD0iMzIiIHZpZXdCb3g9IjAgMCAzMiAzMiIgZmlsbD0ibm9uZSIgeG1sbnM9Imh0dHA6Ly93d3cudzMub3JnLzIwMDAvc3ZnIj4KPHJlY3Qgd2lkdGg9IjMyIiBoZWlnaHQ9IjMyIiBmaWxsPSIjRkZDNzAwIi8+CjxwYXRoIGQ9Ik00LjI3OTMgMjIuMzA1N0M0LjI3OTMgMjQuODczIDUuOTAxMzcgMjcuMjY4NiA5LjQ4OTI2IDI3LjI2ODZDMTIuNzg3MSAyNy4yNjg2IDE0Ljc4NTIgMjUuNDk2MSAxNC43ODUyIDIyLjAyNjRWMTIuMzM2OUgxMS4yMTg4VjIxLjk3MjdDMTEuMjE4OCAyMy4zNzk5IDEwLjcxMzkgMjQuMTQyNiA5LjQ1NzAzIDI0LjE0MjZDOC4zMjkxIDI0LjE0MjYgNy43MDYwNSAyMy40MjI5IDcuNjg0NTcgMjIuMzA1N0g0LjI3OTNaTTE2LjcxODggMjIuNzAzMUMxNi43NTEgMjQuODMwMSAxOC4xNjg5IDI3LjI2ODYgMjIuNjQ4NCAyNy4yNjg2QzI2LjQ4MzQgMjcuMjY4NiAyOC42NzQ4IDI1LjM5OTQgMjguNjc0OCAyMi41MzEyQzI4LjY3NDggMTkuNzQ5IDI2Ljc2MjcgMTguNzA3IDI0LjY1NzIgMTguMjk4OEwyMi41MzAzIDE3Ljg2OTFDMjEuMjYyNyAxNy42MjIxIDIwLjU2NDUgMTcuMTM4NyAyMC41NjQ1IDE2LjM1NDVDMjAuNTY0NSAxNS4zODc3IDIxLjM1OTQgMTQuNzMyNCAyMi43NDUxIDE0LjczMjRDMjQuMzI0MiAxNC43MzI0IDI1LjA4NjkgMTUuNjI0IDI1LjE1MTQgMTYuNTM3MUgyOC40Mjc3QzI4LjM5NTUgMTQuMDM0MiAyNi4zMDA4IDEyLjA1NzYgMjIuNzU1OSAxMi4wNTc2QzE5LjQ3OTUgMTIuMDU3NiAxNy4wMzAzIDEzLjcwMTIgMTcuMDMwMyAxNi42MzM4QzE3LjAzMDMgMTkuMTkwNCAxOC43NTk4IDIwLjM3MjEgMjAuOTI5NyAyMC44MDE4TDIzLjAwMjkgMjEuMjUyOUMyNC40MzE2IDIxLjU1MzcgMjUuMTcyOSAyMS45NzI3IDI1LjE3MjkgMjIuODg1N0MyNS4xNzI5IDIzLjg1MjUgMjQuMzk5NCAyNC41NjE1IDIyLjc3NzMgMjQuNTYxNUMyMS4wNjkzIDI0LjU2MTUgMjAuMTY3IDIzLjY1OTIgMjAuMTAyNSAyMi43MDMxSDE2LjcxODhaIiBmaWxsPSJ3aGl0ZSIvPgo8L3N2Zz4K',
}
