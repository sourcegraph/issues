import expect from 'expect'
import { describe, test } from 'mocha'
import { Driver } from '../../../shared/src/testing/driver'
import { getConfig } from '../../../shared/src/testing/config'
import { getTestTools } from './util/init'
import * as GQL from '../../../shared/src/graphql/schema'
import { GraphQLClient } from './util/GraphQlClient'
import { ensureTestExternalService, search } from './util/api'
import { ensureLoggedInOrCreateTestUser, editGlobalSettings } from './util/helpers'
import { buildSearchURLQuery } from '../../../shared/src/util/url'
import { TestResourceManager } from './util/TestResourceManager'
import { setProperty } from '@sqs/jsonc-parser/lib/edit'
import { Key } from 'ts-key-enum'
import { afterEachSaveScreenshotIfFailed } from '../../../shared/src/testing/screenshotReporter'
import { editUserSettings } from './util/settings'
import assert from 'assert'

/**
 * Reads the number of results from the text at the top of the results page
 */
function getNumberOfResults() {
    // eslint-disable-next-line @typescript-eslint/no-non-null-assertion
    const matches = document.body.textContent!.match(/(\d+)\+?\sresults?/)
    if (!matches || matches.length < 2) {
        return null
    }
    const numberOfResults = parseInt(matches[1], 10)
    return isNaN(numberOfResults) ? null : numberOfResults
}

/**
 * Returns true if "No results" message is present, throws an error if there are search results,
 * and returns false otherwise.
 */
function hasNoResultsOrError(): boolean {
    if (document.querySelectorAll('.test-search-result').length > 0) {
        throw new Error('Expected "No results", but there were search results.')
    }

    const resultsElement = document.querySelector('.test-search-results')
    if (!resultsElement) {
        return false
    }
    const resultsText = (resultsElement as HTMLElement).textContent
    if (!resultsText) {
        return false
    }
    if (resultsText.includes('No results')) {
        return true
    }
    return false
}

describe('Search regression test suite', () => {
    const formattingOptions = { eol: '\n', insertSpaces: true, tabSize: 2 }

    /**
     * Test data
     */
    const testUsername = 'test-search'
    const testExternalServiceInfo = {
        kind: GQL.ExternalServiceKind.GITHUB,
        uniqueDisplayName: '[TEST] GitHub (search.test.ts)',
    }
    const testRepoSlugs = [
        'auth0/go-jwt-middleware',
        'kyoshidajp/ghkw',
        'PalmStoneGames/kube-cert-manager',
        'adjust/go-wrk',
        'P3GLEG/Whaler',
        'sajari/docconv',
        'marianogappa/chart',
        'divan/gobenchui',
        'tuna/tunasync',
        'mthbernardes/GTRS',
        'antonmedv/expr',
        'ClickHouse/clickhouse-go',
        'xwb1989/sqlparser',
        'itcloudy/ERP',
        'iovisor/kubectl-trace',
        'minio/highwayhash',
        'matryer/moq',
        'vkuznecovas/mouthful',
        'DirectXMan12/k8s-prometheus-adapter',
        'stephens2424/php',
        'ericchiang/k8s',
        'jonmorehouse/terraform-provisioner-ansible',
        'solo-io/supergloo',
        'intel-go/bytebuf',
        'xtaci/smux',
        'MatchbookLab/local-persist',
        'ossrs/go-oryx',
        'yep/eth-tweet',
        'deckarep/gosx-notifier',
        'zentures/sequence',
        'nishanths/license',
        'beego/mux',
        'status-im/status-go',
        'antonmedv/countdown',
        'lonng/nanoserver',
        'vbauerster/mpb',
        'evilsocket/sg1',
        'zhenghaoz/gorse',
        'nsf/godit',
        '3xxx/engineercms',
        'howtowhale/dvm',
        'gosuri/uitable',
        'github/vulcanizer',
        'metaparticle-io/package',
        'bwmarrin/snowflake',
        'wyh267/FalconEngine',
        'moul/sshportal',
        'fogleman/fauxgl',
        'DataDog/datadog-agent',
        'line/line-bot-sdk-go',
        'pinterest/bender',
        'esimov/diagram',
        'nytimes/openapi2proto',
        'iris-contrib/examples',
        'munnerz/kube-plex',
        'inbucket/inbucket',
        'golangci/awesome-go-linters',
        'htcat/htcat',
        'tidwall/pinhole',
        'gocraft/health',
        'ivpusic/grpool',
        'Antonito/gfile',
        'yinqiwen/gscan',
        'facebookarchive/httpcontrol',
        'josharian/impl',
        'salihciftci/liman',
        'kelseyhightower/konfd',
        'mohanson/daze',
        'google/ko',
        'freedomofdevelopers/fod',
        'sgtest/mux',
        'facebook/react',
    ]
    const config = getConfig(
        'sudoToken',
        'sudoUsername',
        'gitHubToken',
        'sourcegraphBaseUrl',
        'noCleanup',
        'testUserPassword',
        'logStatusMessages',
        'logBrowserConsole',
        'slowMo',
        'headless',
        'keepBrowser'
    )

    describe('Search over a dozen repositories', () => {
        let driver: Driver
        let gqlClient: GraphQLClient
        let resourceManager: TestResourceManager
        before(async function () {
            this.timeout(3 * 60 * 1000 + 30 * 1000)
            ;({ driver, gqlClient, resourceManager } = await getTestTools(config))
            resourceManager.add(
                'User',
                testUsername,
                await ensureLoggedInOrCreateTestUser(driver, gqlClient, { username: testUsername, ...config })
            )
            resourceManager.add(
                'External service',
                testExternalServiceInfo.uniqueDisplayName,
                await ensureTestExternalService(
                    gqlClient,
                    {
                        ...testExternalServiceInfo,
                        config: {
                            url: 'https://github.com',
                            token: config.gitHubToken,
                            repos: testRepoSlugs,
                            repositoryQuery: ['none'],
                        },
                        waitForRepos: testRepoSlugs.map(slug => 'github.com/' + slug),
                    },
                    { ...config, timeout: 3 * 60 * 1000, indexed: true }
                )
            )
        })

        afterEachSaveScreenshotIfFailed(() => driver.page)

        after(async () => {
            if (!config.noCleanup) {
                await resourceManager.destroyAll()
            }
            if (driver) {
                await driver.close()
            }
        })

        test('Global text search with 0 results.', async () => {
            await driver.page.goto(
                config.sourcegraphBaseUrl + '/search?q=asdfalksd+jflaksjdflkasjdf&patternType=literal'
            )
            await driver.page.waitForFunction(hasNoResultsOrError)
        })
        test('Global text search with double-quoted string constant ("error type:") with a few results.', async () => {
            await driver.page.goto(config.sourcegraphBaseUrl + '/search?q="error+type:%5Cn"')
            await driver.page.waitForFunction(() => document.querySelectorAll('.test-search-result').length >= 3)
        })
        test('Global text search excluding repository ("error type:") with a few results.', async () => {
            await driver.page.goto(config.sourcegraphBaseUrl + '/search?q="error+type:%5Cn"+-repo:google')
            await driver.page.waitForFunction(() => document.querySelectorAll('.test-search-result').length > 0)
            await driver.page.waitForFunction(() => {
                const results = [...document.querySelectorAll('.test-search-result')]
                if (results.length === 0) {
                    return false
                }
                const hasExcludedRepo = results.some(element => element.textContent?.includes('google'))
                if (hasExcludedRepo) {
                    throw new Error('Results contain excluded repository')
                }
                return true
            })
        })
        test('Global text search (error) with many results.', async () => {
            await driver.page.goto(config.sourcegraphBaseUrl + '/search?q=error')
            await driver.page.waitForFunction(() => document.querySelectorAll('.test-search-result').length > 10)
        })
        test('Global text search (error count:>1000), expect many results.', async () => {
            await driver.page.goto(config.sourcegraphBaseUrl + '/search?q=error+count:1000')
            await driver.page.waitForFunction(() => document.querySelectorAll('.test-search-result').length > 10)
        })
        test('Global text search (repohasfile:copying), expect many results.', async () => {
            await driver.page.goto(config.sourcegraphBaseUrl + '/search?q=repohasfile:copying')
            await driver.page.waitForFunction(() => document.querySelectorAll('.test-search-result').length >= 2)
        })
        test('Global text search (repohascommitafter:"5 years ago")', async () => {
            await driver.page.goto(
                config.sourcegraphBaseUrl + '/search?q=repohascommitafter:"5+months+ago"+test&patternType=literal'
            )
            await driver.page.waitForFunction(() => document.querySelectorAll('.test-search-result').length >= 10)
        })
        test('Global text search for something with more than 1000 results and use "count:1000".', async () => {
            await driver.page.goto(config.sourcegraphBaseUrl + '/search?q=.+count:1000')
            await driver.page.waitForFunction(() => document.querySelectorAll('.test-search-result').length > 10)
            await driver.page.addScriptTag({ content: getNumberOfResults.toString() })
            await driver.page.waitForFunction(() => getNumberOfResults() !== null)
            await driver.page.waitForFunction(
                () => {
                    const numberOfResults = getNumberOfResults()
                    return numberOfResults && numberOfResults > 1000
                },
                { timeout: 500 }
            )
        })
        test('Global text search for a regular expression without indexed search: (index:no ^func.*$), expect many results.', async () => {
            await driver.page.goto(config.sourcegraphBaseUrl + '/search?q=index:no+^func.*$')
            await driver.page.waitForFunction(() => document.querySelectorAll('.test-search-result').length > 10)
        })
        test('Global text search for a regular expression with only indexed search: (index:only ^func.*$), expect many results.', async () => {
            await driver.page.goto(config.sourcegraphBaseUrl + '/search?q=index:only+^func.*$')
            await driver.page.waitForFunction(() => document.querySelectorAll('.test-search-result').length > 10)
        })
        test('Search for a repository by name.', async () => {
            await driver.page.goto(config.sourcegraphBaseUrl + '/search?q=repo:auth0/go-jwt-middleware$')
            await driver.page.waitForFunction(() => {
                const results = document.querySelectorAll('.test-search-result')
                return results.length === 1 && (results.item(0).textContent || '').includes('go-jwt-middleware')
            })
        })
        test('Single repository, case-sensitive search.', async () => {
            await driver.page.goto(
                config.sourcegraphBaseUrl +
                    '/search?q=String+repo:%5Egithub.com/adjust/go-wrk%24+&patternType=regexp&case=yes'
            )
            await driver.page.waitForFunction(() => document.querySelectorAll('.test-search-result').length === 3)
        })
        test('Global text search, fork:only, few results', async () => {
            await driver.page.goto(config.sourcegraphBaseUrl + '/search?q=fork:only+router')
            await driver.page.waitForFunction(() => document.querySelectorAll('.test-search-result').length >= 5)
        })
        test('Global text search, fork:only, 1 result', async () => {
            await driver.page.goto(config.sourcegraphBaseUrl + '/search?q=fork:only+FORK_SENTINEL')
            await driver.page.waitForFunction(() => document.querySelectorAll('.test-search-result').length === 1)
        })
        test('Global text search, fork:no, 0 results', async () => {
            await driver.page.goto(config.sourcegraphBaseUrl + '/search?q=fork:only+FORK_SENTINEL')
            await driver.page.waitForFunction(() => document.querySelectorAll('.test-search-result').length === 0)
        })
        test('Text search non-master branch, large repository, many results', async () => {
            // The string `var ExecutionEnvironment = require('ExecutionEnvironment');` occurs 10 times on this old branch, but 0 times in current master.
            await driver.page.goto(
                config.sourcegraphBaseUrl +
                    '/search?q="var+ExecutionEnvironment+%3D+require%28%27ExecutionEnvironment%27%29%3B"+repo:%5Egithub%5C.com/facebook/react%24%400.3-stable&patternType=regexp'
            )
            await driver.page.waitForFunction(() => document.querySelectorAll('.test-search-result').length === 10)
        })
        test('Global text search filtering by language', async () => {
            await driver.page.goto(config.sourcegraphBaseUrl + '/search?q=%5Cbfunc%5Cb+lang:js')
            await driver.page.waitForFunction(() => document.querySelectorAll('.test-search-result').length > 0)
            const filenames: string[] = await driver.page.evaluate(
                () =>
                    [...document.querySelectorAll('.test-search-result')]
                        .map(element => {
                            const header = element.querySelector('[data-testid="result-container-header"')
                            if (!header?.textContent) {
                                return null
                            }
                            const components = header.textContent.split(/\s/)
                            return components[components.length - 1]
                        })
                        .filter(element => element !== null) as string[]
            )
            if (!filenames.every(filename => filename.endsWith('.js'))) {
                throw new Error('found Go results when filtering for JavaScript')
            }
        })
        test('Global search for a filename with 0 results', async () => {
            await driver.page.goto(config.sourcegraphBaseUrl + '/search?q=file:asdfasdf.go')
            await driver.page.waitForFunction(() => document.querySelectorAll('.test-search-result').length === 0)
        })
        test('Global search for a filename with a few results', async () => {
            await driver.page.goto(config.sourcegraphBaseUrl + '/search?q=file:router.go')
            await driver.page.waitForFunction(() => document.querySelectorAll('.test-search-result').length > 5)
        })
        test('Global search for a filename with many results', async () => {
            await driver.page.goto(config.sourcegraphBaseUrl + '/search?q=file:doc.go')
            await driver.page.waitForFunction(() => document.querySelectorAll('.test-search-result').length > 10)
            await driver.page.addScriptTag({ content: getNumberOfResults.toString() })
            await driver.page.waitForFunction(() => getNumberOfResults() !== null)
            await driver.page.waitForFunction(
                () => {
                    const numberResults = getNumberOfResults()
                    return numberResults !== null && numberResults > 25
                },
                { timeout: 500 }
            )
        })
        test('Global symbol search with many results', async () => {
            await driver.page.goto(config.sourcegraphBaseUrl + '/search?q=type:symbol+test+count:100')
            await driver.page.waitForFunction(() => document.querySelectorAll('.test-search-result').length > 10)
            await driver.page.addScriptTag({ content: getNumberOfResults.toString() })
            await driver.page.waitForFunction(() => (getNumberOfResults() || 0) >= 100)
        })
        test('Global symbol search with 0 results', async () => {
            await driver.page.goto(config.sourcegraphBaseUrl + '/search?q=type:symbol+asdfasdf')
            await driver.page.waitForFunction(hasNoResultsOrError)
        })
        test('Global symbol search ("type:symbol ^newroute count:100") with a few results', async () => {
            await driver.page.goto(config.sourcegraphBaseUrl + '/search?q=type:symbol+%5Enewroute+count:100')
            await driver.page.waitForFunction(() => document.querySelectorAll('.test-search-result').length > 0)
        })
        test('Indexed multiline search, many results', async () => {
            const urlQuery = buildSearchURLQuery(
                'repo:^github\\.com/facebook/react$ componentDidMount\\(\\) {\\n\\s*this',
                GQL.SearchPatternType.regexp,
                false
            )
            await driver.page.goto(config.sourcegraphBaseUrl + '/search?' + urlQuery)
            await driver.page.waitForFunction(() => document.querySelectorAll('.test-search-result').length > 10)
        })
        test('Non-indexed multiline search, many results', async () => {
            const urlQuery = buildSearchURLQuery(
                'repo:^github\\.com/facebook/react$ componentDidMount\\(\\) {\\n\\s*this index:no',
                GQL.SearchPatternType.regexp,
                false
            )
            await driver.page.goto(config.sourcegraphBaseUrl + '/search?' + urlQuery)
            await driver.page.waitForFunction(() => document.querySelectorAll('.test-search-result').length > 10)
        })
        test('Indexed multiline search, 0 results', async () => {
            const urlQuery = buildSearchURLQuery(
                'repo:^github\\.com/facebook/react$ componentDidMount\\(\\) {\\n\\s*this\\.props\\.sourcegraph\\(',
                GQL.SearchPatternType.regexp,
                false
            )
            await driver.page.goto(config.sourcegraphBaseUrl + '/search?' + urlQuery)
            await driver.page.waitForFunction(() => document.querySelectorAll('.test-search-result').length === 0)
        })
        test('Non-indexed multiline search, 0 results', async () => {
            const urlQuery = buildSearchURLQuery(
                'repo:^github\\.com/facebook/react$ componentDidMount\\(\\) {\\n\\s*this\\.props\\.sourcegraph\\( index:no',
                GQL.SearchPatternType.regexp,
                false
            )
            await driver.page.goto(config.sourcegraphBaseUrl + '/search?' + urlQuery)
            await driver.page.waitForFunction(() => document.querySelectorAll('.test-search-result').length === 0)
        })
        test('Indexed-only structural search, one or more results', async () => {
            const urlQuery = buildSearchURLQuery(
                'repo:^github\\.com/facebook/react$ index:only patterntype:structural toHaveYielded(:[args])',
                GQL.SearchPatternType.structural,
                false
            )
            await driver.page.goto(config.sourcegraphBaseUrl + '/search?' + urlQuery)
            await driver.page.waitForFunction(() => document.querySelectorAll('.test-search-result').length > 0)
        })
        test('Structural search, return repo results if pattern is empty', async () => {
            const urlQuery = buildSearchURLQuery(
                'repo:^github\\.com/facebook/react$',
                GQL.SearchPatternType.structural,
                false
            )
            await driver.page.goto(config.sourcegraphBaseUrl + '/search?' + urlQuery)
            await driver.page.waitForFunction(() => document.querySelectorAll('.test-search-result').length > 0)
        })
        test('Commit search, nonzero result', async () => {
            const urlQuery = buildSearchURLQuery(
                'repo:^github\\.com/facebook/react$ type:commit hello world',
                GQL.SearchPatternType.regexp,
                false
            )
            await driver.page.goto(config.sourcegraphBaseUrl + '/search?' + urlQuery)
            await driver.page.waitForFunction(() => document.querySelectorAll('.test-search-result').length > 0)
        })
        test('Diff search, nonzero result', async () => {
            const urlQuery = buildSearchURLQuery(
                'repo:^github\\.com/sgtest/mux$ type:diff main',
                GQL.SearchPatternType.regexp,
                false
            )
            await driver.page.goto(config.sourcegraphBaseUrl + '/search?' + urlQuery)
            await driver.page.waitForFunction(() => document.querySelectorAll('.test-search-result').length > 0)
        })
        test('Archived repos excluded by default', async () => {
            const urlQuery = buildSearchURLQuery('type:repo facebookarchive', GQL.SearchPatternType.regexp, false)
            await driver.page.goto(config.sourcegraphBaseUrl + '/search?' + urlQuery)
            await driver.page.waitForFunction(() => document.querySelectorAll('.test-search-result').length === 0)
        })
        test('Archived repos included by archived option', async () => {
            const urlQuery = buildSearchURLQuery(
                'type:repo facebookarchive archived:yes',
                GQL.SearchPatternType.regexp,
                false
            )
            await driver.page.goto(config.sourcegraphBaseUrl + '/search?' + urlQuery)
            await driver.page.waitForFunction(() => document.querySelectorAll('.test-search-result').length > 0)
        })
        test('Single archived repo included if exact', async () => {
            const urlQuery = buildSearchURLQuery(
                'repo:^github\\.com/facebookarchive/httpcontrol$ error',
                GQL.SearchPatternType.regexp,
                false
            )
            await driver.page.goto(config.sourcegraphBaseUrl + '/search?' + urlQuery)
            await driver.page.waitForFunction(() => document.querySelectorAll('.test-search-result').length > 0)
        })
        test('Fork repos excluded by default', async () => {
            const urlQuery = buildSearchURLQuery('type:repo sgtest/mux', GQL.SearchPatternType.regexp, false)
            await driver.page.goto(config.sourcegraphBaseUrl + '/search?' + urlQuery)
            await driver.page.waitForFunction(() => document.querySelectorAll('.test-search-result').length === 0)
        })
        test('Forked repos included by fork option', async () => {
            const urlQuery = buildSearchURLQuery('type:repo sgtest/mux fork:yes', GQL.SearchPatternType.regexp, false)
            await driver.page.goto(config.sourcegraphBaseUrl + '/search?' + urlQuery)
            await driver.page.waitForFunction(() => document.querySelectorAll('.test-search-result').length > 0)
        })
        test('Single forked repo included if exact', async () => {
            const urlQuery = buildSearchURLQuery(
                'repo:^github\\.com/sgtest/mux$ error',
                GQL.SearchPatternType.regexp,
                false
            )
            await driver.page.goto(config.sourcegraphBaseUrl + '/search?' + urlQuery)
            await driver.page.waitForFunction(() => document.querySelectorAll('.test-search-result').length > 0)
        })

        test('Search timeout', async function () {
            this.timeout(2 * 1000)
            const response = await search(gqlClient, 'router index:no timeout:1ns', 'V2', GQL.SearchPatternType.literal)
            expect(response.results.matchCount).toBe(0)
            expect(response.results.alert?.title).toBe('Timed out while searching')
        })

        test('Search repo group', async () => {
            resourceManager.add(
                'Global setting',
                'search.repositoryGroups',
                await editGlobalSettings(gqlClient, contents =>
                    setProperty(
                        contents,
                        ['search.repositoryGroups'],
                        {
                            test_group: ['github.com/auth0/go-jwt-middleware'],
                        },
                        formattingOptions
                    )
                )
            )

            const response = await search(gqlClient, 'repogroup:test_group route', 'V2', GQL.SearchPatternType.literal)
            expect(
                response.results.results.length > 0 &&
                    response.results.results.every(result => {
                        switch (result.__typename) {
                            case 'FileMatch':
                                return result.repository.name === 'github.com/auth0/go-jwt-middleware'
                            case 'Repository':
                                return result.name === 'github.com/auth0/go-jwt-middleware'
                            default:
                                return false
                        }
                    })
            ).toBeTruthy()
        })

        test('Search suggestions', async () => {
            const getSearchFieldValue = (): Promise<string | undefined> =>
                driver.page.evaluate(
                    () => document.querySelector<HTMLTextAreaElement>('#monaco-query-input textarea')?.value
                )
            // Repo autocomplete from homepage
            await driver.page.goto(config.sourcegraphBaseUrl + '/search')
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
            assert.strictEqual(await getSearchFieldValue(), 'repo:^github\\.com/auth0/go-jwt-middleware$ ')

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
            await driver.page.keyboard.press(Key.ArrowDown)
            await driver.page.keyboard.press(Key.Tab)
            assert.strictEqual(
                await getSearchFieldValue(),
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

        test('Search filters', async () => {
            const filterToToken = [
                ['lang:go', 'lang:go'],
                ['-file:_test\\.go$', '-file:_test\\.go$'],
            ]
            const origQuery = 'jwtmiddleware'
            for (const [filter, token] of filterToToken) {
                await driver.page.goto(
                    `${config.sourcegraphBaseUrl}/search?q=${encodeURIComponent(origQuery)}&patternType=literal`
                )
                await driver.findElementWithText(filter, {
                    action: 'click',
                    selector: 'button',
                    wait: { timeout: 5000 },
                })
                await driver.page.waitForFunction(
                    expectedQuery => {
                        const url = new URL(document.location.href)
                        const query = url.searchParams.get('q')
                        return query && query.trim() === expectedQuery
                    },
                    { timeout: 5000 },
                    `${origQuery} ${token}`
                )
            }
        })
    })

    describe('Interactive search mode', () => {
        let driver: Driver
        let gqlClient: GraphQLClient
        let resourceManager: TestResourceManager
        before(async function () {
            this.timeout(3 * 60 * 1000 + 30 * 1000)
            ;({ driver, gqlClient, resourceManager } = await getTestTools(config))
            resourceManager.add(
                'User',
                testUsername,
                await ensureLoggedInOrCreateTestUser(driver, gqlClient, { username: testUsername, ...config })
            )
            resourceManager.add(
                'External service',
                testExternalServiceInfo.uniqueDisplayName,
                await ensureTestExternalService(
                    gqlClient,
                    {
                        ...testExternalServiceInfo,
                        config: {
                            url: 'https://github.com',
                            token: config.gitHubToken,
                            repos: testRepoSlugs,
                            repositoryQuery: ['none'],
                        },
                        waitForRepos: testRepoSlugs.map(slug => 'github.com/' + slug),
                    },
                    { ...config, timeout: 3 * 60 * 1000, indexed: true }
                )
            )

            await editUserSettings(
                testUsername,
                { keyPath: [{ property: 'experimentalFeatures' }], value: { splitSearchModes: true } },
                gqlClient
            )
        })

        afterEachSaveScreenshotIfFailed(() => driver.page)

        after(async () => {
            if (!config.noCleanup) {
                await resourceManager.destroyAll()
            }
            if (driver) {
                await driver.close()
            }
        })

        test('Toggling between plain and interactive mode shows correct elements', async () => {
            await driver.page.goto(`${config.sourcegraphBaseUrl}/search`)
            await driver.page.waitForSelector('#monaco-query-input')
            await driver.page.waitForSelector('.test-search-mode-toggle')
            await driver.page.click('.test-search-mode-toggle')
            await driver.page.waitForSelector('.test-search-mode-toggle__interactive-mode')
            await driver.page.click('.test-search-mode-toggle__interactive-mode')
            await driver.page.waitForSelector('.test-interactive-mode-input')
            expect(await driver.page.evaluate(() => document.querySelectorAll('.test-query-input').length)).toEqual(1)
            expect(await driver.page.evaluate(() => document.querySelectorAll('.test-add-filter-row').length)).toEqual(
                1
            )
        })

        test('Clicking repo filter button displays selected filters row and repo filter input', async () => {
            await driver.page.click('.test-add-filter-button-repo')
            await driver.page.waitForSelector('.selected-filters-row')
            await driver.page.waitForSelector('.filter-input')
            expect(await driver.page.evaluate(() => document.querySelectorAll('.selected-filters-row').length)).toEqual(
                1
            )
            expect(await driver.page.evaluate(() => document.querySelectorAll('.filter-input').length)).toEqual(1)
            expect(
                await driver.page.evaluate(() => document.querySelector('.filter-input__label')?.textContent)
            ).toEqual('repo:')
        })
        test('Conducting a plain search query correctly returns results', async () => {
            await driver.page.goto(config.sourcegraphBaseUrl + '/search')
            await driver.page.waitForSelector('.test-query-input')
            await driver.page.click('.test-query-input')
            await driver.page.keyboard.type('error count:100')
            await driver.page.keyboard.press('Enter')
            await driver.assertWindowLocation('/search?q=error+count:100&patternType=literal')
            await driver.page.waitForSelector('.test-search-result')
            await driver.page.waitForFunction(() => document.querySelectorAll('.test-search-result').length > 10)
        })

        test('Adding a repo filter correctly returns results', async () => {
            await driver.page.goto(config.sourcegraphBaseUrl + '/search')
            await driver.page.waitForSelector('.test-add-filter-button-repo')
            await driver.page.click('.test-add-filter-button-repo')
            await driver.page.waitForSelector('.filter-input')
            await driver.page.keyboard.type('auth0/go-jwt-middleware$')
            await driver.page.keyboard.press('Enter')
            await driver.assertWindowLocation('/search?q=repo:%22auth0/go-jwt-middleware%24%22&patternType=literal')
            await driver.page.waitForSelector('.test-search-result')
            await driver.page.waitForFunction(() => {
                const results = document.querySelectorAll('.test-search-result')
                return results.length === 1 && (results.item(0).textContent || '').includes('go-jwt-middleware')
            })
        })
        test('Adding a term to query input correctly returns results', async () => {
            await driver.page.click('.test-query-input')
            await driver.page.keyboard.type('error')
            await driver.page.keyboard.press('Enter')
            await driver.assertWindowLocation(
                '/search?q=error+repo:%22auth0/go-jwt-middleware%24%22&patternType=literal'
            )
            await driver.page.waitForSelector('.test-search-result')
            await driver.page.waitForFunction(() => {
                const results = document.querySelectorAll('.test-file-match-children-item-wrapper')
                return results.length > 10 && (results.item(0).textContent || '').includes('error')
            })
        })
        test('Adding a file filter to query correctly returns results', async () => {
            await driver.page.waitForSelector('.test-add-filter-button-file')
            await driver.page.click('.test-add-filter-button-file')
            await driver.page.waitForSelector('.filter-input__input-field')
            await driver.page.keyboard.type('README')
            await driver.page.keyboard.press('Enter')
            await driver.assertWindowLocation(
                '/search?q=error+repo:%22auth0/go-jwt-middleware%24%22+file:%22README%22&patternType=literal'
            )
            await driver.page.waitForSelector('.test-search-result')
            await driver.page.waitForFunction(() => {
                const results = document.querySelectorAll('.test-search-result')
                return results.length === 1 && (results.item(0).textContent || '').includes('README')
            })
        })

        test('Adding a language filter from the dropdown correctly returns results', async () => {
            await driver.page.waitForSelector('.add-filter-dropdown')
            await driver.page.click('.add-filter-dropdown')
            await driver.page.select('.test-filter-dropdown', 'lang')
            await driver.page.waitForSelector('.filter-input__input-field')
            await driver.page.keyboard.type('markdown')
            await driver.page.keyboard.press('Enter')
            await driver.assertWindowLocation(
                '/search?q=error+repo:%22auth0/go-jwt-middleware%24%22+file:%22README%22+lang:%22markdown%22&patternType=literal'
            )
            await driver.page.waitForSelector('.test-search-result')
            await driver.page.waitForFunction(() => {
                const results = document.querySelectorAll('.test-search-result')
                return results.length === 1 && (results.item(0).textContent || '').includes('README')
            })
        })

        test('Filters are properly parsed and displayed in the selected filters row', async () => {
            await driver.page.waitForSelector('.selected-filters-row')
            await driver.page.waitForSelector('.filter-input')
            const hasCorrectFilters = () =>
                driver.page.evaluate(() => {
                    const filterInputs = document.querySelectorAll('.filter-input')
                    const textContents: string[] = []
                    for (const filter of filterInputs) {
                        textContents.push(filter.textContent || '')
                    }

                    return (
                        textContents.length === 3 &&
                        textContents.includes('repo:"auth0/go-jwt-middleware$"') &&
                        textContents.includes('file:"README"') &&
                        textContents.includes('lang:"markdown"')
                    )
                })
            assert.strictEqual(await hasCorrectFilters(), true)
        })

        test('Landing on search results page parses query filters properly', async () => {
            await driver.page.goto(
                config.sourcegraphBaseUrl +
                    '/search?q=error+repo:auth0/go-jwt-middleware%24+file:README+lang:markdown&patternType=literal'
            )
            await driver.page.waitForSelector('.selected-filters-row')
            await driver.page.waitForSelector('.filter-input')
            const hasCorrectFilters = () =>
                driver.page.evaluate(() => {
                    const filterInputs = document.querySelectorAll('.filter-input')
                    const textContents: string[] = []
                    for (const filter of filterInputs) {
                        textContents.push(filter.textContent || '')
                    }

                    return (
                        textContents.length === 3 &&
                        textContents.includes('repo:auth0/go-jwt-middleware$') &&
                        textContents.includes('file:README') &&
                        textContents.includes('lang:markdown')
                    )
                })
            assert.strictEqual(await hasCorrectFilters(), true)
        })

        test('Going to search homepage from search results page clears all filters', async () => {
            await driver.page.click('.global-navbar__logo-link')
            await driver.assertWindowLocation('/search')
            await driver.page.waitForFunction(() => {
                const filterInputs = document.querySelectorAll('.filter-input')
                return filterInputs.length === 0
            })
        })

        test('Querying from a repository tree page produces correct query and filter values', async () => {
            await driver.page.goto(config.sourcegraphBaseUrl + '/github.com/auth0/go-jwt-middleware')
            await driver.page.waitForSelector('.query-input2 .test-query-input')
            await driver.page.click('.query-input2 .test-query-input')
            await driver.page.keyboard.type('test')
            await driver.page.keyboard.press('Enter')
            // TODO(uwedeportivo): the query string flips between "test" before or after the repo clause
            await driver.assertWindowLocation(
                '/search?q=repo:%5Egithub%5C.com/auth0/go-jwt-middleware%24+test&patternType=literal'
            )
            const hasCorrectFilters = () =>
                driver.page.evaluate(() => {
                    const filterInputs = document.querySelectorAll('.filter-input')
                    const textContents: string[] = []
                    for (const filter of filterInputs) {
                        textContents.push(filter.textContent || '')
                    }

                    return (
                        textContents.length === 1 &&
                        textContents.includes('repo:^github\\.com/auth0/go-jwt-middleware$')
                    )
                })
            assert.strictEqual(await hasCorrectFilters(), true)

            expect(
                await driver.page.evaluate(() => {
                    const queryInput = document.querySelector<HTMLInputElement>('.test-query-input')
                    return queryInput?.value === 'test'
                })
            )
        })

        test('Toggling from interactive to plain text mode produces correct query', async () => {
            await driver.page.goto(
                config.sourcegraphBaseUrl +
                    '/search?q=error+repo:auth0/go-jwt-middleware%24+file:README+lang:markdown&patternType=literal'
            )
            await driver.page.waitForSelector('.test-search-mode-toggle')
            await driver.page.click('.test-search-mode-toggle')
            await driver.page.waitForSelector('.test-search-mode-toggle__plain-text-mode')
            await driver.page.click('.test-search-mode-toggle__plain-text-mode')
            await driver.page.waitForSelector('.test-query-input')
            expect(
                await driver.page.evaluate(() => {
                    const queryInput = document.querySelector<HTMLInputElement>('.test-query-input')
                    return queryInput?.value === 'error repo:auth0/go-jwt-middleware file:README lang:markdown'
                })
            )
        })

        test('Toggling from plain text to interactive mode correctly identifies filters', async () => {
            await driver.page.waitForSelector('.test-search-mode-toggle')
            await driver.page.click('.test-search-mode-toggle')
            await driver.page.waitForSelector('.test-search-mode-toggle__interactive-mode')
            await driver.page.click('.test-search-mode-toggle__interactive-mode')
            await driver.page.waitForSelector('.test-query-input')
            expect(
                await driver.page.evaluate(() => {
                    const queryInput = document.querySelector<HTMLInputElement>('.test-query-input')
                    return queryInput?.value === 'error'
                })
            )
            await driver.page.waitForSelector('.selected-filters-row')
            const hasCorrectFilters = () =>
                driver.page.evaluate(() => {
                    const filterInputs = document.querySelectorAll('.filter-input')
                    const textContents: string[] = []
                    for (const filter of filterInputs) {
                        textContents.push(filter.textContent || '')
                    }

                    return (
                        textContents.length === 3 &&
                        textContents.includes('repo:auth0/go-jwt-middleware$') &&
                        textContents.includes('file:README') &&
                        textContents.includes('lang:markdown')
                    )
                })
            assert.strictEqual(await hasCorrectFilters(), true)
        })

        test('Filter input suggestions', async () => {
            await driver.page.goto(config.sourcegraphBaseUrl + '/search')
            await driver.page.waitForSelector('.test-add-filter-button-repo', { visible: true })
            await driver.page.click('.test-add-filter-button-repo')
            await driver.page.waitForSelector('.filter-input', { visible: true })
            await driver.page.waitForSelector('.filter-input__input-field')
            await driver.page.keyboard.type('auth0/go-jwt-middlewa')
            await driver.page.waitForSelector('.test-filter-input__suggestions')
            await driver.page.waitForSelector('.test-suggestion-item')
            await driver.page.keyboard.press('ArrowDown')
            await driver.page.keyboard.press('Enter')
            await driver.page.keyboard.press('Enter')
            await driver.assertWindowLocation(
                '/search?q=repo:%22%5Egithub%5C%5C.com/auth0/go-jwt-middleware%24%22&patternType=literal'
            )
        })

        test('Editing text filters', async () => {
            await driver.page.waitForSelector('.filter-input')
            await driver.page.click('.filter-input')
            await driver.page.waitForSelector('.filter-input__input-field')
            await driver.page.keyboard.press('Backspace')
            await driver.page.keyboard.press('Backspace')
            await driver.page.keyboard.press('Enter')
            await driver.assertWindowLocation(
                '/search?q=repo:%22%5Egithub%5C%5C.com/auth0/go-jwt-middlewar%22&patternType=literal'
            )
        })
        test('Adding and editing finite filters', async () => {
            await driver.page.goto(config.sourcegraphBaseUrl + '/search?q=test&patternType=literal')
            await driver.page.waitForSelector('.add-filter-dropdown')
            await driver.page.click('.add-filter-dropdown')
            await driver.page.select('.test-filter-dropdown', 'fork')
            await driver.page.waitForSelector('.test-filter-input-finite-form')
            await driver.page.waitForSelector('.test-filter-input-radio-button-no')
            await driver.page.click('.test-filter-input-radio-button-no')
            await driver.page.click('.test-confirm-filter-button')
            await driver.assertWindowLocation('/search?q=test+fork:%22no%22&patternType=literal')
            await driver.page.waitForSelector('.filter-input')
            await driver.page.click('.filter-input')
            await driver.page.waitForSelector('.test-filter-input-finite-form')
            await driver.page.waitForSelector('.test-filter-input-radio-button-only')
            await driver.page.click('.test-filter-input-radio-button-only')
            await driver.page.click('.test-confirm-filter-button')
            await driver.assertWindowLocation('/search?q=test+fork:%22only%22&patternType=literal')
        })
    })
})
