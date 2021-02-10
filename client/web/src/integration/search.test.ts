import expect from 'expect'
import { commonWebGraphQlResults } from './graphQlResults'
import { RepoGroupsResult, SearchResult, SearchSuggestionsResult, WebGraphQlOperations } from '../graphql-operations'
import { Driver, createDriverForTest, percySnapshot } from '../../../shared/src/testing/driver'
import { afterEachSaveScreenshotIfFailed } from '../../../shared/src/testing/screenshotReporter'
import { WebIntegrationTestContext, createWebIntegrationTestContext } from './context'
import { test } from 'mocha'
import { siteGQLID, siteID } from './jscontext'
import { SharedGraphQlOperations, SymbolKind } from '../../../shared/src/graphql-operations'
import { SearchEvent } from '../search/stream'
import { Key } from 'ts-key-enum'

const searchResults = (): SearchResult => ({
    search: {
        results: {
            __typename: 'SearchResults',
            limitHit: true,
            matchCount: 30,
            approximateResultCount: '30+',
            missing: [],
            cloning: [],
            repositoriesCount: 372,
            timedout: [],
            indexUnavailable: false,
            dynamicFilters: [
                {
                    value: 'archived:yes',
                    label: 'archived:yes',
                    count: 5,
                    limitHit: true,
                    kind: 'repo',
                },
                {
                    value: 'fork:yes',
                    label: 'fork:yes',
                    count: 46,
                    limitHit: true,
                    kind: 'repo',
                },
                {
                    value: 'repo:^github\\.com/Algorilla/manta-ray$',
                    label: 'github.com/Algorilla/manta-ray',
                    count: 1,
                    limitHit: false,
                    kind: 'repo',
                },
            ],
            results: [
                {
                    __typename: 'Repository',
                    id: 'UmVwb3NpdG9yeTozODcxOTM4Nw==',
                    name: 'github.com/Algorilla/manta-ray',
                    label: {
                        html:
                            '\u003Cp\u003E\u003Ca href="/github.com/Algorilla/manta-ray" rel="nofollow"\u003Egithub.com/Algorilla/manta-ray\u003C/a\u003E\u003C/p\u003E\n',
                    },
                    url: '/github.com/Algorilla/manta-ray',
                    icon:
                        'data:image/svg+xml;base64,PHN2ZyB2ZXJzaW9uPSIxLjEiIGlkPSJMYXllcl8xIiB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHhtbG5zOnhsaW5rPSJodHRwOi8vd3d3LnczLm9yZy8xOTk5L3hsaW5rIiB4PSIwcHgiIHk9IjBweCIKCSB2aWV3Qm94PSIwIDAgNjQgNjQiIHN0eWxlPSJlbmFibGUtYmFja2dyb3VuZDpuZXcgMCAwIDY0IDY0OyIgeG1sOnNwYWNlPSJwcmVzZXJ2ZSI+Cjx0aXRsZT5JY29ucyA0MDA8L3RpdGxlPgo8Zz4KCTxwYXRoIGQ9Ik0yMywyMi40YzEuMywwLDIuNC0xLjEsMi40LTIuNHMtMS4xLTIuNC0yLjQtMi40Yy0xLjMsMC0yLjQsMS4xLTIuNCwyLjRTMjEuNywyMi40LDIzLDIyLjR6Ii8+Cgk8cGF0aCBkPSJNMzUsMjYuNGMxLjMsMCwyLjQtMS4xLDIuNC0yLjRzLTEuMS0yLjQtMi40LTIuNHMtMi40LDEuMS0yLjQsMi40UzMzLjcsMjYuNCwzNSwyNi40eiIvPgoJPHBhdGggZD0iTTIzLDQyLjRjMS4zLDAsMi40LTEuMSwyLjQtMi40cy0xLjEtMi40LTIuNC0yLjRzLTIuNCwxLjEtMi40LDIuNFMyMS43LDQyLjQsMjMsNDIuNHoiLz4KCTxwYXRoIGQ9Ik01MCwxNmgtMS41Yy0wLjMsMC0wLjUsMC4yLTAuNSwwLjV2MzVjMCwwLjMtMC4yLDAuNS0wLjUsMC41aC0yN2MtMC41LDAtMS0wLjItMS40LTAuNmwtMC42LTAuNmMtMC4xLTAuMS0wLjEtMC4yLTAuMS0wLjQKCQljMC0wLjMsMC4yLTAuNSwwLjUtMC41SDQ0YzEuMSwwLDItMC45LDItMlYxMmMwLTEuMS0wLjktMi0yLTJIMTRjLTEuMSwwLTIsMC45LTIsMnYzNi4zYzAsMS4xLDAuNCwyLjEsMS4yLDIuOGwzLjEsMy4xCgkJYzEuMSwxLjEsMi43LDEuOCw0LjIsMS44SDUwYzEuMSwwLDItMC45LDItMlYxOEM1MiwxNi45LDUxLjEsMTYsNTAsMTZ6IE0xOSwyMGMwLTIuMiwxLjgtNCw0LTRjMS40LDAsMi44LDAuOCwzLjUsMgoJCWMxLjEsMS45LDAuNCw0LjMtMS41LDUuNFYzM2MxLTAuNiwyLjMtMC45LDQtMC45YzEsMCwyLTAuNSwyLjgtMS4zQzMyLjUsMzAsMzMsMjkuMSwzMywyOHYtMC42Yy0xLjItMC43LTItMi0yLTMuNQoJCWMwLTIuMiwxLjgtNCw0LTRjMi4yLDAsNCwxLjgsNCw0YzAsMS41LTAuOCwyLjctMiwzLjVoMGMtMC4xLDIuMS0wLjksNC40LTIuNSw2Yy0xLjYsMS42LTMuNCwyLjQtNS41LDIuNWMtMC44LDAtMS40LDAuMS0xLjksMC4zCgkJYy0wLjIsMC4xLTEsMC44LTEuMiwwLjlDMjYuNiwzOCwyNywzOC45LDI3LDQwYzAsMi4yLTEuOCw0LTQsNHMtNC0xLjgtNC00YzAtMS41LDAuOC0yLjcsMi0zLjRWMjMuNEMxOS44LDIyLjcsMTksMjEuNCwxOSwyMHoiLz4KPC9nPgo8L3N2Zz4K',
                    detail: { html: '\u003Cp\u003ERepository name match\u003C/p\u003E\n' },
                    matches: [],
                },
            ],
            alert: null,
            elapsedMilliseconds: 103,
        },
    },
})

const commonSearchGraphQLResults: Partial<WebGraphQlOperations & SharedGraphQlOperations> = {
    ...commonWebGraphQlResults,
    Search: searchResults,
    SearchSuggestions: (): SearchSuggestionsResult => ({
        search: {
            suggestions: [],
        },
    }),
    RepoGroups: (): RepoGroupsResult => ({
        repoGroups: [],
    }),
}

describe('Search', () => {
    let driver: Driver
    before(async () => {
        driver = await createDriverForTest()
    })
    after(() => driver?.close())
    let testContext: WebIntegrationTestContext
    beforeEach(async function () {
        testContext = await createWebIntegrationTestContext({
            driver,
            currentTest: this.currentTest!,
            directory: __dirname,
        })
    })
    afterEachSaveScreenshotIfFailed(() => driver.page)
    afterEach(() => testContext?.dispose())

    const waitAndFocusInput = async () => {
        await driver.page.waitForSelector('.monaco-editor .view-lines')
        await driver.page.click('.monaco-editor .view-lines')
    }

    const getSearchFieldValue = (driver: Driver): Promise<string | undefined> =>
        driver.page.evaluate(() => document.querySelector<HTMLTextAreaElement>('#monaco-query-input textarea')?.value)

    describe('Search filters', () => {
        test('Search filters are shown on search result pages and clicking them triggers a new search', async () => {
            testContext.overrideGraphQL({
                ...commonSearchGraphQLResults,
            })
            const dynamicFilters = ['archived:yes', 'repo:^github\\.com/Algorilla/manta-ray$']
            const origQuery = 'foo'
            for (const filter of dynamicFilters) {
                await driver.page.goto(
                    `${driver.sourcegraphBaseUrl}/search?q=${encodeURIComponent(origQuery)}&patternType=literal`
                )
                await driver.page.waitForSelector(`[data-testid="filter-chip"][value=${JSON.stringify(filter)}]`)
                await driver.page.click(`[data-testid="filter-chip"][value=${JSON.stringify(filter)}]`)
                await driver.page.waitForFunction(
                    expectedQuery => {
                        const url = new URL(document.location.href)
                        const query = url.searchParams.get('q')
                        return query && query.trim() === expectedQuery
                    },
                    { timeout: 5000 },
                    `${origQuery} ${filter}`
                )
            }
        })
    })

    describe('Suggestions', () => {
        test('Typing in the search field shows relevant suggestions', async () => {
            testContext.overrideGraphQL({
                ...commonSearchGraphQLResults,
                SearchSuggestions: () => ({
                    search: {
                        suggestions: [
                            { __typename: 'Repository', name: 'github.com/auth0/go-jwt-middleware' },
                            {
                                __typename: 'Symbol',
                                name: 'OnError',
                                containerName: 'jwtmiddleware',
                                url: '/github.com/auth0/go-jwt-middleware/-/blob/jwtmiddleware.go#L56:1-56:14',
                                kind: SymbolKind.STRUCT,
                                location: {
                                    resource: {
                                        path: 'jwtmiddleware.go',
                                        repository: { name: 'github.com/auth0/go-jwt-middleware' },
                                    },
                                },
                            },
                            {
                                __typename: 'File',
                                path: 'jwtmiddleware.go',
                                name: 'jwtmiddleware.go',
                                isDirectory: false,
                                url: '/github.com/auth0/go-jwt-middleware/-/blob/jwtmiddleware.go',
                                repository: { name: 'github.com/auth0/go-jwt-middleware' },
                            },
                        ],
                    },
                }),
            })
            // Repo autocomplete from homepage
            await driver.page.goto(driver.sourcegraphBaseUrl + '/search')
            // Using id selector rather than `test-` classes as Monaco doesn't allow customizing classes
            await driver.page.waitForSelector('#monaco-query-input')
            await driver.replaceText({
                selector: '#monaco-query-input',
                newText: 'go-jwt-middlew',
                enterTextMethod: 'type',
            })
            await driver.page.waitForSelector('.monaco-query-input-container .suggest-widget.visible')
            await driver.findElementWithText('github.com/auth0/go-jwt-middleware', {
                action: 'click',
                wait: { timeout: 5000 },
                selector: '.monaco-query-input-container .suggest-widget.visible span',
            })
            expect(await getSearchFieldValue(driver)).toStrictEqual('repo:^github\\.com/auth0/go-jwt-middleware$ ')

            // Submit search
            await driver.page.keyboard.press(Key.Enter)

            // File autocomplete from repo search bar
            await driver.page.waitForSelector('#monaco-query-input')
            await driver.page.focus('#monaco-query-input')
            await driver.page.keyboard.type('jwtmi')
            await driver.page.waitForSelector('.monaco-query-input-container .suggest-widget.visible')
            await driver.findElementWithText('jwtmiddleware.go', {
                selector: '.monaco-query-input-container .suggest-widget.visible span',
                wait: { timeout: 5000 },
            })
            await driver.page.keyboard.press(Key.Tab)
            expect(await getSearchFieldValue(driver)).toStrictEqual(
                'repo:^github\\.com/auth0/go-jwt-middleware$ file:^jwtmiddleware\\.go$ '
            )

            // Symbol autocomplete in top search bar
            await driver.page.keyboard.type('On')
            await driver.page.waitForSelector('.monaco-query-input-container .suggest-widget.visible')
            await driver.findElementWithText('OnError', {
                selector: '.monaco-query-input-container .suggest-widget.visible span',
                wait: { timeout: 5000 },
            })
        })
    })

    describe('Search field value', () => {
        test('Is set from the URL query parameter when loading a search-related page', async () => {
            testContext.overrideGraphQL({
                ...commonSearchGraphQLResults,
            })
            await driver.page.goto(driver.sourcegraphBaseUrl + '/search?q=foo')
            await driver.page.waitForSelector('#monaco-query-input')
            expect(await getSearchFieldValue(driver)).toStrictEqual('foo')
            // Field value is cleared when navigating to a non search-related page
            await driver.page.waitForSelector('[data-testid="campaigns"]')
            await driver.page.click('[data-testid="campaigns"]')
            expect(await getSearchFieldValue(driver)).toStrictEqual('')
            // Field value is restored when the back button is pressed
            await driver.page.goBack()
            expect(await getSearchFieldValue(driver)).toStrictEqual('foo')
        })
    })

    describe('Case sensitivity toggle', () => {
        test('Clicking toggle turns on case sensitivity', async () => {
            testContext.overrideGraphQL({
                ...commonSearchGraphQLResults,
            })
            await driver.page.goto(driver.sourcegraphBaseUrl + '/search')
            await driver.page.waitForSelector('.test-query-input', { visible: true })
            await driver.page.waitForSelector('.test-case-sensitivity-toggle')
            await waitAndFocusInput()
            await driver.page.type('.test-query-input', 'test')
            await driver.page.click('.test-case-sensitivity-toggle')
            await driver.assertWindowLocation('/search?q=test&patternType=literal&case=yes')
        })

        test('Clicking toggle turns off case sensitivity and removes case= URL parameter', async () => {
            testContext.overrideGraphQL({
                ...commonSearchGraphQLResults,
            })
            await driver.page.goto(driver.sourcegraphBaseUrl + '/search?q=test&patternType=literal&case=yes')
            await driver.page.waitForSelector('.test-query-input', { visible: true })
            await driver.page.waitForSelector('.test-case-sensitivity-toggle')
            await driver.page.click('.test-case-sensitivity-toggle')
            await driver.assertWindowLocation('/search?q=test&patternType=literal')
        })
    })

    describe('Structural search toggle', () => {
        test('Clicking toggle turns on structural search', async () => {
            testContext.overrideGraphQL({
                ...commonSearchGraphQLResults,
            })
            await driver.page.goto(driver.sourcegraphBaseUrl + '/search')
            await driver.page.waitForSelector('.test-query-input', { visible: true })
            await driver.page.waitForSelector('.test-structural-search-toggle')
            await waitAndFocusInput()
            await driver.page.type('.test-query-input', 'test')
            await driver.page.click('.test-structural-search-toggle')
            await driver.assertWindowLocation('/search?q=test&patternType=structural')
        })

        test('Clicking toggle turns on structural search and removes existing patternType parameter', async () => {
            testContext.overrideGraphQL({
                ...commonSearchGraphQLResults,
            })
            await driver.page.goto(driver.sourcegraphBaseUrl + '/search?q=test&patternType=regexp')
            await waitAndFocusInput()
            await driver.page.waitForSelector('.test-query-input', { visible: true })
            await driver.page.waitForSelector('.test-structural-search-toggle')
            await driver.page.click('.test-structural-search-toggle')
            await driver.assertWindowLocation('/search?q=test&patternType=structural')
        })

        test('Clicking toggle turns off structural saerch and reverts to default pattern type', async () => {
            testContext.overrideGraphQL({
                ...commonSearchGraphQLResults,
            })
            await driver.page.goto(driver.sourcegraphBaseUrl + '/search?q=test&patternType=structural')
            await driver.page.waitForSelector('.test-query-input', { visible: true })
            await driver.page.waitForSelector('.test-structural-search-toggle')
            await driver.page.click('.test-structural-search-toggle')
            await driver.assertWindowLocation('/search?q=test&patternType=literal')
        })
    })

    describe('Search button', () => {
        test('Clicking search button executes search', async () => {
            testContext.overrideGraphQL({
                ...commonSearchGraphQLResults,
            })

            await driver.page.goto(driver.sourcegraphBaseUrl + '/search?q=test&patternType=regexp')
            await driver.page.waitForSelector('.test-search-button', { visible: true })
            // Note: Delay added because this test has been intermittently failing without it. Monaco search bar may drop events if it gets too many too fast.
            await driver.page.keyboard.type(' hello', { delay: 50 })
            await driver.page.click('.test-search-button')
            await driver.assertWindowLocation('/search?q=test+hello&patternType=regexp')
        })
    })

    describe('Streaming search', () => {
        const viewerSettingsWithStreamingSearch: Partial<WebGraphQlOperations> = {
            ViewerSettings: () => ({
                viewerSettings: {
                    subjects: [
                        {
                            __typename: 'DefaultSettings',
                            settingsURL: null,
                            viewerCanAdminister: false,
                            latestSettings: {
                                id: 0,
                                contents: JSON.stringify({ experimentalFeatures: { searchStreaming: true } }),
                            },
                        },
                        {
                            __typename: 'Site',
                            id: siteGQLID,
                            siteID,
                            latestSettings: {
                                id: 470,
                                contents: JSON.stringify({ experimentalFeatures: { searchStreaming: true } }),
                            },
                            settingsURL: '/site-admin/global-settings',
                            viewerCanAdminister: true,
                        },
                    ],
                    final: JSON.stringify({}),
                },
            }),
        }

        test('Streaming search with single repo result', async () => {
            const searchStreamEvents: SearchEvent[] = [
                { type: 'matches', data: [{ type: 'repo', repository: 'github.com/sourcegraph/sourcegraph' }] },
                { type: 'done', data: {} },
            ]

            testContext.overrideGraphQL({ ...commonSearchGraphQLResults, ...viewerSettingsWithStreamingSearch })
            testContext.overrideSearchStreamEvents(searchStreamEvents)

            await driver.page.goto(driver.sourcegraphBaseUrl + '/search?q=test&patternType=regexp')
            await driver.page.waitForSelector('.test-search-result', { visible: true })

            const results = await driver.page.evaluate(() =>
                [...document.querySelectorAll('.test-search-result-label')].map(label =>
                    (label.textContent || '').trim()
                )
            )
            expect(results).toEqual(['github.com/sourcegraph/sourcegraph'])
        })
    })

    describe('Search statistics', () => {
        // This is a substring that appears in the sourcegraph/go-diff repository, which is present
        // in the external service added for the e2e test. It is OK if it starts to appear in other
        // repositories (such as sourcegraph/sourcegraph now that it's mentioned here); the test
        // just checks that it is found in at least 1 Go file.
        const uniqueString = 'Incomplete-'
        const uniqueStringPostfix = 'Lines'

        test('button on search results page', async () => {
            testContext.overrideGraphQL({
                ...commonSearchGraphQLResults,
            })
            await driver.page.goto(`${driver.sourcegraphBaseUrl}/search?q=${uniqueString}`)
            await driver.page.waitForSelector(`a[href="/stats?q=${uniqueString}"]`)
        })

        test('page', async () => {
            testContext.overrideGraphQL({
                ...commonSearchGraphQLResults,
            })
            await driver.page.goto(`${driver.sourcegraphBaseUrl}/stats?q=${uniqueString}`)

            // Ensure the global navbar hides the search input (to avoid confusion with the one on
            // the stats page).
            await driver.page.waitForSelector('.global-navbar a.nav-link[href="/search"]')
            expect(
                await driver.page.evaluate(() => document.querySelectorAll('#monaco-query-input').length)
            ).toStrictEqual(0)

            const queryInputValue = () =>
                driver.page.evaluate(() => {
                    const input = document.querySelector<HTMLInputElement>('.test-stats-query')
                    return input ? input.value : null
                })

            // Check for a Go result (the sample repositories have Go files).
            await driver.page.waitForSelector(`a[href*="${uniqueString}+lang:go"]`)
            expect(await queryInputValue()).toStrictEqual(uniqueString)
            await percySnapshot(driver.page, 'Search stats')

            // Update the query and rerun the computation.
            await driver.page.type('.test-stats-query', uniqueStringPostfix) // the uniqueString is followed by 'Incomplete-Lines' in go-diff
            const wantQuery = `${uniqueString}${uniqueStringPostfix}`
            expect(await queryInputValue()).toStrictEqual(wantQuery)
            await driver.page.click('.test-stats-query-update')
            await driver.page.waitForSelector(`a[href*="${wantQuery}+lang:go"]`)
            expect(driver.page.url().endsWith(`/stats?q=${wantQuery}`)).toBeTruthy()
        })
    })

    describe('Search result type tabs', () => {
        test('Search results type tabs appear', async () => {
            testContext.overrideGraphQL({
                ...commonSearchGraphQLResults,
            })
            await driver.page.goto(
                driver.sourcegraphBaseUrl + '/search?q=repo:%5Egithub.com/gorilla/mux%24&patternType=regexp'
            )
            await driver.page.waitForSelector('.test-search-result-type-tabs', { visible: true })
            await driver.page.waitForSelector('.test-search-result-tab--active', { visible: true })
            const tabs = await driver.page.evaluate(() =>
                [...document.querySelectorAll('.test-search-result-tab')].map(tab => tab.textContent)
            )

            expect(tabs.length).toEqual(6)
            expect(tabs).toStrictEqual(['Code', 'Diffs', 'Commits', 'Symbols', 'Repositories', 'Filenames'])

            const activeTab = await driver.page.evaluate(
                () => document.querySelectorAll('.test-search-result-tab--active').length
            )
            expect(activeTab).toEqual(1)

            const label = await driver.page.evaluate(
                () => document.querySelector('.test-search-result-tab--active')!.textContent || ''
            )
            expect(label).toEqual('Code')
        })
    })

    describe('Search component', () => {
        test('redirects to a URL with &patternType=regexp if no patternType in URL', async () => {
            testContext.overrideGraphQL({
                ...commonSearchGraphQLResults,
            })
            await driver.page.goto(driver.sourcegraphBaseUrl + '/search?q=test')
            await driver.assertWindowLocation('/search?q=test&patternType=regexp')
        })

        test('regexp toggle appears and updates patternType query parameter when clicked', async () => {
            testContext.overrideGraphQL({
                ...commonSearchGraphQLResults,
            })
            await driver.page.goto(driver.sourcegraphBaseUrl + '/search?q=test&patternType=literal')
            // Wait for monaco query input to load to avoid race condition with the intermediate input
            await driver.page.waitForSelector('#monaco-query-input')
            await driver.page.waitForSelector('.test-regexp-toggle')
            await driver.page.click('.test-regexp-toggle')
            await driver.page.goto(driver.sourcegraphBaseUrl + '/search?q=test&patternType=regexp')
            // Wait for monaco query input to load to avoid race condition with the intermediate input
            await driver.page.waitForSelector('#monaco-query-input')
            await driver.page.waitForSelector('.test-regexp-toggle')
            await driver.page.click('.test-regexp-toggle')
            await driver.page.goto(driver.sourcegraphBaseUrl + '/search?q=test&patternType=literal')
        })
    })
})
