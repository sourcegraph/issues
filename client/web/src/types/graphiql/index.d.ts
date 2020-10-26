declare module 'graphiql' {
    import * as React from 'react'

    export interface GraphQLParams {
        query: string
        variables?: any
    }

    export interface Props {
        /**
         * a function which accepts GraphQL-HTTP parameters and returns a
         * Promise or Observable which resolves to the GraphQL parsed JSON
         * response.
         */
        fetcher: (graphQLParams: GraphQLParams) => Promise<any>

        // TODO: schema

        /**
         * an optional GraphQL string to use as the initial displayed query, if
         * undefined is provided, the stored query or `defaultQuery` will be
         * used.
         */
        query?: string

        /**
         * an optional GraphQL string to use as the initial displayed query
         * variables, if `undefined` is provided, the stored variables will be
         * used.
         */
        variables?: string

        /** an optional name of which GraphQL operation should be executed. */
        operationName?: string

        /**
         * an optional JSON string to use as the initial displayed response. If
         * not provided, no response will be initially shown. You might provide
         * this if illustrating the result of the initial query.
         */
        response?: string

        // TODO: storage

        /**
         * an optional GraphQL string to use when no query is provided and no
         * stored query exists from a previous session. If `undefined` is
         * provided, GraphiQL will use its own default query.
         */
        defaultQuery?: string

        /**
         * an optional function which will be called when the Query editor
         * changes. The argument to the function will be the query string.
         */
        onEditQuery?: (newQuery: string) => void

        /**
         * an optional function which will be called when the Query variable
         * editor changes. The argument to the function will be the variables
         * string.
         */
        onEditVariables?: (newVariables: string) => void

        /**
         * an optional function which will be called when the operation name to
         * be executed changes.
         */
        onEditOperationName?: (newOperationName: string) => void

        /**
         * an optional function which will be called when the docs will be
         * toggled. The argument to the function will be a boolean whether the
         * docs are now open or closed.
         */
        onToggleDocs?: (open: boolean) => void

        // TODO: getDefaultFieldNames

        /**
         * an optional string naming a CodeMirror theme to be applied to the
         * `QueryEditor`, `ResultViewer`, and `Variables` panes. Defaults to
         * the `graphiql` theme.
         *
         * See https://github.com/graphql/graphiql#applying-an-editor-theme
         */
        editorTheme?: string
    }

    export default class GraphiQL extends React.Component<Props> {
        public static Logo: React.ComponentClass<any>
        public static Toolbar: React.ComponentClass<any>
        public static Button: React.ComponentClass<any>
        public static Menu: React.ComponentClass<any>
        public static MenuItem: React.ComponentClass<any>
        public static Select: React.ComponentClass<any>
        public static SelectOption: React.ComponentClass<any>
        public static Group: React.ComponentClass<any>
        public static Footer: React.ComponentClass<any>

        // TODO: ... probably not a part of the public API. Find out how we can
        // upstream a better change for doing this (adding custom elements to
        // the toolbar without losing the existing toolbar logic like toggling
        // the history pane.
        public handlePrettifyQuery: () => void
        public handleToggleHistory: () => void
    }
}
