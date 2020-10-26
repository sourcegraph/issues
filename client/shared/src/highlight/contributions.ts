import { registerLanguage } from 'highlight.js/lib/core'
import { definer as graphQLLanguage } from 'highlightjs-graphql'

let registered = false

/**
 * Registers syntax highlighters for commonly used languages.
 *
 * This function must be called exactly once. A function is used instead of having the registerLanguage calls be
 * side effects of importing this module to prevent this module from being omitted from production builds due to
 * tree-shaking.
 */
export function registerHighlightContributions(): void {
    if (registered) {
        // Don't double-register these. (There is no way to unregister them.)
        return
    }
    registered = true
    /* eslint-disable @typescript-eslint/no-require-imports */
    /* eslint-disable @typescript-eslint/no-var-requires */
    registerLanguage('go', require('highlight.js/lib/languages/go'))
    registerLanguage('javascript', require('highlight.js/lib/languages/javascript'))
    registerLanguage('typescript', require('highlight.js/lib/languages/typescript'))
    registerLanguage('java', require('highlight.js/lib/languages/java'))
    registerLanguage('python', require('highlight.js/lib/languages/python'))
    registerLanguage('php', require('highlight.js/lib/languages/php'))
    registerLanguage('bash', require('highlight.js/lib/languages/bash'))
    registerLanguage('clojure', require('highlight.js/lib/languages/clojure'))
    // This is a dependency of cpp.
    registerLanguage('c-like', require('highlight.js/lib/languages/c-like'))
    registerLanguage('cpp', require('highlight.js/lib/languages/cpp'))
    registerLanguage('cs', require('highlight.js/lib/languages/csharp'))
    registerLanguage('csharp', require('highlight.js/lib/languages/csharp'))
    registerLanguage('css', require('highlight.js/lib/languages/css'))
    registerLanguage('dockerfile', require('highlight.js/lib/languages/dockerfile'))
    registerLanguage('elixir', require('highlight.js/lib/languages/elixir'))
    registerLanguage('haskell', require('highlight.js/lib/languages/haskell'))
    registerLanguage('html', require('highlight.js/lib/languages/xml'))
    registerLanguage('lua', require('highlight.js/lib/languages/lua'))
    registerLanguage('ocaml', require('highlight.js/lib/languages/ocaml'))
    registerLanguage('r', require('highlight.js/lib/languages/r'))
    registerLanguage('ruby', require('highlight.js/lib/languages/ruby'))
    registerLanguage('rust', require('highlight.js/lib/languages/rust'))
    registerLanguage('swift', require('highlight.js/lib/languages/swift'))
    registerLanguage('markdown', require('highlight.js/lib/languages/markdown'))
    registerLanguage('diff', require('highlight.js/lib/languages/diff'))
    registerLanguage('json', require('highlight.js/lib/languages/json'))
    registerLanguage('jsonc', require('highlight.js/lib/languages/json'))
    registerLanguage('yaml', require('highlight.js/lib/languages/yaml'))
    registerLanguage('kotlin', require('highlight.js/lib/languages/kotlin'))
    registerLanguage('dart', require('highlight.js/lib/languages/dart'))
    registerLanguage('perl', require('highlight.js/lib/languages/perl'))
    registerLanguage('scala', require('highlight.js/lib/languages/scala'))
    registerLanguage('graphql', graphQLLanguage)
    /* eslint-enable @typescript-eslint/no-require-imports */
    /* eslint-enable @typescript-eslint/no-var-requires */
}
