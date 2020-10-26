import assert from 'assert'
import { commonWebGraphQlResults } from './graphQlResults'
import { Driver, createDriverForTest, percySnapshot } from '../../../shared/src/testing/driver'
import { ExtensionManifest } from '../../../shared/src/schema/extensionSchema'
import { WebIntegrationTestContext, createWebIntegrationTestContext } from './context'
import {
    createRepositoryRedirectResult,
    createResolveRevisionResult,
    createFileExternalLinksResult,
    createTreeEntriesResult,
    createBlobContentResult,
} from './graphQlResponseHelpers'
import { WebGraphQlOperations } from '../graphql-operations'
import { SharedGraphQlOperations } from '../../../shared/src/graphql-operations'
import { Settings } from '../schema/settings.schema'
import type * as sourcegraph from 'sourcegraph'
import { afterEachSaveScreenshotIfFailed } from '../../../shared/src/testing/screenshotReporter'
import { Page } from 'puppeteer'

describe('Blob viewer', () => {
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

    const repositoryName = 'github.com/sourcegraph/jsonrpc2'
    const repositorySourcegraphUrl = `/${repositoryName}`
    const fileName = 'test.ts'
    const files = ['README.md', fileName]

    const commonBlobGraphQlResults: Partial<WebGraphQlOperations & SharedGraphQlOperations> = {
        ...commonWebGraphQlResults,
        RepositoryRedirect: ({ repoName }) => createRepositoryRedirectResult(repoName),
        ResolveRev: () => createResolveRevisionResult(repositorySourcegraphUrl),
        FileExternalLinks: ({ filePath }) =>
            createFileExternalLinksResult(`https://${repositoryName}/blob/master/${filePath}`),
        TreeEntries: () => createTreeEntriesResult(repositorySourcegraphUrl, ['README.md', fileName]),
        Blob: ({ filePath }) => createBlobContentResult(`content for: ${filePath}\nsecond line\nthird line`),
    }
    beforeEach(() => {
        testContext.overrideGraphQL(commonBlobGraphQlResults)
    })

    describe('general layout for viewing a file', () => {
        it('populates editor content and FILES tab', async () => {
            await driver.page.goto(`${driver.sourcegraphBaseUrl}/${repositoryName}/-/blob/${fileName}`)
            await driver.page.waitForSelector('.test-repo-blob')
            const blobContent = await driver.page.evaluate(
                () => document.querySelector<HTMLElement>('.test-repo-blob')?.textContent
            )

            // editor shows the return string content from Blob request
            assert.strictEqual(blobContent, `content for: ${fileName}\nsecond line\nthird line`)

            // collect all files/links visible the the "Files" tab
            const allFilesInTheTree = await driver.page.evaluate(() => {
                const allFiles = document.querySelectorAll<HTMLAnchorElement>('.test-tree-file-link')

                return [...allFiles].map(fileAnchor => ({
                    content: fileAnchor.textContent,
                    href: fileAnchor.href,
                }))
            })

            // files from TreeEntries request
            assert.deepStrictEqual(
                allFilesInTheTree,
                files.map(name => ({
                    content: name,
                    href: `${driver.sourcegraphBaseUrl}/${repositoryName}/-/blob/${name}`,
                }))
            )
        })
    })

    // Describes the ways the blob viewer can be extended through Sourcegraph extensions.
    describe('extensibility', () => {
        const getHoverContents = async (): Promise<string[]> => {
            // Search for any child of e2e-tooltip-content: as e2e-tooltip-content has display: contents,
            // it will never be detected as visible by waitForSelector(), but its children will.
            await driver.page.waitForSelector('.test-tooltip-content *', { visible: true })
            return driver.page.evaluate(() =>
                [...document.querySelectorAll('.test-tooltip-content')].map(content => content.textContent ?? '')
            )
        }

        beforeEach(() => {
            const userSettings: Settings = {
                extensions: {
                    'test/test': true,
                },
            }
            const extensionManifest: ExtensionManifest = {
                url: new URL('/-/static/extension/0001-test-test.js?hash--test-test', driver.sourcegraphBaseUrl).href,
                activationEvents: ['*'],
            }
            testContext.overrideGraphQL({
                ...commonBlobGraphQlResults,
                ViewerSettings: () => ({
                    viewerSettings: {
                        final: JSON.stringify(userSettings),
                        subjects: [
                            {
                                __typename: 'User',
                                displayName: 'Test User',
                                id: 'TestUserSettingsID',
                                latestSettings: {
                                    id: 123,
                                    contents: JSON.stringify(userSettings),
                                },
                                username: 'test',
                                viewerCanAdminister: true,
                                settingsURL: '/users/test/settings',
                            },
                        ],
                    },
                }),
                Blob: () => ({
                    repository: {
                        commit: {
                            file: {
                                content: '// Log to console\nconsole.log("Hello world")',
                                richHTML: '',
                                highlight: {
                                    aborted: false,
                                    html:
                                        // Note: whitespace in this string is significant.
                                        '<table><tbody><tr><td class="line" data-line="1"></td><td class="code"><div><span style="color: gray">&sol;&sol; Log to console\n' +
                                        '</span></div></td></tr><tr><td class="line" data-line="2"></td><td class="code"><div><span style="color: #859900;" class="test-console-token">console</span><span style="color: #657b83;">.</span><span style="color: #859900;" class="test-log-token">log</span><span style="color: #657b83;">(</span><span style="color: #839496;">&quot;</span><span style="color: #2aa198;">Hello world</span><span style="color: #839496;">&quot;</span><span style="color: #657b83;">)\n' +
                                        '</span></div></td></tr></tbody></table>',
                                },
                            },
                        },
                    },
                }),
                Extensions: () => ({
                    extensionRegistry: {
                        extensions: {
                            nodes: [
                                {
                                    id: 'TestExtensionID',
                                    extensionID: 'test/test',
                                    manifest: {
                                        raw: JSON.stringify(extensionManifest),
                                    },
                                    url: '/extensions/test/test',
                                    viewerCanAdminister: false,
                                },
                            ],
                        },
                    },
                }),
            })

            // Serve a mock extension bundle with a simple hover provider
            testContext.server
                .get(new URL(extensionManifest.url, driver.sourcegraphBaseUrl).href)
                .intercept((request, response) => {
                    function extensionBundle(): void {
                        // eslint-disable-next-line @typescript-eslint/no-require-imports, @typescript-eslint/no-var-requires
                        const sourcegraph = require('sourcegraph') as typeof import('sourcegraph')

                        function activate(context: sourcegraph.ExtensionContext): void {
                            context.subscriptions.add(
                                sourcegraph.languages.registerHoverProvider([{ language: 'typescript' }], {
                                    provideHover: () => ({
                                        contents: {
                                            kind: sourcegraph.MarkupKind.Markdown,
                                            value: 'Test hover content',
                                        },
                                    }),
                                })
                            )
                        }

                        exports.activate = activate
                    }
                    // Create an immediately-invoked function expression for the extensionBundle function
                    const extensionBundleString = `(${extensionBundle.toString()})()`
                    response.type('application/javascript; charset=utf-8').send(extensionBundleString)
                })
        })
        it('truncates long file paths properly', async function () {
            await driver.page.goto(
                `${driver.sourcegraphBaseUrl}/${repositoryName}/-/blob/this_is_a_long_file_path/apps/rest-showcase/src/main/java/org/demo/rest/example/OrdersController.java`
            )
            await driver.page.waitForSelector('.test-repo-blob')
            await driver.page.waitForSelector('.test-breadcrumb')
            await percySnapshot(driver.page, this.test!.fullTitle())
        })

        it.skip('shows a hover overlay from a hover provider when a token is hovered', async () => {
            await driver.page.goto(`${driver.sourcegraphBaseUrl}/${repositoryName}/-/blob/${fileName}`)
            await driver.page.waitForSelector('.test-repo-blob')
            // TODO
        })

        it('shows a hover overlay from a hover provider and updates the URL when a token is clicked', async function () {
            await driver.page.goto(`${driver.sourcegraphBaseUrl}/github.com/sourcegraph/test/-/blob/test.ts`)

            // Click on "log" in "console.log()" in line 2
            await driver.page.waitForSelector('.test-log-token', { visible: true })
            await driver.page.click('.test-log-token')

            await driver.assertWindowLocation('/github.com/sourcegraph/test/-/blob/test.ts#L2:9')
            assert.deepStrictEqual(await getHoverContents(), ['Test hover content\n'])
            await percySnapshot(driver.page, this.test!.fullTitle())
        })

        it.skip('gets displayed when navigating to a URL with a token position', async () => {
            await driver.page.goto(`${driver.sourcegraphBaseUrl}/github.com/sourcegraph/test/-/test.ts#2:9`)
            // TODO
        })

        describe('browser extension discoverability', () => {
            const HOVER_THRESHOLD = 5
            const HOVER_COUNT_KEY = 'hover-count'

            it(`shows a popover about the browser extension when the user has seen ${HOVER_THRESHOLD} hovers and clicks "View on [code host]" button`, async () => {
                await driver.page.goto(`${driver.sourcegraphBaseUrl}/github.com/sourcegraph/test/-/blob/test.ts`)
                await driver.page.evaluate(() => localStorage.removeItem('hover-count'))
                await driver.page.reload()

                await driver.page.waitForSelector('.test-go-to-code-host', { visible: true })
                // Close new tab after clicking link
                const newPage = new Promise<Page>(resolve =>
                    driver.browser.once('targetcreated', target => resolve(target.page()))
                )
                await driver.page.click('.test-go-to-code-host', { button: 'middle' })
                await (await newPage).close()

                assert(
                    !(await driver.page.$('.test-install-browser-extension-popover')),
                    'Expected popover to not be displayed before user reaches hover threshold'
                )

                // Click 'console' and 'log' 5 times combined
                await driver.page.waitForSelector('.test-log-token', { visible: true })
                for (let index = 0; index < HOVER_THRESHOLD; index++) {
                    await driver.page.click(index % 2 === 0 ? '.test-log-token' : '.test-console-token')
                    await driver.page.waitForSelector('.hover-overlay', { visible: true })
                }

                await driver.page.click('.test-go-to-code-host', { button: 'middle' })
                await driver.page.waitForSelector('.test-install-browser-extension-popover', { visible: true })
                assert(
                    !!(await driver.page.$('.test-install-browser-extension-popover')),
                    'Expected popover to be displayed after user reaches hover threshold'
                )

                const popoverHeader = await driver.page.evaluate(
                    () => document.querySelector('.test-install-browser-extension-popover-header')?.textContent
                )
                assert.strictEqual(
                    popoverHeader,
                    "Take Sourcegraph's code intelligence to GitHub!",
                    'Expected popover header text to reflect code host'
                )
            })

            it(`shows an alert about the browser extension when the user has seen ${HOVER_THRESHOLD} hovers`, async () => {
                await driver.page.goto(`${driver.sourcegraphBaseUrl}/github.com/sourcegraph/test/-/blob/test.ts`)
                await driver.page.evaluate(HOVER_COUNT_KEY => localStorage.removeItem(HOVER_COUNT_KEY), HOVER_COUNT_KEY)
                await driver.page.reload()

                // Alert should not be visible before the user reaches the hover threshold
                assert(
                    !(await driver.page.$('.install-browser-extension-alert')),
                    'Expected "Install browser extension" alert to not be displayed before user reaches hover threshold'
                )

                // Click 'console' and 'log' $HOVER_THRESHOLD times combined
                await driver.page.waitForSelector('.test-log-token', { visible: true })
                for (let index = 0; index < HOVER_THRESHOLD; index++) {
                    await driver.page.click(index % 2 === 0 ? '.test-log-token' : '.test-console-token')
                    await driver.page.waitForSelector('.hover-overlay', { visible: true })
                }
                await driver.page.reload()

                // Alert should be visible now that the user has seen $HOVER_THRESHOLD hovers
                await driver.page.waitForSelector('.install-browser-extension-alert', { timeout: 5000 })

                // Dismiss alert
                await driver.page.click('.test-close-alert')
                await driver.page.reload()

                // Alert should not show up now that the user has dismissed it once
                await driver.page.waitForSelector('.repo-header')
                // `browserExtensionInstalled` emits false after 500ms, so
                // wait 500ms after .repo-header is visible, at which point we know
                // that `RepoContainer` has subscribed to `browserExtensionInstalled`.
                // After this point, we know whether or not the alert will be displayed for this page load.
                await driver.page.waitFor(500)
                assert(
                    !(await driver.page.$('.install-browser-extension-alert')),
                    'Expected "Install browser extension" alert to not be displayed before user dismisses it once'
                )
            })
        })
    })
})
