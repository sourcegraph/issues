import { SearchPatternType } from '../graphql-operations'

export interface ExampleQuery {
    title: string
    description?: string
    /** HTML string to display the example query. */
    exampleQuery: JSX.Element
    /** The raw query string. */
    rawQuery: string
    patternType: SearchPatternType
}

export interface RepogroupMetadata {
    /**
     * The title of the repogroup. This is displayed on the search homepage, and is typically prose. E.g. Refactor python 2 to 3.
     */
    title: string
    /**
     * The name of the repogroup, must match the repogroup name as configured in settings. E.g. python2-to-3.
     */
    name: string
    /**
     * The URL pathname for the repogroup page.
     */
    url: string
    /**
     * A list of example queries using the repogroup. Don't include the `repogroup:name` portion of the query. It will be automatically added.
     */
    examples: ExampleQuery[]
    /**
     * A description of the repogroup to be displayed on the page.
     */
    description: JSX.Element | string
    /**
     * Base64 data uri to an icon.
     */
    homepageIcon: string
    /**
     * A description when displayed on the search homepage.
     */
    homepageDescription: string

    /**
     * Whether to display this in a minimal repogroup page, without examples/repositories/descriptions below the search bar.
     */
    lowProfile?: boolean
}
