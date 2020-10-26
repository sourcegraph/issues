declare module 'highlightjs-graphql' {
    import { HLJSStatic, IModeBase } from 'highlight.js'

    function hljsDefineGraphQL(hljs: typeof import('highlight.js')): void
    namespace hljsDefineGraphQL {
        export const definer: (hljs?: HLJSStatic) => IModeBase
    }

    export = hljsDefineGraphQL
}
