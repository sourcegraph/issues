import expect from 'expect'
import { sortBy } from 'lodash'
import { Driver, createDriverForTest, percySnapshot } from '../../../shared/src/testing/driver'
import { retry } from '../../../shared/src/testing/utils'
import { afterEachSaveScreenshotIfFailed } from '../../../shared/src/testing/screenshotReporter'
import { WebIntegrationTestContext, createWebIntegrationTestContext } from './context'
import { test } from 'mocha'

describe('Repository component', () => {
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
    const blobTableSelector = '.test-blob > table'
    /**
     * @param line 1-indexed line number
     * @param spanOffset 1-indexed index of the span that's to be clicked
     */
    const clickToken = async (line: number, spanOffset: number): Promise<void> => {
        const selector = `${blobTableSelector} tr:nth-child(${line}) > td.code > div:nth-child(1) > span:nth-child(${spanOffset})`
        await driver.page.waitForSelector(selector, { visible: true })
        await driver.page.click(selector)
    }

    const getHoverContents = async (): Promise<string[]> => {
        // Search for any child of test-tooltip-content: as test-tooltip-content has display: contents,
        // it will never be detected as visible by waitForSelector(), but its children will.
        const selector = '.test-tooltip-content *'
        await driver.page.waitForSelector(selector, { visible: true })
        return driver.page.evaluate(() =>
            // You can't reference hoverContentSelector in puppeteer's driver.page.evaluate
            [...document.querySelectorAll('.test-tooltip-content')].map(content => content.textContent || '')
        )
    }
    const assertHoverContentContains = async (value: string): Promise<void> => {
        expect(await getHoverContents()).toEqual(expect.arrayContaining([expect.stringContaining(value)]))
    }

    const clickHoverJ2D = async (): Promise<void> => {
        const selector = '.test-tooltip-go-to-definition'
        await driver.page.waitForSelector(selector, { visible: true })
        await driver.page.click(selector)
    }
    const clickHoverFindReferences = async (): Promise<void> => {
        const selector = '.test-tooltip-find-references'
        await driver.page.waitForSelector(selector, { visible: true })
        await driver.page.click(selector)
    }

    describe('file tree', () => {
        test('does navigation on file click', async () => {
            await driver.page.goto(
                driver.sourcegraphBaseUrl + '/github.com/sourcegraph/jsonrpc2@c6c7b9aa99fb76ee5460ccd3912ba35d419d493d'
            )
            await (
                await driver.page.waitForSelector('[data-tree-path="async.go"]', {
                    visible: true,
                })
            ).click()
            await driver.assertWindowLocation(
                '/github.com/sourcegraph/jsonrpc2@c6c7b9aa99fb76ee5460ccd3912ba35d419d493d/-/blob/async.go'
            )
        })

        test('expands directory on row click (no navigation)', async () => {
            await driver.page.goto(
                driver.sourcegraphBaseUrl + '/github.com/sourcegraph/jsonrpc2@c6c7b9aa99fb76ee5460ccd3912ba35d419d493d'
            )
            await driver.page.waitForSelector('.tree__row-icon', { visible: true })
            await driver.page.click('.tree__row-icon')
            await driver.page.waitForSelector('.tree__row--selected [data-tree-path="websocket"]', {
                visible: true,
            })
            await driver.page.waitForSelector('.tree__row--expanded [data-tree-path="websocket"]', {
                visible: true,
            })
            await driver.assertWindowLocation(
                '/github.com/sourcegraph/jsonrpc2@c6c7b9aa99fb76ee5460ccd3912ba35d419d493d'
            )
        })

        test('does navigation on directory row click', async () => {
            await driver.page.goto(
                driver.sourcegraphBaseUrl + '/github.com/sourcegraph/jsonrpc2@c6c7b9aa99fb76ee5460ccd3912ba35d419d493d'
            )
            await driver.page.waitForSelector('.tree__row-label', { visible: true })
            await driver.page.click('.tree__row-label')
            await driver.page.waitForSelector('.tree__row--selected [data-tree-path="websocket"]', {
                visible: true,
            })
            await driver.page.waitForSelector('.tree__row--expanded [data-tree-path="websocket"]', {
                visible: true,
            })
            await driver.assertWindowLocation(
                '/github.com/sourcegraph/jsonrpc2@c6c7b9aa99fb76ee5460ccd3912ba35d419d493d/-/tree/websocket'
            )
        })

        test('selects the current file', async () => {
            await driver.page.goto(
                driver.sourcegraphBaseUrl +
                    '/github.com/sourcegraph/jsonrpc2@c6c7b9aa99fb76ee5460ccd3912ba35d419d493d/-/blob/async.go'
            )
            await driver.page.waitForSelector('.tree__row--active [data-tree-path="async.go"]', {
                visible: true,
            })
        })

        test('shows partial tree when opening directory', async () => {
            await driver.page.goto(
                driver.sourcegraphBaseUrl +
                    '/github.com/sourcegraph/jsonrpc2@c6c7b9aa99fb76ee5460ccd3912ba35d419d493d/-/tree/websocket'
            )
            await driver.page.waitForSelector('.tree__row', { visible: true })
            expect(await driver.page.evaluate(() => document.querySelectorAll('.tree__row').length)).toEqual(1)
        })

        test('responds to keyboard shortcuts', async () => {
            const assertNumberRowsExpanded = async (expectedCount: number): Promise<void> => {
                expect(
                    await driver.page.evaluate(() => document.querySelectorAll('.tree__row--expanded').length)
                ).toEqual(expectedCount)
            }
            await driver.page.goto(
                driver.sourcegraphBaseUrl +
                    '/github.com/sourcegraph/go-diff@3f415a150aec0685cb81b73cc201e762e075006d/-/blob/.travis.yml'
            )
            await driver.page.waitForSelector('.tree__row', { visible: true }) // waitForSelector for tree to render

            await driver.page.click('.test-repo-revision-sidebar .tree')
            await driver.page.keyboard.press('ArrowUp') // arrow up to 'diff' directory
            await driver.page.waitForSelector('.tree__row--selected [data-tree-path="diff"]', { visible: true })
            await driver.page.keyboard.press('ArrowRight') // arrow right (expand 'diff' directory)
            await driver.page.waitForSelector('.tree__row--selected [data-tree-path="diff"]', { visible: true })
            await driver.page.waitForSelector('.tree__row--expanded [data-tree-path="diff"]', { visible: true })
            await driver.page.waitForSelector('.tree__row [data-tree-path="diff/testdata"]', { visible: true })
            await driver.page.keyboard.press('ArrowRight') // arrow right (move to nested 'diff/testdata' directory)
            await driver.page.waitForSelector('.tree__row--selected [data-tree-path="diff/testdata"]', {
                visible: true,
            })
            await assertNumberRowsExpanded(1) // only `diff` directory is expanded, though `diff/testdata` is expanded

            await driver.page.keyboard.press('ArrowRight') // arrow right (expand 'diff/testdata' directory)
            await driver.page.waitForSelector('.tree__row--selected [data-tree-path="diff/testdata"]', {
                visible: true,
            })
            await driver.page.waitForSelector('.tree__row--expanded [data-tree-path="diff/testdata"]', {
                visible: true,
            })
            await assertNumberRowsExpanded(2) // `diff` and `diff/testdata` directories expanded

            await driver.page.waitForSelector('.tree__row [data-tree-path="diff/testdata/empty.diff"]', {
                visible: true,
            })
            // select some file nested under `diff/testdata`
            await driver.page.keyboard.press('ArrowDown') // arrow down
            await driver.page.keyboard.press('ArrowDown') // arrow down
            await driver.page.keyboard.press('ArrowDown') // arrow down
            await driver.page.keyboard.press('ArrowDown') // arrow down
            await driver.page.waitForSelector('.tree__row--selected [data-tree-path="diff/testdata/empty_orig.diff"]', {
                visible: true,
            })

            await driver.page.keyboard.press('ArrowLeft') // arrow left (navigate immediately up to parent directory `diff/testdata`)
            await driver.page.waitForSelector('.tree__row--selected [data-tree-path="diff/testdata"]', {
                visible: true,
            })
            await assertNumberRowsExpanded(2) // `diff` and `diff/testdata` directories expanded

            await driver.page.keyboard.press('ArrowLeft') // arrow left
            await driver.page.waitForSelector('.tree__row--selected [data-tree-path="diff/testdata"]', {
                visible: true,
            }) // `diff/testdata` still selected
            await assertNumberRowsExpanded(1) // only `diff` directory expanded
        })
    })
    describe('symbol sidebar', () => {
        const listSymbolsTests = [
            {
                name: 'lists symbols in file for Go',
                filePath:
                    '/github.com/sourcegraph/go-diff@3f415a150aec0685cb81b73cc201e762e075006d/-/blob/cmd/go-diff/go-diff.go',
                symbolNames: ['main', 'stdin', 'diffPath', 'fileIdx', 'main'],
                symbolTypes: ['package', 'constant', 'variable', 'variable', 'function'],
            },
            {
                name: 'lists symbols in another file for Go',
                filePath:
                    '/github.com/sourcegraph/go-diff@3f415a150aec0685cb81b73cc201e762e075006d/-/blob/diff/diff.go',
                symbolNames: [
                    'diff',
                    'Stat',
                    'Stat',
                    'hunkPrefix',
                    'hunkHeader',
                    'diffTimeParseLayout',
                    'diffTimeFormatLayout',
                    'add',
                ],
                symbolTypes: [
                    'package',
                    'function',
                    'function',
                    'variable',
                    'constant',
                    'constant',
                    'constant',
                    'function',
                ],
            },
            {
                name: 'lists symbols in file for Python',
                filePath:
                    '/github.com/sourcegraph/appdash@ebfcffb1b5c00031ce797183546746715a3cfe87/-/blob/python/appdash/sockcollector.py',
                symbolNames: ['RemoteCollector', 'sock', '_debug', '__init__', '_log', 'connect', 'collect', 'close'],
                symbolTypes: ['class', 'variable', 'variable', 'field', 'field', 'field', 'field', 'field'],
            },
            {
                name: 'lists symbols in file for TypeScript',
                filePath:
                    '/github.com/sourcegraph/sourcegraph-typescript@a7b7a61e31af76dad3543adec359fa68737a58a1/-/blob/server/src/cancellation.ts',
                symbolNames: [
                    'createAbortError',
                    'isAbortError',
                    'throwIfCancelled',
                    'tryCancel',
                    'toAxiosCancelToken',
                    'source',
                ],
                symbolTypes: ['constant', 'constant', 'function', 'function', 'function', 'constant'],
            },
            {
                name: 'lists symbols in file for Java',
                filePath:
                    '/github.com/sourcegraph/java-langserver@03efbe9558acc532e88f5288b4e6cfa155c6f2dc/-/blob/src/main/java/com/sourcegraph/common/Config.java',
                symbolNames: [
                    'com.sourcegraph.common',
                    'Config',
                    'LIGHTSTEP_INCLUDE_SENSITIVE',
                    'LIGHTSTEP_PROJECT',
                    'LIGHTSTEP_TOKEN',
                    'ANDROID_JAR_PATH',
                    'IGNORE_DEPENDENCY_RESOLUTION_CACHE',
                    'LSP_TIMEOUT',
                    'LANGSERVER_ROOT',
                    'LOCAL_REPOSITORY',
                    'EXECUTE_GRADLE_ORIGINAL_ROOT_PATHS',
                    'shouldExecuteGradle',
                    'PRIVATE_REPO_ID',
                    'PRIVATE_REPO_URL',
                    'PRIVATE_REPO_USERNAME',
                    'PRIVATE_REPO_PASSWORD',
                    'log',
                    'checkEnv',
                    'ConfigException',
                ],
                symbolTypes: [
                    'package',
                    'class',
                    'field',
                    'field',
                    'field',
                    'field',
                    'field',
                    'field',
                    'field',
                    'field',
                    'field',
                    'method',
                    'field',
                    'field',
                    'field',
                    'field',
                    'field',
                    'method',
                    'class',
                ],
            },
        ]

        for (const symbolTest of listSymbolsTests) {
            test(symbolTest.name, async () => {
                await driver.page.goto(driver.sourcegraphBaseUrl + symbolTest.filePath)

                await (await driver.page.waitForSelector('[data-test-tab="symbols"]')).click()

                await driver.page.waitForSelector('.test-symbol-name', { visible: true })

                const symbolNames = await driver.page.evaluate(() =>
                    [...document.querySelectorAll('.test-symbol-name')].map(name => name.textContent || '')
                )
                const symbolTypes = await driver.page.evaluate(() =>
                    [...document.querySelectorAll('.test-symbol-icon')].map(
                        icon => icon.getAttribute('data-tooltip') || ''
                    )
                )

                expect(sortBy(symbolNames)).toEqual(sortBy(symbolTest.symbolNames))
                expect(sortBy(symbolTypes)).toEqual(sortBy(symbolTest.symbolTypes))
            })
        }

        const navigateToSymbolTests = [
            {
                name: 'navigates to file on symbol click for Go',
                repoPath: '/github.com/sourcegraph/go-diff@3f415a150aec0685cb81b73cc201e762e075006d',
                filePath: '/tree/cmd',
                symbolPath: '/blob/cmd/go-diff/go-diff.go#L19:2-19:10',
            },
            {
                name: 'navigates to file on symbol click for Java',
                repoPath: '/github.com/sourcegraph/java-langserver@03efbe9558acc532e88f5288b4e6cfa155c6f2dc',
                filePath: '/tree/src/main/java/com/sourcegraph/common',
                symbolPath: '/blob/src/main/java/com/sourcegraph/common/Config.java#L14:20-14:26',
                skip: true,
            },
            {
                name:
                    'displays valid symbols at different file depths for Go (./examples/cmd/webapp-opentracing/main.go.go)',
                repoPath: '/github.com/sourcegraph/appdash@ebfcffb1b5c00031ce797183546746715a3cfe87',
                filePath: '/tree/examples',
                symbolPath: '/blob/examples/cmd/webapp-opentracing/main.go#L26:6-26:10',
                skip: true,
            },
            {
                name: 'displays valid symbols at different file depths for Go (./sqltrace/sql.go)',
                repoPath: '/github.com/sourcegraph/appdash@ebfcffb1b5c00031ce797183546746715a3cfe87',
                filePath: '/tree/sqltrace',
                symbolPath: '/blob/sqltrace/sql.go#L14:2-14:5',
                skip: true,
            },
        ]

        for (const navigationTest of navigateToSymbolTests) {
            const testFunc = navigationTest.skip ? test.skip : test
            testFunc(navigationTest.name, async () => {
                const repoBaseURL = driver.sourcegraphBaseUrl + navigationTest.repoPath + '/-'

                await driver.page.goto(repoBaseURL + navigationTest.filePath)

                await (await driver.page.waitForSelector('[data-test-tab="symbols"]')).click()

                await driver.page.waitForSelector('.test-symbol-name', { visible: true })

                await (
                    await driver.page.waitForSelector(`.test-symbol-link[href*="${navigationTest.symbolPath}"]`, {
                        visible: true,
                    })
                ).click()
                await driver.assertWindowLocation(repoBaseURL + navigationTest.symbolPath, true)
            })
        }

        const highlightSymbolTests = [
            {
                name: 'highlights correct line for Go',
                filePath:
                    '/github.com/sourcegraph/go-diff@3f415a150aec0685cb81b73cc201e762e075006d/-/blob/diff/diff.go',
                index: 5,
                line: 65,
            },
            {
                name: 'highlights correct line for TypeScript',
                filePath:
                    '/github.com/sourcegraph/sourcegraph-typescript@a7b7a61e31af76dad3543adec359fa68737a58a1/-/blob/server/src/cancellation.ts',
                index: 2,
                line: 17,
            },
        ]

        for (const { name, filePath, index, line } of highlightSymbolTests) {
            test(name, async () => {
                await driver.page.goto(driver.sourcegraphBaseUrl + filePath)
                await driver.page.waitForSelector('[data-test-tab="symbols"]')
                await driver.page.click('[data-test-tab="symbols"]')
                await driver.page.waitForSelector('.test-symbol-name', { visible: true })
                await driver.page.click(`.filtered-connection__nodes li:nth-child(${index + 1}) a`)

                await driver.page.waitForSelector('.test-blob .selected .line')
                const selectedLineNumber = await driver.page.evaluate(() => {
                    const element = document.querySelector<HTMLElement>('.test-blob .selected .line')
                    return element?.dataset.line && parseInt(element.dataset.line, 10)
                })

                expect(selectedLineNumber).toEqual(line)
            })
        }
    })

    describe('revision resolution', () => {
        test('shows clone in progress interstitial page', async () => {
            await driver.page.goto(driver.sourcegraphBaseUrl + '/github.com/sourcegraphtest/AlwaysCloningTest')
            await driver.page.waitForSelector('.hero-page__subtitle', { visible: true })
            await retry(async () =>
                expect(
                    await driver.page.evaluate(() => document.querySelector('.hero-page__subtitle')!.textContent)
                ).toEqual('Cloning in progress')
            )
        })

        test('resolves default branch when unspecified', async () => {
            await driver.page.goto(driver.sourcegraphBaseUrl + '/github.com/sourcegraph/go-diff/-/blob/diff/diff.go')
            await driver.page.waitForSelector('#repo-revision-popover', { visible: true })
            await retry(async () => {
                expect(
                    await driver.page.evaluate(() => document.querySelector('.test-revision')!.textContent!.trim())
                ).toEqual('master')
            })
            // Verify file contents are loaded.
            await driver.page.waitForSelector(blobTableSelector)
        })

        test('updates revision with switcher', async () => {
            await driver.page.goto(driver.sourcegraphBaseUrl + '/github.com/sourcegraph/go-diff/-/blob/diff/diff.go')
            // Open revision switcher
            await driver.page.waitForSelector('#repo-revision-popover', { visible: true })
            await driver.page.click('#repo-revision-popover')
            // Click "Tags" tab
            await driver.page.click('.revisions-popover .tab-bar__tab:nth-child(2)')
            await driver.page.waitForSelector('a.git-ref-node[href*="0.5.0"]', { visible: true })
            await driver.page.click('a.git-ref-node[href*="0.5.0"]')
            await driver.assertWindowLocation('/github.com/sourcegraph/go-diff@v0.5.0/-/blob/diff/diff.go')
        })
    })

    describe('hovers', () => {
        describe('Blob', () => {
            test('gets displayed and updates URL when clicking on a token', async () => {
                await driver.page.goto(
                    driver.sourcegraphBaseUrl +
                        '/github.com/gorilla/mux@15a353a636720571d19e37b34a14499c3afa9991/-/blob/mux.go'
                )
                await driver.page.waitForSelector(blobTableSelector)
                await clickToken(24, 5)
                await driver.assertWindowLocation(
                    '/github.com/gorilla/mux@15a353a636720571d19e37b34a14499c3afa9991/-/blob/mux.go#L24:19'
                )
                await getHoverContents() // verify there is a hover
                await percySnapshot(driver.page, 'Code intel hover tooltip')
            })

            test('gets displayed when navigating to a URL with a token position', async () => {
                await driver.page.goto(
                    driver.sourcegraphBaseUrl +
                        '/github.com/gorilla/mux@15a353a636720571d19e37b34a14499c3afa9991/-/blob/mux.go#L151:23'
                )
                await assertHoverContentContains(
                    'ErrMethodMismatch is returned when the method in the request does not match'
                )
            })

            describe('jump to definition', () => {
                test('noops when on the definition', async () => {
                    await driver.page.goto(
                        driver.sourcegraphBaseUrl +
                            '/github.com/sourcegraph/go-diff@3f415a150aec0685cb81b73cc201e762e075006d/-/blob/diff/parse.go#L29:6'
                    )
                    await clickHoverJ2D()
                    await driver.assertWindowLocation(
                        '/github.com/sourcegraph/go-diff@3f415a150aec0685cb81b73cc201e762e075006d/-/blob/diff/parse.go#L29:6'
                    )
                })

                test('does navigation (same repo, same file)', async () => {
                    await driver.page.goto(
                        driver.sourcegraphBaseUrl +
                            '/github.com/sourcegraph/go-diff@3f415a150aec0685cb81b73cc201e762e075006d/-/blob/diff/parse.go#L25:10'
                    )
                    await clickHoverJ2D()
                    await driver.assertWindowLocation(
                        '/github.com/sourcegraph/go-diff@3f415a150aec0685cb81b73cc201e762e075006d/-/blob/diff/parse.go#L29:6'
                    )
                })

                test('does navigation (same repo, different file)', async () => {
                    await driver.page.goto(
                        driver.sourcegraphBaseUrl +
                            '/github.com/sourcegraph/go-diff@3f415a150aec0685cb81b73cc201e762e075006d/-/blob/diff/print.go#L13:31'
                    )
                    await clickHoverJ2D()
                    await driver.assertWindowLocation(
                        '/github.com/sourcegraph/go-diff@3f415a150aec0685cb81b73cc201e762e075006d/-/blob/diff/diff.pb.go#L38:6'
                    )
                    // Verify file tree is highlighting the new path.
                    await driver.page.waitForSelector('.tree__row--active [data-tree-path="diff/diff.pb.go"]', {
                        visible: true,
                    })
                })
            })

            describe('find references', () => {
                test('opens widget and fetches local references', async function () {
                    this.timeout(120000)

                    await driver.page.goto(
                        driver.sourcegraphBaseUrl +
                            '/github.com/sourcegraph/go-diff@3f415a150aec0685cb81b73cc201e762e075006d/-/blob/diff/parse.go#L29:6'
                    )
                    await clickHoverFindReferences()
                    await driver.assertWindowLocation(
                        '/github.com/sourcegraph/go-diff@3f415a150aec0685cb81b73cc201e762e075006d/-/blob/diff/parse.go#L29:6&tab=references'
                    )

                    await driver.assertNonemptyLocalRefs()

                    // verify the appropriate # of references are fetched
                    await driver.page.waitForSelector('.panel__tabs-content .file-match-children', {
                        visible: true,
                    })
                    await retry(async () =>
                        expect(
                            await driver.page.evaluate(
                                () =>
                                    document.querySelectorAll('.panel__tabs-content .file-match-children__item').length
                            )
                        ).toEqual(
                            // Basic code intel finds 8 references with some overlapping context, resulting in 4 hunks.
                            4
                        )
                    )

                    // verify all the matches highlight a `MultiFileDiffReader` token
                    await driver.assertAllHighlightedTokens('MultiFileDiffReader')
                })
            })
        })
    })
})
