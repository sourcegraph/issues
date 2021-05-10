/* eslint-disable no-template-curly-in-string */
import { WebGraphQlOperations } from '../graphql-operations'
import { SearchEvent } from '../search/stream'

export const diffSearchStreamEvents: SearchEvent[] = [
    {
        type: 'matches',
        data: [
            {
                type: 'commit',
                label:
                    '[sourcegraph/sourcegraph](/gitlab.sgdev.org/sourcegraph/sourcegraph) › [Rijnard van Tonder](/gitlab.sgdev.org/sourcegraph/sourcegraph/-/commit/b6dd338737c090fdab31d324542bfdaa7ce9f766): [search: if not specified, set fork:no by default (#8739)](/gitlab.sgdev.org/sourcegraph/sourcegraph/-/commit/b6dd338737c090fdab31d324542bfdaa7ce9f766)',
                url: '/gitlab.sgdev.org/sourcegraph/sourcegraph/-/commit/b6dd338737c090fdab31d324542bfdaa7ce9f766',
                detail:
                    '[`b6dd338` one year ago](/gitlab.sgdev.org/sourcegraph/sourcegraph/-/commit/b6dd338737c090fdab31d324542bfdaa7ce9f766)',
                repository: 'gitlab.sgdev.org/sourcegraph/sourcegraph',
                content:
                    "```diff\nweb/src/regression/search.test.ts web/src/regression/search.test.ts\n@@ -434,1 +434,4 @@ describe('Search regression test suite', () =\u003E {\n         })\n+        test('Fork repos excluded by default', async () =\u003E {\n+            const urlQuery = buildSearchURLQuery('type:repo sgtest/mux', GQL.SearchPatternType.regexp, false)\n+            await driver.page.goto(config.sourcegraphBaseUrl + '/search?' + urlQuery)\n@@ -435,0 +439,4 @@ describe('Search regression test suite', () =\u003E {\n+        })\n+        test('Forked repos included by by fork option', async () =\u003E {\n+            const urlQuery = buildSearchURLQuery('type:repo sgtest/mux fork:yes', GQL.SearchPatternType.regexp, false)\n+            await driver.page.goto(config.sourcegraphBaseUrl + '/search?' + urlQuery)\n```",
                ranges: [
                    [-1, 30, 4],
                    [-4, 30, 4],
                    [3, 9, 4],
                    [4, 63, 4],
                    [8, 9, 4],
                    [9, 63, 4],
                ],
            },
        ],
    },
    { type: 'done', data: {} },
]

export const diffHighlightResult: Partial<WebGraphQlOperations> = {
    highlightCode: () => ({
        highlightCode:
            '<table><tbody><tr><td class="line" data-line="1"/><td class="code"><div><span class="hl-source hl-diff">web/src/regression/search.test.ts web/src/regression/search.test.ts\n</span></div></td></tr><tr><td class="line" data-line="2"/><td class="code"><div><span class="hl-source hl-diff"><span class="hl-meta hl-diff hl-range hl-unified"><span class="hl-meta hl-range hl-unified hl-diff"><span class="hl-punctuation hl-definition hl-range hl-diff">@@</span> <span class="hl-meta hl-toc-list hl-line-number hl-diff">-434,1 +434,4</span> <span class="hl-punctuation hl-definition hl-range hl-diff">@@</span> <span class="hl-entity hl-name hl-section hl-diff">describe(&#39;Search regression test suite&#39;, () =&gt; {</span>\n</span></span></span></div></td></tr><tr><td class="line" data-line="3"/><td class="code"><div><span class="hl-source hl-diff">         })\n</span></div></td></tr><tr><td class="line" data-line="4"/><td class="code"><div><span class="hl-source hl-diff"><span class="hl-markup hl-inserted hl-diff"><span class="hl-punctuation hl-definition hl-inserted hl-diff">+</span>        test(&#39;Fork repos excluded by default&#39;, async () =&gt; {\n</span></span></div></td></tr><tr><td class="line" data-line="5"/><td class="code"><div><span class="hl-source hl-diff"><span class="hl-markup hl-inserted hl-diff"><span class="hl-punctuation hl-definition hl-inserted hl-diff">+</span>            const urlQuery = buildSearchURLQuery(&#39;type:repo sgtest/mux&#39;, GQL.SearchPatternType.regexp, false)\n</span></span></div></td></tr><tr><td class="line" data-line="6"/><td class="code"><div><span class="hl-source hl-diff"><span class="hl-markup hl-inserted hl-diff"><span class="hl-punctuation hl-definition hl-inserted hl-diff">+</span>            await driver.page.goto(config.sourcegraphBaseUrl + &#39;/search?&#39; + urlQuery)\n</span></span></div></td></tr><tr><td class="line" data-line="7"/><td class="code"><div><span class="hl-source hl-diff"><span class="hl-meta hl-diff hl-range hl-unified"><span class="hl-meta hl-range hl-unified hl-diff"><span class="hl-punctuation hl-definition hl-range hl-diff">@@</span> <span class="hl-meta hl-toc-list hl-line-number hl-diff">-435,0 +439,4</span> <span class="hl-punctuation hl-definition hl-range hl-diff">@@</span> <span class="hl-entity hl-name hl-section hl-diff">describe(&#39;Search regression test suite&#39;, () =&gt; {</span>\n</span></span></span></div></td></tr><tr><td class="line" data-line="8"/><td class="code"><div><span class="hl-source hl-diff"><span class="hl-markup hl-inserted hl-diff"><span class="hl-punctuation hl-definition hl-inserted hl-diff">+</span>        })\n</span></span></div></td></tr><tr><td class="line" data-line="9"/><td class="code"><div><span class="hl-source hl-diff"><span class="hl-markup hl-inserted hl-diff"><span class="hl-punctuation hl-definition hl-inserted hl-diff">+</span>        test(&#39;Forked repos included by by fork option&#39;, async () =&gt; {\n</span></span></div></td></tr><tr><td class="line" data-line="10"/><td class="code"><div><span class="hl-source hl-diff"><span class="hl-markup hl-inserted hl-diff"><span class="hl-punctuation hl-definition hl-inserted hl-diff">+</span>            const urlQuery = buildSearchURLQuery(&#39;type:repo sgtest/mux fork:yes&#39;, GQL.SearchPatternType.regexp, false)\n</span></span></div></td></tr><tr><td class="line" data-line="11"/><td class="code"><div><span class="hl-source hl-diff"><span class="hl-markup hl-inserted hl-diff"><span class="hl-punctuation hl-definition hl-inserted hl-diff">+</span>            await driver.page.goto(config.sourcegraphBaseUrl + &#39;/search?&#39; + urlQuery)</span></span></div></td></tr></tbody></table>',
    }),
}

export const commitSearchStreamEvents: SearchEvent[] = [
    {
        type: 'matches',
        data: [
            {
                type: 'commit',
                label:
                    '[sourcegraph/sourcegraph](/github.com/sourcegraph/sourcegraph) › [Camden Cheek](/github.com/sourcegraph/sourcegraph/-/commit/f7d28599cad80e200913d9c4612618a73199bac1): [search: Incorporate search blitz (#19567)](/github.com/sourcegraph/sourcegraph/-/commit/f7d28599cad80e200913d9c4612618a73199bac1)',
                url: '/github.com/sourcegraph/sourcegraph/-/commit/f7d28599cad80e200913d9c4612618a73199bac1',
                detail:
                    '[`f7d2859` 2 days ago](/github.com/sourcegraph/sourcegraph/-/commit/f7d28599cad80e200913d9c4612618a73199bac1)',
                repository: 'github.com/sourcegraph/sourcegraph',
                content:
                    '```COMMIT_EDITMSG\nsearch: Incorporate search blitz (#19567)\n\nIncorporates search blitz into sourcegraph/sourcegraph so it has access to the internal streaming client\n```',
                ranges: [
                    [3, 37, 5],
                    [3, 49, 5],
                ],
            },
        ],
    },
    { type: 'done', data: {} },
]

export const commitHighlightResult: Partial<WebGraphQlOperations> = {
    highlightCode: () => ({
        highlightCode:
            '<table><tbody><tr><td class="line" data-line="1"></td><td class="code"><div><span style="color:#657b83;">search: Incorporate search blitz (#19567)\n</span></div></td></tr><tr><td class="line" data-line="2"></td><td class="code"><div><span style="color:#657b83;">\n</span></div></td></tr><tr><td class="line" data-line="3"></td><td class="code"><div><span style="color:#657b83;">Incorporates search blitz into sourcegraph/sourcegraph so it has access to the internal streaming client</span></div></td></tr></tbody></table>',
    }),
}

export const mixedSearchStreamEvents: SearchEvent[] = [
    {
        type: 'matches',
        data: [
            { type: 'repo', repository: 'gitlab.sgdev.org/lg-test-private/lg-test' },
            {
                type: 'file',
                name: 'overridable/bool_or_string_test.go',
                repository: 'gitlab.sgdev.org/aharvey/batch-change-utils',
                branches: [''],
                version: '206c057cc03eea48300a4bd33f4dc4222d242114',
                lineMatches: [],
            },
            {
                type: 'file',
                name: 'src/main.test.ts',
                repository: 'gitlab.sgdev.org/sourcegraph/lsif-cpp',
                branches: [''],
                version: '2e3569cf60646c9ce4e37a43e5cf698a00cbd41a',
                lineMatches: [
                    {
                        line: "test('does not emit items with duplicate IDs', async () =\u003E {",
                        lineNumber: 38,
                        offsetAndLengths: [[0, 4]],
                    },
                    { line: "test('five', async () =\u003E {", lineNumber: 63, offsetAndLengths: [[0, 4]] },
                ],
            },
        ],
    },
    {
        type: 'filters',
        data: [
            { value: 'lang:go', label: 'lang:go', count: 1092, limitHit: false, kind: 'lang' },
            {
                value: '-file:_test\\.go$',
                label: '-file:_test\\.go$',
                count: 663,
                limitHit: false,
                kind: 'file',
            },
            {
                value: 'lang:typescript',
                label: 'lang:typescript',
                count: 379,
                limitHit: false,
                kind: 'lang',
            },
            { value: 'lang:markdown', label: 'lang:markdown', count: 343, limitHit: false, kind: 'lang' },
            { value: 'lang:yaml', label: 'lang:yaml', count: 193, limitHit: false, kind: 'lang' },
            {
                value: 'repo:^gitlab\\.sgdev\\.org/sourcegraph/src-cli$',
                label: 'gitlab.sgdev.org/sourcegraph/src-cli',
                count: 156,
                limitHit: false,
                kind: 'repo',
            },
            {
                value: 'repo:^ghe\\.sgdev\\.org/sourcegraph/gorillalabs-sparkling$',
                label: 'ghe.sgdev.org/sourcegraph/gorillalabs-sparkling',
                count: 145,
                limitHit: false,
                kind: 'repo',
            },
            {
                value: 'repo:^gitlab\\.sgdev\\.org/sourcegraph/java-langserver$',
                label: 'gitlab.sgdev.org/sourcegraph/java-langserver',
                count: 142,
                limitHit: false,
                kind: 'repo',
            },
            {
                value: 'repo:^gitlab\\.sgdev\\.org/sourcegraph/go-jsonschema$',
                label: 'gitlab.sgdev.org/sourcegraph/go-jsonschema',
                count: 130,
                limitHit: false,
                kind: 'repo',
            },
            {
                value: 'repo:^gitlab\\.sgdev\\.org/aharvey/batch-change-utils$',
                label: 'gitlab.sgdev.org/aharvey/batch-change-utils',
                count: 125,
                limitHit: false,
                kind: 'repo',
            },
            {
                value: 'repo:^gitlab\\.sgdev\\.org/sourcegraph/about$',
                label: 'gitlab.sgdev.org/sourcegraph/about',
                count: 125,
                limitHit: false,
                kind: 'repo',
            },
            {
                value: 'repo:^ghe\\.sgdev\\.org/sourcegraph/gorilla-websocket$',
                label: 'ghe.sgdev.org/sourcegraph/gorilla-websocket',
                count: 123,
                limitHit: false,
                kind: 'repo',
            },
            {
                value: 'repo:^github\\.com/hashicorp/go-multierror$',
                label: 'github.com/hashicorp/go-multierror',
                count: 121,
                limitHit: false,
                kind: 'repo',
            },
            {
                value: 'repo:^gitlab\\.sgdev\\.org/sourcegraph/sourcegraph$',
                label: 'gitlab.sgdev.org/sourcegraph/sourcegraph',
                count: 115,
                limitHit: false,
                kind: 'repo',
            },
            {
                value: 'repo:^github\\.com/sourcegraph/sourcegraph$',
                label: 'github.com/sourcegraph/sourcegraph',
                count: 112,
                limitHit: false,
                kind: 'repo',
            },
            {
                value: 'repo:^gitlab\\.sgdev\\.org/aharvey/sourcegraph$',
                label: 'gitlab.sgdev.org/aharvey/sourcegraph',
                count: 109,
                limitHit: false,
                kind: 'repo',
            },
            {
                value: 'repo:^ghe\\.sgdev\\.org/sourcegraph/gorilla-mux$',
                label: 'ghe.sgdev.org/sourcegraph/gorilla-mux',
                count: 108,
                limitHit: false,
                kind: 'repo',
            },
            { value: 'lang:java', label: 'lang:java', count: 95, limitHit: false, kind: 'lang' },
            { value: 'lang:json', label: 'lang:json', count: 77, limitHit: false, kind: 'lang' },
            { value: 'lang:graphql', label: 'lang:graphql', count: 70, limitHit: false, kind: 'lang' },
            { value: 'lang:text', label: 'lang:text', count: 50, limitHit: false, kind: 'lang' },
            { value: 'lang:clojure', label: 'lang:clojure', count: 45, limitHit: false, kind: 'lang' },
            { value: 'lang:css', label: 'lang:css', count: 32, limitHit: false, kind: 'lang' },
        ],
    },
    { type: 'done', data: {} },
]

export const highlightFileResult: Partial<WebGraphQlOperations> = {
    HighlightedFile: () => ({
        repository: {
            commit: {
                file: {
                    isDirectory: false,
                    richHTML: '',
                    highlight: {
                        aborted: false,
                        lineRanges: [
                            [
                                '<tr><td class="line" data-line="1"/><td class="code"><div><span class="hl-source hl-ts"><span class="hl-meta hl-import hl-ts"><span class="hl-keyword hl-control hl-import hl-ts">import</span> <span class="hl-meta hl-block hl-ts"><span class="hl-punctuation hl-definition hl-block hl-ts">{</span> <span class="hl-variable hl-other hl-readwrite hl-alias hl-ts">index</span> <span class="hl-punctuation hl-definition hl-block hl-ts">}</span></span> <span class="hl-keyword hl-control hl-from hl-ts">from</span> <span class="hl-string hl-quoted hl-single hl-ts"><span class="hl-punctuation hl-definition hl-string hl-begin hl-ts">&#39;</span>./index<span class="hl-punctuation hl-definition hl-string hl-end hl-ts">&#39;</span></span></span>\n</span></div></td></tr>',
                                '<tr><td class="line" data-line="2"/><td class="code"><div><span class="hl-source hl-ts"><span class="hl-meta hl-import hl-ts"><span class="hl-keyword hl-control hl-import hl-ts">import</span> <span class="hl-meta hl-block hl-ts"><span class="hl-punctuation hl-definition hl-block hl-ts">{</span> <span class="hl-variable hl-other hl-readwrite hl-alias hl-ts">Edge</span><span class="hl-punctuation hl-separator hl-comma hl-ts">,</span> <span class="hl-variable hl-other hl-readwrite hl-alias hl-ts">Vertex</span> <span class="hl-punctuation hl-definition hl-block hl-ts">}</span></span> <span class="hl-keyword hl-control hl-from hl-ts">from</span> <span class="hl-string hl-quoted hl-single hl-ts"><span class="hl-punctuation hl-definition hl-string hl-begin hl-ts">&#39;</span>lsif-protocol<span class="hl-punctuation hl-definition hl-string hl-end hl-ts">&#39;</span></span></span>\n</span></div></td></tr>',
                                '<tr><td class="line" data-line="3"/><td class="code"><div><span class="hl-source hl-ts"><span class="hl-meta hl-import hl-ts"><span class="hl-keyword hl-control hl-import hl-ts">import</span> <span class="hl-variable hl-other hl-readwrite hl-alias hl-ts">_</span> <span class="hl-keyword hl-control hl-from hl-ts">from</span> <span class="hl-string hl-quoted hl-single hl-ts"><span class="hl-punctuation hl-definition hl-string hl-begin hl-ts">&#39;</span>lodash<span class="hl-punctuation hl-definition hl-string hl-end hl-ts">&#39;</span></span></span>\n</span></div></td></tr>',
                                '<tr><td class="line" data-line="4"/><td class="code"><div><span class="hl-source hl-ts"><span class="hl-meta hl-import hl-ts"><span class="hl-keyword hl-control hl-import hl-ts">import</span> <span class="hl-constant hl-language hl-import-export-all hl-ts">*</span> <span class="hl-keyword hl-control hl-as hl-ts">as</span> <span class="hl-variable hl-other hl-readwrite hl-alias hl-ts">path</span> <span class="hl-keyword hl-control hl-from hl-ts">from</span> <span class="hl-string hl-quoted hl-single hl-ts"><span class="hl-punctuation hl-definition hl-string hl-begin hl-ts">&#39;</span>path<span class="hl-punctuation hl-definition hl-string hl-end hl-ts">&#39;</span></span></span>\n</span></div></td></tr>',
                                '<tr><td class="line" data-line="5"/><td class="code"><div><span class="hl-source hl-ts"><span class="hl-meta hl-import hl-ts"><span class="hl-keyword hl-control hl-import hl-ts">import</span> <span class="hl-constant hl-language hl-import-export-all hl-ts">*</span> <span class="hl-keyword hl-control hl-as hl-ts">as</span> <span class="hl-variable hl-other hl-readwrite hl-alias hl-ts">cp</span> <span class="hl-keyword hl-control hl-from hl-ts">from</span> <span class="hl-string hl-quoted hl-single hl-ts"><span class="hl-punctuation hl-definition hl-string hl-begin hl-ts">&#39;</span>child_process<span class="hl-punctuation hl-definition hl-string hl-end hl-ts">&#39;</span></span></span>\n</span></div></td></tr>',
                                '<tr><td class="line" data-line="6"/><td class="code"><div><span class="hl-source hl-ts">\n</span></div></td></tr>',
                                '<tr><td class="line" data-line="7"/><td class="code"><div><span class="hl-source hl-ts"><span class="hl-meta hl-var hl-expr hl-ts"><span class="hl-storage hl-type hl-ts">const</span> <span class="hl-meta hl-var-single-variable hl-expr hl-ts"><span class="hl-meta hl-definition hl-variable hl-ts"><span class="hl-variable hl-other hl-constant hl-ts">GENERATE</span></span> </span><span class="hl-keyword hl-operator hl-assignment hl-ts">=</span> <span class="hl-constant hl-language hl-boolean hl-false hl-ts">false</span></span>\n</span></div></td></tr>',
                                '<tr><td class="line" data-line="8"/><td class="code"><div><span class="hl-source hl-ts">\n</span></div></td></tr>',
                                '<tr><td class="line" data-line="9"/><td class="code"><div><span class="hl-source hl-ts"><span class="hl-meta hl-function hl-ts"><span class="hl-storage hl-type hl-function hl-ts">function</span> <span class="hl-meta hl-definition hl-function hl-ts"><span class="hl-entity hl-name hl-function hl-ts">generate</span></span><span class="hl-meta hl-parameters hl-ts"><span class="hl-punctuation hl-definition hl-parameters hl-begin hl-ts">(</span><span class="hl-variable hl-parameter hl-ts">example</span><span class="hl-meta hl-type hl-annotation hl-ts"><span class="hl-keyword hl-operator hl-type hl-annotation hl-ts">:</span> <span class="hl-support hl-type hl-primitive hl-ts">string</span></span><span class="hl-punctuation hl-definition hl-parameters hl-end hl-ts">)</span></span><span class="hl-meta hl-return hl-type hl-ts"><span class="hl-keyword hl-operator hl-type hl-annotation hl-ts">:</span> <span class="hl-support hl-type hl-primitive hl-ts">void</span> </span><span class="hl-meta hl-block hl-ts"><span class="hl-punctuation hl-definition hl-block hl-ts">{</span>\n</span></span></span></div></td></tr>',
                                '<tr><td class="line" data-line="10"/><td class="code"><div><span class="hl-source hl-ts"><span class="hl-meta hl-function hl-ts"><span class="hl-meta hl-block hl-ts">    <span class="hl-meta hl-function-call hl-ts"><span class="hl-variable hl-other hl-object hl-ts">cp</span><span class="hl-punctuation hl-accessor hl-ts">.</span><span class="hl-entity hl-name hl-function hl-ts">execFileSync</span></span><span class="hl-meta hl-brace hl-round hl-ts">(</span><span class="hl-string hl-quoted hl-single hl-ts"><span class="hl-punctuation hl-definition hl-string hl-begin hl-ts">&#39;</span>./generate-csv<span class="hl-punctuation hl-definition hl-string hl-end hl-ts">&#39;</span></span><span class="hl-punctuation hl-separator hl-comma hl-ts">,</span><span class="hl-meta hl-array hl-literal hl-ts"> <span class="hl-meta hl-brace hl-square hl-ts">[</span><span class="hl-string hl-quoted hl-single hl-ts"><span class="hl-punctuation hl-definition hl-string hl-begin hl-ts">&#39;</span>$CXX -c *.cpp<span class="hl-punctuation hl-definition hl-string hl-end hl-ts">&#39;</span></span><span class="hl-meta hl-brace hl-square hl-ts">]</span></span><span class="hl-punctuation hl-separator hl-comma hl-ts">,</span> <span class="hl-meta hl-objectliteral hl-ts"><span class="hl-punctuation hl-definition hl-block hl-ts">{</span>\n</span></span></span></span></div></td></tr>',
                                '<tr><td class="line" data-line="11"/><td class="code"><div><span class="hl-source hl-ts"><span class="hl-meta hl-function hl-ts"><span class="hl-meta hl-block hl-ts"><span class="hl-meta hl-objectliteral hl-ts">        <span class="hl-meta hl-object hl-member hl-ts"><span class="hl-meta hl-object-literal hl-key hl-ts">env</span></span><span class="hl-meta hl-object hl-member hl-ts"><span class="hl-meta hl-object-literal hl-key hl-ts"><span class="hl-punctuation hl-separator hl-key-value hl-ts">:</span></span> <span class="hl-meta hl-objectliteral hl-ts"><span class="hl-punctuation hl-definition hl-block hl-ts">{</span>\n</span></span></span></span></span></span></div></td></tr>',
                                '<tr><td class="line" data-line="12"/><td class="code"><div><span class="hl-source hl-ts"><span class="hl-meta hl-function hl-ts"><span class="hl-meta hl-block hl-ts"><span class="hl-meta hl-objectliteral hl-ts"><span class="hl-meta hl-object hl-member hl-ts"><span class="hl-meta hl-objectliteral hl-ts">            <span class="hl-meta hl-object hl-member hl-ts"><span class="hl-meta hl-object-literal hl-key hl-ts">ABSROOTDIR</span></span><span class="hl-meta hl-object hl-member hl-ts"><span class="hl-meta hl-object-literal hl-key hl-ts"><span class="hl-punctuation hl-separator hl-key-value hl-ts">:</span></span> <span class="hl-meta hl-function-call hl-ts"><span class="hl-variable hl-other hl-object hl-ts">path</span><span class="hl-punctuation hl-accessor hl-ts">.</span><span class="hl-support hl-function hl-promise hl-ts">resolve</span></span><span class="hl-meta hl-brace hl-round hl-ts">(</span><span class="hl-string hl-template hl-ts"><span class="hl-punctuation hl-definition hl-string hl-template hl-begin hl-ts">`</span>examples/<span class="hl-meta hl-template hl-expression hl-ts"><span class="hl-punctuation hl-definition hl-template-expression hl-begin hl-ts">${</span></span><span class="hl-meta hl-template hl-expression hl-ts"><span class="hl-meta hl-embedded hl-line hl-ts"><span class="hl-variable hl-other hl-readwrite hl-ts">example</span></span><span class="hl-punctuation hl-definition hl-template-expression hl-end hl-ts">}</span></span>/root<span class="hl-punctuation hl-definition hl-string hl-template hl-end hl-ts">`</span></span><span class="hl-meta hl-brace hl-round hl-ts">)</span></span><span class="hl-punctuation hl-separator hl-comma hl-ts">,</span>\n</span></span></span></span></span></span></div></td></tr>',
                                '<tr><td class="line" data-line="13"/><td class="code"><div><span class="hl-source hl-ts"><span class="hl-meta hl-function hl-ts"><span class="hl-meta hl-block hl-ts"><span class="hl-meta hl-objectliteral hl-ts"><span class="hl-meta hl-object hl-member hl-ts"><span class="hl-meta hl-objectliteral hl-ts">            <span class="hl-meta hl-object hl-member hl-ts"><span class="hl-meta hl-object-literal hl-key hl-ts">ABSOUTDIR</span></span><span class="hl-meta hl-object hl-member hl-ts"><span class="hl-meta hl-object-literal hl-key hl-ts"><span class="hl-punctuation hl-separator hl-key-value hl-ts">:</span></span> <span class="hl-meta hl-function-call hl-ts"><span class="hl-variable hl-other hl-object hl-ts">path</span><span class="hl-punctuation hl-accessor hl-ts">.</span><span class="hl-support hl-function hl-promise hl-ts">resolve</span></span><span class="hl-meta hl-brace hl-round hl-ts">(</span><span class="hl-string hl-template hl-ts"><span class="hl-punctuation hl-definition hl-string hl-template hl-begin hl-ts">`</span>examples/<span class="hl-meta hl-template hl-expression hl-ts"><span class="hl-punctuation hl-definition hl-template-expression hl-begin hl-ts">${</span></span><span class="hl-meta hl-template hl-expression hl-ts"><span class="hl-meta hl-embedded hl-line hl-ts"><span class="hl-variable hl-other hl-readwrite hl-ts">example</span></span><span class="hl-punctuation hl-definition hl-template-expression hl-end hl-ts">}</span></span>/output<span class="hl-punctuation hl-definition hl-string hl-template hl-end hl-ts">`</span></span><span class="hl-meta hl-brace hl-round hl-ts">)</span></span><span class="hl-punctuation hl-separator hl-comma hl-ts">,</span>\n</span></span></span></span></span></span></div></td></tr>',
                                '<tr><td class="line" data-line="14"/><td class="code"><div><span class="hl-source hl-ts"><span class="hl-meta hl-function hl-ts"><span class="hl-meta hl-block hl-ts"><span class="hl-meta hl-objectliteral hl-ts"><span class="hl-meta hl-object hl-member hl-ts"><span class="hl-meta hl-objectliteral hl-ts">            <span class="hl-meta hl-object hl-member hl-ts"><span class="hl-meta hl-object-literal hl-key hl-ts">CLEAN</span></span><span class="hl-meta hl-object hl-member hl-ts"><span class="hl-meta hl-object-literal hl-key hl-ts"><span class="hl-punctuation hl-separator hl-key-value hl-ts">:</span></span> <span class="hl-string hl-quoted hl-single hl-ts"><span class="hl-punctuation hl-definition hl-string hl-begin hl-ts">&#39;</span>true<span class="hl-punctuation hl-definition hl-string hl-end hl-ts">&#39;</span></span></span><span class="hl-punctuation hl-separator hl-comma hl-ts">,</span>\n</span></span></span></span></span></span></div></td></tr>',
                                '<tr><td class="line" data-line="15"/><td class="code"><div><span class="hl-source hl-ts"><span class="hl-meta hl-function hl-ts"><span class="hl-meta hl-block hl-ts"><span class="hl-meta hl-objectliteral hl-ts"><span class="hl-meta hl-object hl-member hl-ts"><span class="hl-meta hl-objectliteral hl-ts">        <span class="hl-punctuation hl-definition hl-block hl-ts">}</span></span></span><span class="hl-punctuation hl-separator hl-comma hl-ts">,</span>\n</span></span></span></span></div></td></tr>',
                                '<tr><td class="line" data-line="16"/><td class="code"><div><span class="hl-source hl-ts"><span class="hl-meta hl-function hl-ts"><span class="hl-meta hl-block hl-ts"><span class="hl-meta hl-objectliteral hl-ts">    <span class="hl-punctuation hl-definition hl-block hl-ts">}</span></span><span class="hl-meta hl-brace hl-round hl-ts">)</span>\n</span></span></span></div></td></tr>',
                                '<tr><td class="line" data-line="17"/><td class="code"><div><span class="hl-source hl-ts"><span class="hl-meta hl-function hl-ts"><span class="hl-meta hl-block hl-ts"><span class="hl-punctuation hl-definition hl-block hl-ts">}</span></span></span>\n</span></div></td></tr>',
                                '<tr><td class="line" data-line="18"/><td class="code"><div><span class="hl-source hl-ts">\n</span></div></td></tr>',
                                '<tr><td class="line" data-line="19"/><td class="code"><div><span class="hl-source hl-ts"><span class="hl-meta hl-function hl-ts"><span class="hl-storage hl-modifier hl-async hl-ts">async</span> <span class="hl-storage hl-type hl-function hl-ts">function</span> <span class="hl-meta hl-definition hl-function hl-ts"><span class="hl-entity hl-name hl-function hl-ts">indexExample</span></span><span class="hl-meta hl-parameters hl-ts"><span class="hl-punctuation hl-definition hl-parameters hl-begin hl-ts">(</span><span class="hl-variable hl-parameter hl-ts">example</span><span class="hl-meta hl-type hl-annotation hl-ts"><span class="hl-keyword hl-operator hl-type hl-annotation hl-ts">:</span> <span class="hl-support hl-type hl-primitive hl-ts">string</span></span><span class="hl-punctuation hl-definition hl-parameters hl-end hl-ts">)</span></span><span class="hl-meta hl-return hl-type hl-ts"><span class="hl-keyword hl-operator hl-type hl-annotation hl-ts">:</span> <span class="hl-entity hl-name hl-type hl-ts">Promise</span><span class="hl-meta hl-type hl-parameters hl-ts"><span class="hl-punctuation hl-definition hl-typeparameters hl-begin hl-ts">&lt;</span></span><span class="hl-meta hl-type hl-parameters hl-ts"><span class="hl-meta hl-type hl-paren hl-cover hl-ts"><span class="hl-meta hl-brace hl-round hl-ts">(</span><span class="hl-entity hl-name hl-type hl-ts">Edge</span> <span class="hl-keyword hl-operator hl-type hl-ts">|</span> <span class="hl-entity hl-name hl-type hl-ts">Vertex</span><span class="hl-meta hl-brace hl-round hl-ts">)</span></span><span class="hl-meta hl-type hl-tuple hl-ts"><span class="hl-meta hl-brace hl-square hl-ts">[</span><span class="hl-meta hl-brace hl-square hl-ts">]</span></span></span><span class="hl-meta hl-type hl-parameters hl-ts"><span class="hl-punctuation hl-definition hl-typeparameters hl-end hl-ts">&gt;</span></span> </span><span class="hl-meta hl-block hl-ts"><span class="hl-punctuation hl-definition hl-block hl-ts">{</span>\n</span></span></span></div></td></tr>',
                                '<tr><td class="line" data-line="20"/><td class="code"><div><span class="hl-source hl-ts"><span class="hl-meta hl-function hl-ts"><span class="hl-meta hl-block hl-ts">    <span class="hl-keyword hl-control hl-conditional hl-ts">if</span> <span class="hl-meta hl-brace hl-round hl-ts">(</span><span class="hl-variable hl-other hl-constant hl-ts">GENERATE</span><span class="hl-meta hl-brace hl-round hl-ts">)</span> <span class="hl-meta hl-block hl-ts"><span class="hl-punctuation hl-definition hl-block hl-ts">{</span>\n</span></span></span></span></div></td></tr>',
                                '<tr><td class="line" data-line="21"/><td class="code"><div><span class="hl-source hl-ts"><span class="hl-meta hl-function hl-ts"><span class="hl-meta hl-block hl-ts"><span class="hl-meta hl-block hl-ts">        <span class="hl-meta hl-function-call hl-ts"><span class="hl-entity hl-name hl-function hl-ts">generate</span></span><span class="hl-meta hl-brace hl-round hl-ts">(</span><span class="hl-variable hl-other hl-readwrite hl-ts">example</span><span class="hl-meta hl-brace hl-round hl-ts">)</span>\n</span></span></span></span></div></td></tr>',
                                '<tr><td class="line" data-line="22"/><td class="code"><div><span class="hl-source hl-ts"><span class="hl-meta hl-function hl-ts"><span class="hl-meta hl-block hl-ts"><span class="hl-meta hl-block hl-ts">    <span class="hl-punctuation hl-definition hl-block hl-ts">}</span></span>\n</span></span></span></div></td></tr>',
                                '<tr><td class="line" data-line="23"/><td class="code"><div><span class="hl-source hl-ts"><span class="hl-meta hl-function hl-ts"><span class="hl-meta hl-block hl-ts">\n</span></span></span></div></td></tr>',
                                '<tr><td class="line" data-line="24"/><td class="code"><div><span class="hl-source hl-ts"><span class="hl-meta hl-function hl-ts"><span class="hl-meta hl-block hl-ts">    <span class="hl-meta hl-var hl-expr hl-ts"><span class="hl-storage hl-type hl-ts">const</span> <span class="hl-meta hl-var-single-variable hl-expr hl-ts"><span class="hl-meta hl-definition hl-variable hl-ts"><span class="hl-variable hl-other hl-constant hl-ts">output</span></span><span class="hl-meta hl-type hl-annotation hl-ts"><span class="hl-keyword hl-operator hl-type hl-annotation hl-ts">:</span> <span class="hl-meta hl-type hl-paren hl-cover hl-ts"><span class="hl-meta hl-brace hl-round hl-ts">(</span><span class="hl-entity hl-name hl-type hl-ts">Edge</span> <span class="hl-keyword hl-operator hl-type hl-ts">|</span> <span class="hl-entity hl-name hl-type hl-ts">Vertex</span><span class="hl-meta hl-brace hl-round hl-ts">)</span></span><span class="hl-meta hl-type hl-tuple hl-ts"><span class="hl-meta hl-brace hl-square hl-ts">[</span><span class="hl-meta hl-brace hl-square hl-ts">]</span></span> </span></span><span class="hl-keyword hl-operator hl-assignment hl-ts">=</span><span class="hl-meta hl-array hl-literal hl-ts"> <span class="hl-meta hl-brace hl-square hl-ts">[</span><span class="hl-meta hl-brace hl-square hl-ts">]</span></span></span>\n</span></span></span></div></td></tr>',
                                '<tr><td class="line" data-line="25"/><td class="code"><div><span class="hl-source hl-ts"><span class="hl-meta hl-function hl-ts"><span class="hl-meta hl-block hl-ts">\n</span></span></span></div></td></tr>',
                                '<tr><td class="line" data-line="26"/><td class="code"><div><span class="hl-source hl-ts"><span class="hl-meta hl-function hl-ts"><span class="hl-meta hl-block hl-ts">    <span class="hl-keyword hl-control hl-flow hl-ts">await</span> <span class="hl-meta hl-function-call hl-ts"><span class="hl-entity hl-name hl-function hl-ts">index</span></span><span class="hl-meta hl-brace hl-round hl-ts">(</span><span class="hl-meta hl-objectliteral hl-ts"><span class="hl-punctuation hl-definition hl-block hl-ts">{</span>\n</span></span></span></span></div></td></tr>',
                                '<tr><td class="line" data-line="27"/><td class="code"><div><span class="hl-source hl-ts"><span class="hl-meta hl-function hl-ts"><span class="hl-meta hl-block hl-ts"><span class="hl-meta hl-objectliteral hl-ts">        <span class="hl-meta hl-object hl-member hl-ts"><span class="hl-meta hl-object-literal hl-key hl-ts">csvFileGlob</span></span><span class="hl-meta hl-object hl-member hl-ts"><span class="hl-meta hl-object-literal hl-key hl-ts"><span class="hl-punctuation hl-separator hl-key-value hl-ts">:</span></span> <span class="hl-string hl-template hl-ts"><span class="hl-punctuation hl-definition hl-string hl-template hl-begin hl-ts">`</span>examples/<span class="hl-meta hl-template hl-expression hl-ts"><span class="hl-punctuation hl-definition hl-template-expression hl-begin hl-ts">${</span></span><span class="hl-meta hl-template hl-expression hl-ts"><span class="hl-meta hl-embedded hl-line hl-ts"><span class="hl-variable hl-other hl-readwrite hl-ts">example</span></span><span class="hl-punctuation hl-definition hl-template-expression hl-end hl-ts">}</span></span>/output/*.csv<span class="hl-punctuation hl-definition hl-string hl-template hl-end hl-ts">`</span></span></span><span class="hl-punctuation hl-separator hl-comma hl-ts">,</span>\n</span></span></span></span></div></td></tr>',
                                '<tr><td class="line" data-line="28"/><td class="code"><div><span class="hl-source hl-ts"><span class="hl-meta hl-function hl-ts"><span class="hl-meta hl-block hl-ts"><span class="hl-meta hl-objectliteral hl-ts">        <span class="hl-meta hl-object hl-member hl-ts"><span class="hl-meta hl-object-literal hl-key hl-ts">root</span></span><span class="hl-meta hl-object hl-member hl-ts"><span class="hl-meta hl-object-literal hl-key hl-ts"><span class="hl-punctuation hl-separator hl-key-value hl-ts">:</span></span> <span class="hl-string hl-template hl-ts"><span class="hl-punctuation hl-definition hl-string hl-template hl-begin hl-ts">`</span>examples/<span class="hl-meta hl-template hl-expression hl-ts"><span class="hl-punctuation hl-definition hl-template-expression hl-begin hl-ts">${</span></span><span class="hl-meta hl-template hl-expression hl-ts"><span class="hl-meta hl-embedded hl-line hl-ts"><span class="hl-variable hl-other hl-readwrite hl-ts">example</span></span><span class="hl-punctuation hl-definition hl-template-expression hl-end hl-ts">}</span></span>/root<span class="hl-punctuation hl-definition hl-string hl-template hl-end hl-ts">`</span></span></span><span class="hl-punctuation hl-separator hl-comma hl-ts">,</span>\n</span></span></span></span></div></td></tr>',
                                '<tr><td class="line" data-line="29"/><td class="code"><div><span class="hl-source hl-ts"><span class="hl-meta hl-function hl-ts"><span class="hl-meta hl-block hl-ts"><span class="hl-meta hl-objectliteral hl-ts">        <span class="hl-meta hl-object hl-member hl-ts"><span class="hl-meta hl-object-literal hl-key hl-ts"><span class="hl-entity hl-name hl-function hl-ts">emit</span></span></span><span class="hl-meta hl-object hl-member hl-ts"><span class="hl-meta hl-object-literal hl-key hl-ts"><span class="hl-punctuation hl-separator hl-key-value hl-ts">:</span></span> <span class="hl-meta hl-arrow hl-ts"><span class="hl-variable hl-parameter hl-ts">item</span> </span><span class="hl-meta hl-arrow hl-ts"><span class="hl-storage hl-type hl-function hl-arrow hl-ts">=&gt;</span>\n</span></span></span></span></span></span></div></td></tr>',
                                '<tr><td class="line" data-line="30"/><td class="code"><div><span class="hl-source hl-ts"><span class="hl-meta hl-function hl-ts"><span class="hl-meta hl-block hl-ts"><span class="hl-meta hl-objectliteral hl-ts"><span class="hl-meta hl-object hl-member hl-ts"><span class="hl-meta hl-arrow hl-ts">            </span><span class="hl-new hl-expr hl-ts"><span class="hl-keyword hl-operator hl-new hl-ts">new</span> <span class="hl-entity hl-name hl-type hl-ts">Promise</span><span class="hl-meta hl-brace hl-round hl-ts">(</span><span class="hl-meta hl-arrow hl-ts"><span class="hl-variable hl-parameter hl-ts">resolve</span> </span><span class="hl-meta hl-arrow hl-ts"><span class="hl-storage hl-type hl-function hl-arrow hl-ts">=&gt;</span> <span class="hl-meta hl-block hl-ts"><span class="hl-punctuation hl-definition hl-block hl-ts">{</span>\n</span></span></span></span></span></span></span></span></div></td></tr>',
                                '<tr><td class="line" data-line="31"/><td class="code"><div><span class="hl-source hl-ts"><span class="hl-meta hl-function hl-ts"><span class="hl-meta hl-block hl-ts"><span class="hl-meta hl-objectliteral hl-ts"><span class="hl-meta hl-object hl-member hl-ts"><span class="hl-new hl-expr hl-ts"><span class="hl-meta hl-arrow hl-ts"><span class="hl-meta hl-block hl-ts">                <span class="hl-meta hl-function-call hl-ts"><span class="hl-variable hl-other hl-object hl-ts">output</span><span class="hl-punctuation hl-accessor hl-ts">.</span><span class="hl-support hl-function hl-ts">push</span></span><span class="hl-meta hl-brace hl-round hl-ts">(</span><span class="hl-variable hl-other hl-readwrite hl-ts">item</span><span class="hl-meta hl-brace hl-round hl-ts">)</span>\n</span></span></span></span></span></span></span></span></div></td></tr>',
                                '<tr><td class="line" data-line="32"/><td class="code"><div><span class="hl-source hl-ts"><span class="hl-meta hl-function hl-ts"><span class="hl-meta hl-block hl-ts"><span class="hl-meta hl-objectliteral hl-ts"><span class="hl-meta hl-object hl-member hl-ts"><span class="hl-new hl-expr hl-ts"><span class="hl-meta hl-arrow hl-ts"><span class="hl-meta hl-block hl-ts">                <span class="hl-meta hl-function-call hl-ts"><span class="hl-entity hl-name hl-function hl-ts">resolve</span></span><span class="hl-meta hl-brace hl-round hl-ts">(</span><span class="hl-meta hl-brace hl-round hl-ts">)</span>\n</span></span></span></span></span></span></span></span></div></td></tr>',
                                '<tr><td class="line" data-line="33"/><td class="code"><div><span class="hl-source hl-ts"><span class="hl-meta hl-function hl-ts"><span class="hl-meta hl-block hl-ts"><span class="hl-meta hl-objectliteral hl-ts"><span class="hl-meta hl-object hl-member hl-ts"><span class="hl-new hl-expr hl-ts"><span class="hl-meta hl-arrow hl-ts"><span class="hl-meta hl-block hl-ts">            <span class="hl-punctuation hl-definition hl-block hl-ts">}</span></span></span><span class="hl-meta hl-brace hl-round hl-ts">)</span></span></span><span class="hl-punctuation hl-separator hl-comma hl-ts">,</span>\n</span></span></span></span></div></td></tr>',
                                '<tr><td class="line" data-line="34"/><td class="code"><div><span class="hl-source hl-ts"><span class="hl-meta hl-function hl-ts"><span class="hl-meta hl-block hl-ts"><span class="hl-meta hl-objectliteral hl-ts">    <span class="hl-punctuation hl-definition hl-block hl-ts">}</span></span><span class="hl-meta hl-brace hl-round hl-ts">)</span>\n</span></span></span></div></td></tr>',
                                '<tr><td class="line" data-line="35"/><td class="code"><div><span class="hl-source hl-ts"><span class="hl-meta hl-function hl-ts"><span class="hl-meta hl-block hl-ts">\n</span></span></span></div></td></tr>',
                                '<tr><td class="line" data-line="36"/><td class="code"><div><span class="hl-source hl-ts"><span class="hl-meta hl-function hl-ts"><span class="hl-meta hl-block hl-ts">    <span class="hl-keyword hl-control hl-flow hl-ts">return</span> <span class="hl-variable hl-other hl-readwrite hl-ts">output</span>\n</span></span></span></div></td></tr>',
                                '<tr><td class="line" data-line="37"/><td class="code"><div><span class="hl-source hl-ts"><span class="hl-meta hl-function hl-ts"><span class="hl-meta hl-block hl-ts"><span class="hl-punctuation hl-definition hl-block hl-ts">}</span></span></span>\n</span></div></td></tr>',
                                '<tr><td class="line" data-line="38"/><td class="code"><div><span class="hl-source hl-ts">\n</span></div></td></tr>',
                                '<tr><td class="line" data-line="39"/><td class="code"><div><span class="hl-source hl-ts"><span class="hl-meta hl-function-call hl-ts"><span class="hl-entity hl-name hl-function hl-ts">test</span></span><span class="hl-meta hl-brace hl-round hl-ts">(</span><span class="hl-string hl-quoted hl-single hl-ts"><span class="hl-punctuation hl-definition hl-string hl-begin hl-ts">&#39;</span>does not emit items with duplicate IDs<span class="hl-punctuation hl-definition hl-string hl-end hl-ts">&#39;</span></span><span class="hl-punctuation hl-separator hl-comma hl-ts">,</span> <span class="hl-meta hl-arrow hl-ts"><span class="hl-storage hl-modifier hl-async hl-ts">async</span> <span class="hl-meta hl-parameters hl-ts"><span class="hl-punctuation hl-definition hl-parameters hl-begin hl-ts">(</span><span class="hl-punctuation hl-definition hl-parameters hl-end hl-ts">)</span></span> </span><span class="hl-meta hl-arrow hl-ts"><span class="hl-storage hl-type hl-function hl-arrow hl-ts">=&gt;</span> <span class="hl-meta hl-block hl-ts"><span class="hl-punctuation hl-definition hl-block hl-ts">{</span>\n</span></span></span></div></td></tr>',
                                '<tr><td class="line" data-line="40"/><td class="code"><div><span class="hl-source hl-ts"><span class="hl-meta hl-arrow hl-ts"><span class="hl-meta hl-block hl-ts">    <span class="hl-meta hl-var hl-expr hl-ts"><span class="hl-storage hl-type hl-ts">const</span> <span class="hl-meta hl-var-single-variable hl-expr hl-ts"><span class="hl-meta hl-definition hl-variable hl-ts"><span class="hl-variable hl-other hl-constant hl-ts">output</span></span> </span><span class="hl-keyword hl-operator hl-assignment hl-ts">=</span> <span class="hl-keyword hl-control hl-flow hl-ts">await</span> <span class="hl-meta hl-function-call hl-ts"><span class="hl-entity hl-name hl-function hl-ts">indexExample</span></span><span class="hl-meta hl-brace hl-round hl-ts">(</span><span class="hl-string hl-quoted hl-single hl-ts"><span class="hl-punctuation hl-definition hl-string hl-begin hl-ts">&#39;</span>five<span class="hl-punctuation hl-definition hl-string hl-end hl-ts">&#39;</span></span><span class="hl-meta hl-brace hl-round hl-ts">)</span></span>\n</span></span></span></div></td></tr>',
                                '<tr><td class="line" data-line="41"/><td class="code"><div><span class="hl-source hl-ts"><span class="hl-meta hl-arrow hl-ts"><span class="hl-meta hl-block hl-ts">\n</span></span></span></div></td></tr>',
                                '<tr><td class="line" data-line="42"/><td class="code"><div><span class="hl-source hl-ts"><span class="hl-meta hl-arrow hl-ts"><span class="hl-meta hl-block hl-ts">    <span class="hl-meta hl-var hl-expr hl-ts"><span class="hl-storage hl-type hl-ts">const</span> <span class="hl-meta hl-var-single-variable hl-expr hl-ts"><span class="hl-meta hl-definition hl-variable hl-ts"><span class="hl-variable hl-other hl-constant hl-ts">setsOfDupes</span></span> </span><span class="hl-keyword hl-operator hl-assignment hl-ts">=</span> <span class="hl-meta hl-function-call hl-ts"><span class="hl-entity hl-name hl-function hl-ts">_</span></span><span class="hl-meta hl-brace hl-round hl-ts">(</span><span class="hl-variable hl-other hl-readwrite hl-ts">output</span><span class="hl-meta hl-brace hl-round hl-ts">)</span></span>\n</span></span></span></div></td></tr>',
                                '<tr><td class="line" data-line="43"/><td class="code"><div><span class="hl-source hl-ts"><span class="hl-meta hl-arrow hl-ts"><span class="hl-meta hl-block hl-ts">        <span class="hl-meta hl-function-call hl-ts"><span class="hl-punctuation hl-accessor hl-ts">.</span><span class="hl-entity hl-name hl-function hl-ts">groupBy</span></span><span class="hl-meta hl-brace hl-round hl-ts">(</span><span class="hl-meta hl-arrow hl-ts"><span class="hl-variable hl-parameter hl-ts">item</span> </span><span class="hl-meta hl-arrow hl-ts"><span class="hl-storage hl-type hl-function hl-arrow hl-ts">=&gt;</span> </span><span class="hl-variable hl-other hl-object hl-ts">item</span><span class="hl-punctuation hl-accessor hl-ts">.</span><span class="hl-support hl-variable hl-property hl-dom hl-ts">id</span><span class="hl-meta hl-brace hl-round hl-ts">)</span>\n</span></span></span></div></td></tr>',
                                '<tr><td class="line" data-line="44"/><td class="code"><div><span class="hl-source hl-ts"><span class="hl-meta hl-arrow hl-ts"><span class="hl-meta hl-block hl-ts">        <span class="hl-meta hl-function-call hl-ts"><span class="hl-punctuation hl-accessor hl-ts">.</span><span class="hl-support hl-function hl-dom hl-ts">values</span></span><span class="hl-meta hl-brace hl-round hl-ts">(</span><span class="hl-meta hl-brace hl-round hl-ts">)</span>\n</span></span></span></div></td></tr>',
                                '<tr><td class="line" data-line="45"/><td class="code"><div><span class="hl-source hl-ts"><span class="hl-meta hl-arrow hl-ts"><span class="hl-meta hl-block hl-ts">        <span class="hl-meta hl-function-call hl-ts"><span class="hl-punctuation hl-accessor hl-ts">.</span><span class="hl-entity hl-name hl-function hl-ts">map</span></span><span class="hl-meta hl-brace hl-round hl-ts">(</span><span class="hl-meta hl-arrow hl-ts"><span class="hl-variable hl-parameter hl-ts">group</span> </span><span class="hl-meta hl-arrow hl-ts"><span class="hl-storage hl-type hl-function hl-arrow hl-ts">=&gt;</span> </span><span class="hl-meta hl-brace hl-round hl-ts">(</span><span class="hl-meta hl-objectliteral hl-ts"><span class="hl-punctuation hl-definition hl-block hl-ts">{</span> <span class="hl-meta hl-object hl-member hl-ts"><span class="hl-variable hl-other hl-readwrite hl-ts">group</span></span><span class="hl-punctuation hl-separator hl-comma hl-ts">,</span> <span class="hl-meta hl-object hl-member hl-ts"><span class="hl-meta hl-object-literal hl-key hl-ts">count</span></span><span class="hl-meta hl-object hl-member hl-ts"><span class="hl-meta hl-object-literal hl-key hl-ts"><span class="hl-punctuation hl-separator hl-key-value hl-ts">:</span></span> <span class="hl-variable hl-other hl-object hl-ts">group</span><span class="hl-punctuation hl-accessor hl-ts">.</span><span class="hl-support hl-variable hl-property hl-ts">length</span> </span><span class="hl-punctuation hl-definition hl-block hl-ts">}</span></span><span class="hl-meta hl-brace hl-round hl-ts">)</span><span class="hl-meta hl-brace hl-round hl-ts">)</span>\n</span></span></span></div></td></tr>',
                                '<tr><td class="line" data-line="46"/><td class="code"><div><span class="hl-source hl-ts"><span class="hl-meta hl-arrow hl-ts"><span class="hl-meta hl-block hl-ts">        <span class="hl-meta hl-function-call hl-ts"><span class="hl-punctuation hl-accessor hl-ts">.</span><span class="hl-entity hl-name hl-function hl-ts">value</span></span><span class="hl-meta hl-brace hl-round hl-ts">(</span><span class="hl-meta hl-brace hl-round hl-ts">)</span>\n</span></span></span></div></td></tr>',
                                '<tr><td class="line" data-line="47"/><td class="code"><div><span class="hl-source hl-ts"><span class="hl-meta hl-arrow hl-ts"><span class="hl-meta hl-block hl-ts">        <span class="hl-meta hl-function-call hl-ts"><span class="hl-punctuation hl-accessor hl-ts">.</span><span class="hl-entity hl-name hl-function hl-ts">filter</span></span><span class="hl-meta hl-brace hl-round hl-ts">(</span><span class="hl-meta hl-arrow hl-ts"><span class="hl-meta hl-parameters hl-ts"><span class="hl-punctuation hl-definition hl-parameters hl-begin hl-ts">(</span><span class="hl-meta hl-parameter hl-object-binding-pattern hl-ts"><span class="hl-punctuation hl-definition hl-binding-pattern hl-object hl-ts">{</span> <span class="hl-variable hl-parameter hl-ts">count</span> <span class="hl-punctuation hl-definition hl-binding-pattern hl-object hl-ts">}</span></span><span class="hl-punctuation hl-definition hl-parameters hl-end hl-ts">)</span></span> </span><span class="hl-meta hl-arrow hl-ts"><span class="hl-storage hl-type hl-function hl-arrow hl-ts">=&gt;</span> </span><span class="hl-variable hl-other hl-readwrite hl-ts">count</span> <span class="hl-keyword hl-operator hl-relational hl-ts">&gt;</span> <span class="hl-constant hl-numeric hl-decimal hl-ts">1</span><span class="hl-meta hl-brace hl-round hl-ts">)</span>\n</span></span></span></div></td></tr>',
                                '<tr><td class="line" data-line="48"/><td class="code"><div><span class="hl-source hl-ts"><span class="hl-meta hl-arrow hl-ts"><span class="hl-meta hl-block hl-ts">        <span class="hl-meta hl-function-call hl-ts"><span class="hl-punctuation hl-accessor hl-ts">.</span><span class="hl-entity hl-name hl-function hl-ts">map</span></span><span class="hl-meta hl-brace hl-round hl-ts">(</span><span class="hl-meta hl-arrow hl-ts"><span class="hl-meta hl-parameters hl-ts"><span class="hl-punctuation hl-definition hl-parameters hl-begin hl-ts">(</span><span class="hl-meta hl-parameter hl-object-binding-pattern hl-ts"><span class="hl-punctuation hl-definition hl-binding-pattern hl-object hl-ts">{</span> <span class="hl-variable hl-parameter hl-ts">group</span> <span class="hl-punctuation hl-definition hl-binding-pattern hl-object hl-ts">}</span></span><span class="hl-punctuation hl-definition hl-parameters hl-end hl-ts">)</span></span> </span><span class="hl-meta hl-arrow hl-ts"><span class="hl-storage hl-type hl-function hl-arrow hl-ts">=&gt;</span> </span><span class="hl-variable hl-other hl-readwrite hl-ts">group</span><span class="hl-meta hl-brace hl-round hl-ts">)</span>\n</span></span></span></div></td></tr>',
                                '<tr><td class="line" data-line="49"/><td class="code"><div><span class="hl-source hl-ts"><span class="hl-meta hl-arrow hl-ts"><span class="hl-meta hl-block hl-ts">\n</span></span></span></div></td></tr>',
                                '<tr><td class="line" data-line="50"/><td class="code"><div><span class="hl-source hl-ts"><span class="hl-meta hl-arrow hl-ts"><span class="hl-meta hl-block hl-ts">    <span class="hl-keyword hl-control hl-conditional hl-ts">if</span> <span class="hl-meta hl-brace hl-round hl-ts">(</span><span class="hl-variable hl-other hl-object hl-ts">setsOfDupes</span><span class="hl-punctuation hl-accessor hl-ts">.</span><span class="hl-support hl-variable hl-property hl-ts">length</span> <span class="hl-keyword hl-operator hl-relational hl-ts">&gt;</span> <span class="hl-constant hl-numeric hl-decimal hl-ts">0</span><span class="hl-meta hl-brace hl-round hl-ts">)</span> <span class="hl-meta hl-block hl-ts"><span class="hl-punctuation hl-definition hl-block hl-ts">{</span>\n</span></span></span></span></div></td></tr>',
                                '<tr><td class="line" data-line="51"/><td class="code"><div><span class="hl-source hl-ts"><span class="hl-meta hl-arrow hl-ts"><span class="hl-meta hl-block hl-ts"><span class="hl-meta hl-block hl-ts">        <span class="hl-meta hl-function-call hl-ts"><span class="hl-entity hl-name hl-function hl-ts">fail</span></span><span class="hl-meta hl-brace hl-round hl-ts">(</span>\n</span></span></span></span></div></td></tr>',
                                '<tr><td class="line" data-line="52"/><td class="code"><div><span class="hl-source hl-ts"><span class="hl-meta hl-arrow hl-ts"><span class="hl-meta hl-block hl-ts"><span class="hl-meta hl-block hl-ts">            <span class="hl-new hl-expr hl-ts"><span class="hl-keyword hl-operator hl-new hl-ts">new</span> <span class="hl-entity hl-name hl-type hl-ts">Error</span><span class="hl-meta hl-brace hl-round hl-ts">(</span>\n</span></span></span></span></span></div></td></tr>',
                                '<tr><td class="line" data-line="53"/><td class="code"><div><span class="hl-source hl-ts"><span class="hl-meta hl-arrow hl-ts"><span class="hl-meta hl-block hl-ts"><span class="hl-meta hl-block hl-ts"><span class="hl-new hl-expr hl-ts">                <span class="hl-string hl-template hl-ts"><span class="hl-punctuation hl-definition hl-string hl-template hl-begin hl-ts">`</span>Sets of lines with duplicate IDs:<span class="hl-constant hl-character hl-escape hl-ts">\\n</span><span class="hl-punctuation hl-definition hl-string hl-template hl-end hl-ts">`</span></span> <span class="hl-keyword hl-operator hl-arithmetic hl-ts">+</span>\n</span></span></span></span></span></div></td></tr>',
                                '<tr><td class="line" data-line="54"/><td class="code"><div><span class="hl-source hl-ts"><span class="hl-meta hl-arrow hl-ts"><span class="hl-meta hl-block hl-ts"><span class="hl-meta hl-block hl-ts"><span class="hl-new hl-expr hl-ts">                    <span class="hl-variable hl-other hl-readwrite hl-ts">setsOfDupes</span>\n</span></span></span></span></span></div></td></tr>',
                                '<tr><td class="line" data-line="55"/><td class="code"><div><span class="hl-source hl-ts"><span class="hl-meta hl-arrow hl-ts"><span class="hl-meta hl-block hl-ts"><span class="hl-meta hl-block hl-ts"><span class="hl-new hl-expr hl-ts">                        <span class="hl-meta hl-function-call hl-ts"><span class="hl-punctuation hl-accessor hl-ts">.</span><span class="hl-entity hl-name hl-function hl-ts">map</span></span><span class="hl-meta hl-brace hl-round hl-ts">(</span><span class="hl-meta hl-arrow hl-ts"><span class="hl-variable hl-parameter hl-ts">dupes</span> </span><span class="hl-meta hl-arrow hl-ts"><span class="hl-storage hl-type hl-function hl-arrow hl-ts">=&gt;</span>\n</span></span></span></span></span></span></div></td></tr>',
                                '<tr><td class="line" data-line="56"/><td class="code"><div><span class="hl-source hl-ts"><span class="hl-meta hl-arrow hl-ts"><span class="hl-meta hl-block hl-ts"><span class="hl-meta hl-block hl-ts"><span class="hl-new hl-expr hl-ts"><span class="hl-meta hl-arrow hl-ts">                            </span><span class="hl-meta hl-function-call hl-ts"><span class="hl-variable hl-other hl-object hl-ts">dupes</span><span class="hl-punctuation hl-accessor hl-ts">.</span><span class="hl-entity hl-name hl-function hl-ts">map</span></span><span class="hl-meta hl-brace hl-round hl-ts">(</span><span class="hl-meta hl-arrow hl-ts"><span class="hl-variable hl-parameter hl-ts">item</span> </span><span class="hl-meta hl-arrow hl-ts"><span class="hl-storage hl-type hl-function hl-arrow hl-ts">=&gt;</span> </span><span class="hl-meta hl-function-call hl-ts"><span class="hl-support hl-constant hl-json hl-ts">JSON</span><span class="hl-punctuation hl-accessor hl-ts">.</span><span class="hl-support hl-function hl-json hl-ts">stringify</span></span><span class="hl-meta hl-brace hl-round hl-ts">(</span><span class="hl-variable hl-other hl-readwrite hl-ts">item</span><span class="hl-meta hl-brace hl-round hl-ts">)</span><span class="hl-meta hl-brace hl-round hl-ts">)</span><span class="hl-meta hl-function-call hl-ts"><span class="hl-punctuation hl-accessor hl-ts">.</span><span class="hl-support hl-function hl-ts">join</span></span><span class="hl-meta hl-brace hl-round hl-ts">(</span><span class="hl-string hl-quoted hl-single hl-ts"><span class="hl-punctuation hl-definition hl-string hl-begin hl-ts">&#39;</span><span class="hl-constant hl-character hl-escape hl-ts">\\n</span><span class="hl-punctuation hl-definition hl-string hl-end hl-ts">&#39;</span></span><span class="hl-meta hl-brace hl-round hl-ts">)</span>\n</span></span></span></span></span></div></td></tr>',
                                '<tr><td class="line" data-line="57"/><td class="code"><div><span class="hl-source hl-ts"><span class="hl-meta hl-arrow hl-ts"><span class="hl-meta hl-block hl-ts"><span class="hl-meta hl-block hl-ts"><span class="hl-new hl-expr hl-ts">                        <span class="hl-meta hl-brace hl-round hl-ts">)</span>\n</span></span></span></span></span></div></td></tr>',
                                '<tr><td class="line" data-line="58"/><td class="code"><div><span class="hl-source hl-ts"><span class="hl-meta hl-arrow hl-ts"><span class="hl-meta hl-block hl-ts"><span class="hl-meta hl-block hl-ts"><span class="hl-new hl-expr hl-ts">                        <span class="hl-meta hl-function-call hl-ts"><span class="hl-punctuation hl-accessor hl-ts">.</span><span class="hl-support hl-function hl-ts">join</span></span><span class="hl-meta hl-brace hl-round hl-ts">(</span><span class="hl-string hl-quoted hl-single hl-ts"><span class="hl-punctuation hl-definition hl-string hl-begin hl-ts">&#39;</span><span class="hl-constant hl-character hl-escape hl-ts">\\n</span><span class="hl-constant hl-character hl-escape hl-ts">\\n</span><span class="hl-punctuation hl-definition hl-string hl-end hl-ts">&#39;</span></span><span class="hl-meta hl-brace hl-round hl-ts">)</span>\n</span></span></span></span></span></div></td></tr>',
                                '<tr><td class="line" data-line="59"/><td class="code"><div><span class="hl-source hl-ts"><span class="hl-meta hl-arrow hl-ts"><span class="hl-meta hl-block hl-ts"><span class="hl-meta hl-block hl-ts"><span class="hl-new hl-expr hl-ts">            <span class="hl-meta hl-brace hl-round hl-ts">)</span></span>\n</span></span></span></span></div></td></tr>',
                                '<tr><td class="line" data-line="60"/><td class="code"><div><span class="hl-source hl-ts"><span class="hl-meta hl-arrow hl-ts"><span class="hl-meta hl-block hl-ts"><span class="hl-meta hl-block hl-ts">        <span class="hl-meta hl-brace hl-round hl-ts">)</span>\n</span></span></span></span></div></td></tr>',
                                '<tr><td class="line" data-line="61"/><td class="code"><div><span class="hl-source hl-ts"><span class="hl-meta hl-arrow hl-ts"><span class="hl-meta hl-block hl-ts"><span class="hl-meta hl-block hl-ts">    <span class="hl-punctuation hl-definition hl-block hl-ts">}</span></span>\n</span></span></span></div></td></tr>',
                                '<tr><td class="line" data-line="62"/><td class="code"><div><span class="hl-source hl-ts"><span class="hl-meta hl-arrow hl-ts"><span class="hl-meta hl-block hl-ts"><span class="hl-punctuation hl-definition hl-block hl-ts">}</span></span></span><span class="hl-meta hl-brace hl-round hl-ts">)</span>\n</span></div></td></tr>',
                                '<tr><td class="line" data-line="63"/><td class="code"><div><span class="hl-source hl-ts">\n</span></div></td></tr>',
                                '<tr><td class="line" data-line="64"/><td class="code"><div><span class="hl-source hl-ts"><span class="hl-meta hl-function-call hl-ts"><span class="hl-entity hl-name hl-function hl-ts">test</span></span><span class="hl-meta hl-brace hl-round hl-ts">(</span><span class="hl-string hl-quoted hl-single hl-ts"><span class="hl-punctuation hl-definition hl-string hl-begin hl-ts">&#39;</span>five<span class="hl-punctuation hl-definition hl-string hl-end hl-ts">&#39;</span></span><span class="hl-punctuation hl-separator hl-comma hl-ts">,</span> <span class="hl-meta hl-arrow hl-ts"><span class="hl-storage hl-modifier hl-async hl-ts">async</span> <span class="hl-meta hl-parameters hl-ts"><span class="hl-punctuation hl-definition hl-parameters hl-begin hl-ts">(</span><span class="hl-punctuation hl-definition hl-parameters hl-end hl-ts">)</span></span> </span><span class="hl-meta hl-arrow hl-ts"><span class="hl-storage hl-type hl-function hl-arrow hl-ts">=&gt;</span> <span class="hl-meta hl-block hl-ts"><span class="hl-punctuation hl-definition hl-block hl-ts">{</span>\n</span></span></span></div></td></tr>',
                                '<tr><td class="line" data-line="65"/><td class="code"><div><span class="hl-source hl-ts"><span class="hl-meta hl-arrow hl-ts"><span class="hl-meta hl-block hl-ts">    <span class="hl-meta hl-var hl-expr hl-ts"><span class="hl-storage hl-type hl-ts">const</span> <span class="hl-meta hl-var-single-variable hl-expr hl-ts"><span class="hl-meta hl-definition hl-variable hl-ts"><span class="hl-variable hl-other hl-constant hl-ts">output</span></span> </span><span class="hl-keyword hl-operator hl-assignment hl-ts">=</span> <span class="hl-meta hl-brace hl-round hl-ts">(</span><span class="hl-keyword hl-control hl-flow hl-ts">await</span> <span class="hl-meta hl-function-call hl-ts"><span class="hl-entity hl-name hl-function hl-ts">indexExample</span></span><span class="hl-meta hl-brace hl-round hl-ts">(</span><span class="hl-string hl-quoted hl-single hl-ts"><span class="hl-punctuation hl-definition hl-string hl-begin hl-ts">&#39;</span>five<span class="hl-punctuation hl-definition hl-string hl-end hl-ts">&#39;</span></span><span class="hl-meta hl-brace hl-round hl-ts">)</span><span class="hl-meta hl-brace hl-round hl-ts">)</span><span class="hl-meta hl-function-call hl-ts"><span class="hl-punctuation hl-accessor hl-ts">.</span><span class="hl-entity hl-name hl-function hl-ts">map</span></span><span class="hl-meta hl-brace hl-round hl-ts">(</span><span class="hl-meta hl-arrow hl-ts"><span class="hl-variable hl-parameter hl-ts">v</span> </span><span class="hl-meta hl-arrow hl-ts"><span class="hl-storage hl-type hl-function hl-arrow hl-ts">=&gt;</span> </span><span class="hl-meta hl-function-call hl-ts"><span class="hl-support hl-constant hl-json hl-ts">JSON</span><span class="hl-punctuation hl-accessor hl-ts">.</span><span class="hl-support hl-function hl-json hl-ts">stringify</span></span><span class="hl-meta hl-brace hl-round hl-ts">(</span><span class="hl-variable hl-other hl-readwrite hl-ts">v</span><span class="hl-meta hl-brace hl-round hl-ts">)</span><span class="hl-meta hl-brace hl-round hl-ts">)</span></span>\n</span></span></span></div></td></tr>',
                                '<tr><td class="line" data-line="66"/><td class="code"><div><span class="hl-source hl-ts"><span class="hl-meta hl-arrow hl-ts"><span class="hl-meta hl-block hl-ts">\n</span></span></span></div></td></tr>',
                                '<tr><td class="line" data-line="67"/><td class="code"><div><span class="hl-source hl-ts"><span class="hl-meta hl-arrow hl-ts"><span class="hl-meta hl-block hl-ts">    <span class="hl-meta hl-function-call hl-ts"><span class="hl-entity hl-name hl-function hl-ts">expect</span></span><span class="hl-meta hl-brace hl-round hl-ts">(</span><span class="hl-meta hl-function-call hl-ts"><span class="hl-variable hl-other hl-object hl-ts">output</span><span class="hl-punctuation hl-accessor hl-ts">.</span><span class="hl-support hl-function hl-ts">join</span></span><span class="hl-meta hl-brace hl-round hl-ts">(</span><span class="hl-string hl-quoted hl-single hl-ts"><span class="hl-punctuation hl-definition hl-string hl-begin hl-ts">&#39;</span><span class="hl-constant hl-character hl-escape hl-ts">\\n</span><span class="hl-punctuation hl-definition hl-string hl-end hl-ts">&#39;</span></span><span class="hl-meta hl-brace hl-round hl-ts">)</span><span class="hl-meta hl-brace hl-round hl-ts">)</span><span class="hl-meta hl-function-call hl-ts"><span class="hl-punctuation hl-accessor hl-ts">.</span><span class="hl-entity hl-name hl-function hl-ts">toMatchSnapshot</span></span><span class="hl-meta hl-brace hl-round hl-ts">(</span><span class="hl-meta hl-brace hl-round hl-ts">)</span>\n</span></span></span></div></td></tr>',
                                '<tr><td class="line" data-line="68"/><td class="code"><div><span class="hl-source hl-ts"><span class="hl-meta hl-arrow hl-ts"><span class="hl-meta hl-block hl-ts"><span class="hl-punctuation hl-definition hl-block hl-ts">}</span></span></span><span class="hl-meta hl-brace hl-round hl-ts">)</span>\n</span></div></td></tr>',
                                '<tr><td class="line" data-line="69"/><td class="code"><div><span class="hl-source hl-ts">\n</span></div></td></tr>',
                                '<tr><td class="line" data-line="70"/><td class="code"><div><span class="hl-source hl-ts"><span class="hl-meta hl-function-call hl-ts"><span class="hl-entity hl-name hl-function hl-ts">test</span></span><span class="hl-meta hl-brace hl-round hl-ts">(</span><span class="hl-string hl-quoted hl-single hl-ts"><span class="hl-punctuation hl-definition hl-string hl-begin hl-ts">&#39;</span>cross-app<span class="hl-punctuation hl-definition hl-string hl-end hl-ts">&#39;</span></span><span class="hl-punctuation hl-separator hl-comma hl-ts">,</span> <span class="hl-meta hl-arrow hl-ts"><span class="hl-storage hl-modifier hl-async hl-ts">async</span> <span class="hl-meta hl-parameters hl-ts"><span class="hl-punctuation hl-definition hl-parameters hl-begin hl-ts">(</span><span class="hl-punctuation hl-definition hl-parameters hl-end hl-ts">)</span></span> </span><span class="hl-meta hl-arrow hl-ts"><span class="hl-storage hl-type hl-function hl-arrow hl-ts">=&gt;</span> <span class="hl-meta hl-block hl-ts"><span class="hl-punctuation hl-definition hl-block hl-ts">{</span>\n</span></span></span></div></td></tr>',
                                '<tr><td class="line" data-line="71"/><td class="code"><div><span class="hl-source hl-ts"><span class="hl-meta hl-arrow hl-ts"><span class="hl-meta hl-block hl-ts">    <span class="hl-meta hl-var hl-expr hl-ts"><span class="hl-storage hl-type hl-ts">const</span> <span class="hl-meta hl-var-single-variable hl-expr hl-ts"><span class="hl-meta hl-definition hl-variable hl-ts"><span class="hl-variable hl-other hl-constant hl-ts">app</span></span> </span><span class="hl-keyword hl-operator hl-assignment hl-ts">=</span> <span class="hl-meta hl-brace hl-round hl-ts">(</span><span class="hl-keyword hl-control hl-flow hl-ts">await</span> <span class="hl-meta hl-function-call hl-ts"><span class="hl-entity hl-name hl-function hl-ts">indexExample</span></span><span class="hl-meta hl-brace hl-round hl-ts">(</span><span class="hl-string hl-quoted hl-single hl-ts"><span class="hl-punctuation hl-definition hl-string hl-begin hl-ts">&#39;</span>cross-app<span class="hl-punctuation hl-definition hl-string hl-end hl-ts">&#39;</span></span><span class="hl-meta hl-brace hl-round hl-ts">)</span><span class="hl-meta hl-brace hl-round hl-ts">)</span><span class="hl-meta hl-function-call hl-ts"><span class="hl-punctuation hl-accessor hl-ts">.</span><span class="hl-entity hl-name hl-function hl-ts">map</span></span><span class="hl-meta hl-brace hl-round hl-ts">(</span><span class="hl-meta hl-arrow hl-ts"><span class="hl-variable hl-parameter hl-ts">v</span> </span><span class="hl-meta hl-arrow hl-ts"><span class="hl-storage hl-type hl-function hl-arrow hl-ts">=&gt;</span> </span><span class="hl-meta hl-function-call hl-ts"><span class="hl-support hl-constant hl-json hl-ts">JSON</span><span class="hl-punctuation hl-accessor hl-ts">.</span><span class="hl-support hl-function hl-json hl-ts">stringify</span></span><span class="hl-meta hl-brace hl-round hl-ts">(</span><span class="hl-variable hl-other hl-readwrite hl-ts">v</span><span class="hl-meta hl-brace hl-round hl-ts">)</span><span class="hl-meta hl-brace hl-round hl-ts">)</span></span>\n</span></span></span></div></td></tr>',
                                '<tr><td class="line" data-line="72"/><td class="code"><div><span class="hl-source hl-ts"><span class="hl-meta hl-arrow hl-ts"><span class="hl-meta hl-block hl-ts">    <span class="hl-meta hl-function-call hl-ts"><span class="hl-entity hl-name hl-function hl-ts">expect</span></span><span class="hl-meta hl-brace hl-round hl-ts">(</span><span class="hl-meta hl-function-call hl-ts"><span class="hl-variable hl-other hl-object hl-ts">app</span><span class="hl-punctuation hl-accessor hl-ts">.</span><span class="hl-support hl-function hl-ts">join</span></span><span class="hl-meta hl-brace hl-round hl-ts">(</span><span class="hl-string hl-quoted hl-single hl-ts"><span class="hl-punctuation hl-definition hl-string hl-begin hl-ts">&#39;</span><span class="hl-constant hl-character hl-escape hl-ts">\\n</span><span class="hl-punctuation hl-definition hl-string hl-end hl-ts">&#39;</span></span><span class="hl-meta hl-brace hl-round hl-ts">)</span><span class="hl-meta hl-brace hl-round hl-ts">)</span><span class="hl-meta hl-function-call hl-ts"><span class="hl-punctuation hl-accessor hl-ts">.</span><span class="hl-entity hl-name hl-function hl-ts">toMatchSnapshot</span></span><span class="hl-meta hl-brace hl-round hl-ts">(</span><span class="hl-meta hl-brace hl-round hl-ts">)</span>\n</span></span></span></div></td></tr>',
                                '<tr><td class="line" data-line="73"/><td class="code"><div><span class="hl-source hl-ts"><span class="hl-meta hl-arrow hl-ts"><span class="hl-meta hl-block hl-ts"><span class="hl-punctuation hl-definition hl-block hl-ts">}</span></span></span><span class="hl-meta hl-brace hl-round hl-ts">)</span>\n</span></div></td></tr>',
                                '<tr><td class="line" data-line="74"/><td class="code"><div><span class="hl-source hl-ts">\n</span></div></td></tr>',
                                '<tr><td class="line" data-line="75"/><td class="code"><div><span class="hl-source hl-ts"><span class="hl-meta hl-function-call hl-ts"><span class="hl-entity hl-name hl-function hl-ts">test</span></span><span class="hl-meta hl-brace hl-round hl-ts">(</span><span class="hl-string hl-quoted hl-single hl-ts"><span class="hl-punctuation hl-definition hl-string hl-begin hl-ts">&#39;</span>cross-lib<span class="hl-punctuation hl-definition hl-string hl-end hl-ts">&#39;</span></span><span class="hl-punctuation hl-separator hl-comma hl-ts">,</span> <span class="hl-meta hl-arrow hl-ts"><span class="hl-storage hl-modifier hl-async hl-ts">async</span> <span class="hl-meta hl-parameters hl-ts"><span class="hl-punctuation hl-definition hl-parameters hl-begin hl-ts">(</span><span class="hl-punctuation hl-definition hl-parameters hl-end hl-ts">)</span></span> </span><span class="hl-meta hl-arrow hl-ts"><span class="hl-storage hl-type hl-function hl-arrow hl-ts">=&gt;</span> <span class="hl-meta hl-block hl-ts"><span class="hl-punctuation hl-definition hl-block hl-ts">{</span>\n</span></span></span></div></td></tr>',
                                '<tr><td class="line" data-line="76"/><td class="code"><div><span class="hl-source hl-ts"><span class="hl-meta hl-arrow hl-ts"><span class="hl-meta hl-block hl-ts">    <span class="hl-meta hl-var hl-expr hl-ts"><span class="hl-storage hl-type hl-ts">const</span> <span class="hl-meta hl-var-single-variable hl-expr hl-ts"><span class="hl-meta hl-definition hl-variable hl-ts"><span class="hl-variable hl-other hl-constant hl-ts">lib</span></span> </span><span class="hl-keyword hl-operator hl-assignment hl-ts">=</span> <span class="hl-meta hl-brace hl-round hl-ts">(</span><span class="hl-keyword hl-control hl-flow hl-ts">await</span> <span class="hl-meta hl-function-call hl-ts"><span class="hl-entity hl-name hl-function hl-ts">indexExample</span></span><span class="hl-meta hl-brace hl-round hl-ts">(</span><span class="hl-string hl-quoted hl-single hl-ts"><span class="hl-punctuation hl-definition hl-string hl-begin hl-ts">&#39;</span>cross-lib<span class="hl-punctuation hl-definition hl-string hl-end hl-ts">&#39;</span></span><span class="hl-meta hl-brace hl-round hl-ts">)</span><span class="hl-meta hl-brace hl-round hl-ts">)</span><span class="hl-meta hl-function-call hl-ts"><span class="hl-punctuation hl-accessor hl-ts">.</span><span class="hl-entity hl-name hl-function hl-ts">map</span></span><span class="hl-meta hl-brace hl-round hl-ts">(</span><span class="hl-meta hl-arrow hl-ts"><span class="hl-variable hl-parameter hl-ts">v</span> </span><span class="hl-meta hl-arrow hl-ts"><span class="hl-storage hl-type hl-function hl-arrow hl-ts">=&gt;</span> </span><span class="hl-meta hl-function-call hl-ts"><span class="hl-support hl-constant hl-json hl-ts">JSON</span><span class="hl-punctuation hl-accessor hl-ts">.</span><span class="hl-support hl-function hl-json hl-ts">stringify</span></span><span class="hl-meta hl-brace hl-round hl-ts">(</span><span class="hl-variable hl-other hl-readwrite hl-ts">v</span><span class="hl-meta hl-brace hl-round hl-ts">)</span><span class="hl-meta hl-brace hl-round hl-ts">)</span></span>\n</span></span></span></div></td></tr>',
                                '<tr><td class="line" data-line="77"/><td class="code"><div><span class="hl-source hl-ts"><span class="hl-meta hl-arrow hl-ts"><span class="hl-meta hl-block hl-ts">    <span class="hl-meta hl-function-call hl-ts"><span class="hl-entity hl-name hl-function hl-ts">expect</span></span><span class="hl-meta hl-brace hl-round hl-ts">(</span><span class="hl-meta hl-function-call hl-ts"><span class="hl-variable hl-other hl-object hl-ts">lib</span><span class="hl-punctuation hl-accessor hl-ts">.</span><span class="hl-support hl-function hl-ts">join</span></span><span class="hl-meta hl-brace hl-round hl-ts">(</span><span class="hl-string hl-quoted hl-single hl-ts"><span class="hl-punctuation hl-definition hl-string hl-begin hl-ts">&#39;</span><span class="hl-constant hl-character hl-escape hl-ts">\\n</span><span class="hl-punctuation hl-definition hl-string hl-end hl-ts">&#39;</span></span><span class="hl-meta hl-brace hl-round hl-ts">)</span><span class="hl-meta hl-brace hl-round hl-ts">)</span><span class="hl-meta hl-function-call hl-ts"><span class="hl-punctuation hl-accessor hl-ts">.</span><span class="hl-entity hl-name hl-function hl-ts">toMatchSnapshot</span></span><span class="hl-meta hl-brace hl-round hl-ts">(</span><span class="hl-meta hl-brace hl-round hl-ts">)</span>\n</span></span></span></div></td></tr>',
                                '<tr><td class="line" data-line="78"/><td class="code"><div><span class="hl-source hl-ts"><span class="hl-meta hl-arrow hl-ts"><span class="hl-meta hl-block hl-ts"><span class="hl-punctuation hl-definition hl-block hl-ts">}</span></span></span><span class="hl-meta hl-brace hl-round hl-ts">)</span></span></div></td></tr>',
                            ],
                        ],
                    },
                },
            },
        },
    }),
}

export const symbolSearchStreamEvents: SearchEvent[] = [
    {
        type: 'matches',
        data: [
            {
                type: 'symbol',
                name: 'website/src/components/TestimonialCarousel.tsx',
                repository: 'gitlab.sgdev.org/sourcegraph/about',
                branches: [''],
                version: 'b1812108c8c8f0d24c03d69a883060159ebe1ae3',
                symbols: [
                    {
                        url:
                            '/gitlab.sgdev.org/sourcegraph/about/-/blob/website/src/components/TestimonialCarousel.tsx#L22:18-22:29',
                        name: 'Testimonial',
                        containerName: '',
                        kind: 'INTERFACE',
                    },
                    {
                        url:
                            '/gitlab.sgdev.org/sourcegraph/about/-/blob/website/src/components/TestimonialCarousel.tsx#L36:14-36:33',
                        name: 'TestimonialCarousel',
                        containerName: '',
                        kind: 'VARIABLE',
                    },
                ],
            },
            {
                type: 'symbol',
                name: 'src/characters.test.ts',
                repository: 'gitlab.sgdev.org/sourcegraph/event-positions',
                branches: [''],
                version: '03f7c3714a1eefe96fdaca48dd234ea3a19224ff',
                symbols: [
                    {
                        url: '/gitlab.sgdev.org/sourcegraph/event-positions/-/blob/src/characters.test.ts#L43:9-43:18',
                        name: 'testcases',
                        containerName: '',
                        kind: 'VARIABLE',
                    },
                    {
                        url:
                            '/gitlab.sgdev.org/sourcegraph/event-positions/-/blob/src/characters.test.ts#L153:15-153:20',
                        name: 'tests',
                        containerName: '',
                        kind: 'CONSTANT',
                    },
                ],
            },
            {
                type: 'symbol',
                name: 'src/positions_events.test.ts',
                repository: 'gitlab.sgdev.org/sourcegraph/event-positions',
                branches: [''],
                version: '03f7c3714a1eefe96fdaca48dd234ea3a19224ff',
                symbols: [
                    {
                        url:
                            '/gitlab.sgdev.org/sourcegraph/event-positions/-/blob/src/positions_events.test.ts#L15:9-15:18',
                        name: 'testcases',
                        containerName: '',
                        kind: 'VARIABLE',
                    },
                ],
            },
            {
                type: 'symbol',
                name: 'src/typings/SourcegraphGQL.d.ts',
                repository: 'gitlab.sgdev.org/sourcegraph/sourcegraph-code-discussions',
                branches: [''],
                version: 'f8c71486372087822b7995f0d572c6422b7ae0e5',
                symbols: [
                    {
                        url:
                            '/gitlab.sgdev.org/sourcegraph/sourcegraph-code-discussions/-/blob/src/typings/SourcegraphGQL.d.ts#L493:9-493:31',
                        name: 'lastIndexedRevOrLatest',
                        containerName: 'SourcegraphGQL.IRepository',
                        kind: 'CLASS',
                    },
                    {
                        url:
                            '/gitlab.sgdev.org/sourcegraph/sourcegraph-code-discussions/-/blob/src/typings/SourcegraphGQL.d.ts#L1046:9-1046:23',
                        name: 'latestSettings',
                        containerName: 'SourcegraphGQL.IUser',
                        kind: 'FUNCTION',
                    },
                    {
                        url:
                            '/gitlab.sgdev.org/sourcegraph/sourcegraph-code-discussions/-/blob/src/typings/SourcegraphGQL.d.ts#L1170:9-1170:23',
                        name: 'latestSettings',
                        containerName: 'SourcegraphGQL.IConfigurationSubject',
                        kind: 'ENUM',
                    },
                    {
                        url:
                            '/gitlab.sgdev.org/sourcegraph/sourcegraph-code-discussions/-/blob/src/typings/SourcegraphGQL.d.ts#L1317:9-1317:23',
                        name: 'latestSettings',
                        containerName: 'SourcegraphGQL.IOrg',
                        kind: 'PROPERTY',
                    },
                    {
                        url:
                            '/gitlab.sgdev.org/sourcegraph/sourcegraph-code-discussions/-/blob/src/typings/SourcegraphGQL.d.ts#L4633:9-4633:23',
                        name: 'latestSettings',
                        containerName: 'SourcegraphGQL.ISite',
                        kind: 'PROPERTY',
                    },
                ],
            },
        ],
    },
    { type: 'done', data: {} },
]
