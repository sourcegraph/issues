import { RepogroupMetadata } from './types'
import * as React from 'react'
import { SearchPatternType } from '../../../shared/src/graphql-operations'
export const reactHooks: RepogroupMetadata = {
    title: 'React Hooks',
    name: 'react-hooks',
    url: '/react-hooks',
    description:
        "Search popular React Hook repositories from the GitHub repository 'rehooks/awesome-react-hooks'. The search examples show usage of the React Hook useState with various data types.",
    examples: [
        {
            title: 'Find imports of useState with regex search',
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
    homepageDescription: 'Learn how to use React Hooks with search examples.',
    homepageIcon: 'https://cdn4.iconfinder.com/data/icons/logos-3/600/React.js_logo-512.png',
}
