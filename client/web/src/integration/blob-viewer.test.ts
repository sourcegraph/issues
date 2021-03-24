import assert from 'assert'
import { commonWebGraphQlResults } from './graphQlResults'
import { Driver, createDriverForTest } from '../../../shared/src/testing/driver'
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
        it('truncates long file paths properly', async () => {
            await driver.page.goto(
                `${driver.sourcegraphBaseUrl}/${repositoryName}/-/blob/this_is_a_long_file_path/apps/rest-showcase/src/main/java/org/demo/rest/example/OrdersController.java`
            )
            await driver.page.waitForSelector('.test-repo-blob')
            await driver.page.waitForSelector('.test-breadcrumb')
            // Uncomment this snapshot once https://github.com/sourcegraph/sourcegraph/issues/15126 is resolved
            // await percySnapshot(driver.page, this.test!.fullTitle())
        })

        it.skip('shows a hover overlay from a hover provider when a token is hovered', async () => {
            await driver.page.goto(`${driver.sourcegraphBaseUrl}/${repositoryName}/-/blob/${fileName}`)
            await driver.page.waitForSelector('.test-repo-blob')
            // TODO
        })

        it('shows a hover overlay from a hover provider and updates the URL when a token is clicked', async () => {
            await driver.page.goto(`${driver.sourcegraphBaseUrl}/github.com/sourcegraph/test/-/blob/test.ts`)

            // Click on "log" in "console.log()" in line 2
            await driver.page.waitForSelector('.test-log-token', { visible: true })
            await driver.page.click('.test-log-token')

            await driver.assertWindowLocation('/github.com/sourcegraph/test/-/blob/test.ts#L2:9')
            assert.deepStrictEqual(await getHoverContents(), ['Test hover content\n'])
            // Uncomment this snapshot once https://github.com/sourcegraph/sourcegraph/issues/15126 is resolved
            // await percySnapshot(driver.page, this.test!.fullTitle())
        })

        interface MockExtension {
            id: string
            extensionID: string
            extensionManifest: ExtensionManifest
            /** The URL of the JavaScript bundle */
            url: string
            viewerCanAdminister: boolean
            /**
             * A function whose body is a Sourcegraph extension.
             *
             * Bundle must import 'sourcegraph' (e.g. `const sourcegraph = require('sourcegraph')`)
             * */
            bundle: () => void
        }

        /**
         * This test is meant to prevent regression: https://github.com/sourcegraph/sourcegraph/pull/15099
         */
        it('adds and clears line decoration attachments properly', async () => {
            const mockExtensions: MockExtension[] = [
                {
                    id: 'TestFixedLineID',
                    extensionID: 'test/fixed-line',
                    extensionManifest: {
                        url: new URL(
                            '/-/static/extension/0001-test-fixed-line.js?hash--test-fixed-line',
                            driver.sourcegraphBaseUrl
                        ).href,
                        activationEvents: ['*'],
                    },
                    url: '/extensions/test/fixed-line',
                    viewerCanAdminister: false,
                    bundle: function extensionBundle(): void {
                        // eslint-disable-next-line @typescript-eslint/no-require-imports, @typescript-eslint/no-var-requires
                        const sourcegraph = require('sourcegraph') as typeof import('sourcegraph')

                        const fixedLineDecorationType = sourcegraph.app.createDecorationType()

                        function decorateLineTwo(editor: sourcegraph.CodeEditor) {
                            // Always decorate line 2
                            editor.setDecorations(fixedLineDecorationType, [
                                {
                                    range: new sourcegraph.Range(
                                        new sourcegraph.Position(1, 0),
                                        new sourcegraph.Position(1, 0)
                                    ),
                                    after: {
                                        contentText: 'fixed line content',
                                        backgroundColor: 'red',
                                    },
                                },
                            ])
                        }

                        function activate(context: sourcegraph.ExtensionContext): void {
                            // check initial viewer in case it was already emitted before extension was activated
                            const initialViewer = sourcegraph.app.activeWindow?.activeViewComponent
                            if (initialViewer?.type === 'CodeEditor') {
                                decorateLineTwo(initialViewer)
                            }

                            // subscribe to viewer changes
                            let previousViewerSubscription: sourcegraph.Unsubscribable | undefined
                            context.subscriptions.add(
                                sourcegraph.app.activeWindowChanges.subscribe(activeWindow => {
                                    const viewerSubscription = activeWindow?.activeViewComponentChanges.subscribe(
                                        viewer => {
                                            if (viewer?.type === 'CodeEditor') {
                                                // Always decorate line 2
                                                decorateLineTwo(viewer)
                                            }
                                        }
                                    )
                                    if (previousViewerSubscription) {
                                        previousViewerSubscription.unsubscribe()
                                    }
                                    previousViewerSubscription = viewerSubscription
                                })
                            )
                        }

                        exports.activate = activate
                    },
                },
                {
                    id: 'TestSelectedLineID',
                    extensionID: 'test/selected-line',
                    extensionManifest: {
                        url: new URL(
                            '/-/static/extension/0001-test-selected-line.js?hash--test-selected-line',
                            driver.sourcegraphBaseUrl
                        ).href,
                        activationEvents: ['*'],
                    },
                    url: '/extensions/test/selected-line',
                    viewerCanAdminister: false,
                    bundle: function extensionBundle(): void {
                        // eslint-disable-next-line @typescript-eslint/no-require-imports, @typescript-eslint/no-var-requires
                        const sourcegraph = require('sourcegraph') as typeof import('sourcegraph')

                        const selectedLineDecorationType = sourcegraph.app.createDecorationType()

                        function activate(context: sourcegraph.ExtensionContext): void {
                            let previousViewerSubscription: sourcegraph.Unsubscribable | undefined
                            let previousSelectionsSubscription: sourcegraph.Unsubscribable | undefined

                            context.subscriptions.add(
                                sourcegraph.app.activeWindowChanges.subscribe(activeWindow => {
                                    const viewerSubscription = activeWindow?.activeViewComponentChanges.subscribe(
                                        viewer => {
                                            let selectionsSubscription: sourcegraph.Unsubscribable | undefined
                                            if (viewer?.type === 'CodeEditor') {
                                                selectionsSubscription = viewer.selectionsChanges.subscribe(
                                                    selections => {
                                                        viewer.setDecorations(
                                                            selectedLineDecorationType,
                                                            selections.map(selection => ({
                                                                range: new sourcegraph.Range(
                                                                    selection.start,
                                                                    selection.end
                                                                ),
                                                                after: {
                                                                    contentText: `selected line content for line ${selection.start.line}`,
                                                                    backgroundColor: 'green',
                                                                },
                                                            }))
                                                        )
                                                    }
                                                )
                                            }

                                            if (previousSelectionsSubscription) {
                                                previousSelectionsSubscription.unsubscribe()
                                            }
                                            previousSelectionsSubscription = selectionsSubscription
                                        }
                                    )

                                    if (previousViewerSubscription) {
                                        previousViewerSubscription.unsubscribe()
                                    }
                                    previousViewerSubscription = viewerSubscription
                                })
                            )
                        }

                        exports.activate = activate
                    },
                },
            ]

            const userSettings: Settings = {
                extensions: mockExtensions.reduce((extensionsSettings: Record<string, boolean>, mockExtension) => {
                    extensionsSettings[mockExtension.extensionID] = true
                    return extensionsSettings
                }, {}),
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
                                content: '// Log to console\nconsole.log("Hello world")\n// Third line',
                                richHTML: '',
                                highlight: {
                                    aborted: false,
                                    html:
                                        // Note: whitespace in this string is significant.
                                        '<table><tbody><tr><td class="line" data-line="1"></td><td class="code"><div><span style="color: gray">&sol;&sol; Log to console\n' +
                                        '</span></div></td></tr><tr><td class="line" data-line="2"></td><td class="code"><div><span style="color: #859900;" class="test-console-token">console</span><span style="color: #657b83;">.</span><span style="color: #859900;" class="test-log-token">log</span><span style="color: #657b83;">(</span><span style="color: #839496;">&quot;</span><span style="color: #2aa198;">Hello world</span><span style="color: #839496;">&quot;</span><span style="color: #657b83;">)\n' +
                                        '</span></div></td></tr><tr><td class="line" data-line="3"></td><td class="code"><div><span style="color: gray">&sol;&sol; Third line</span></td></tr></tbody></table>',
                                },
                            },
                        },
                    },
                }),
                Extensions: () => ({
                    extensionRegistry: {
                        extensions: {
                            nodes: mockExtensions.map(mockExtension => ({
                                ...mockExtension,
                                manifest: { raw: JSON.stringify(mockExtension.extensionManifest) },
                            })),
                        },
                    },
                }),
            })

            // Serve mock extension bundles
            for (const mockExtension of mockExtensions) {
                testContext.server
                    .get(new URL(mockExtension.extensionManifest.url, driver.sourcegraphBaseUrl).href)
                    .intercept((request, response) => {
                        // Create an immediately-invoked function expression for the extensionBundle function
                        const extensionBundleString = `(${mockExtension.bundle.toString()})()`
                        response.type('application/javascript; charset=utf-8').send(extensionBundleString)
                    })
            }
            const timeout = 5000
            await driver.page.goto(`${driver.sourcegraphBaseUrl}/github.com/sourcegraph/test/-/blob/test.ts`)

            // Wait for some line decoration attachment portal
            await driver.page.waitForSelector('.line-decoration-attachment-portal', { timeout })
            assert(
                !(await driver.page.$('#line-decoration-attachment-1')),
                'Expected line 1 to not have a decoration attachment portal'
            )
            assert(
                await driver.page.$('#line-decoration-attachment-2'),
                'Expected line 2 to have a decoration attachment portal before selecting a line'
            )
            // Count child nodes of existing portals
            assert.strictEqual(
                await driver.page.evaluate(
                    () => document.querySelector('#line-decoration-attachment-2')?.childElementCount
                ),
                1,
                'Expected line 2 to have 1 decoration'
            )

            // Select line 1. Line 1
            await driver.page.click('[data-line="1"]')
            await driver.page.waitForSelector('#line-decoration-attachment-1', { timeout })
            assert(
                await driver.page.$('#line-decoration-attachment-1'),
                'Expected line 1 to have a decoration attachment portal'
            )
            assert(
                await driver.page.$('#line-decoration-attachment-2'),
                'Expected line 2 to have a decoration attachment portal'
            )
            assert.strictEqual(
                await driver.page.evaluate(
                    () => document.querySelector('#line-decoration-attachment-2')?.childElementCount
                ),
                1,
                'Expected line 2 to have 1 decoration'
            )

            // Select line 2. Assert that everything is normal
            await driver.page.click('[data-line="2"]')
            await driver.page.waitFor(() => !document.querySelector('#line-decoration-attachment-1'), { timeout })
            assert(
                !(await driver.page.$('#line-decoration-attachment-1')),
                'Expected line 1 to not have a decoration attachment portal after selecting line 2'
            )
            assert(
                await driver.page.$('#line-decoration-attachment-2'),
                'Expected line 2 to have a decoration attachment portal'
            )

            // Count child nodes of existing portals
            assert.strictEqual(
                await driver.page.evaluate(
                    () => document.querySelector('#line-decoration-attachment-2')?.childElementCount
                ),
                2,
                'Expected line 2 to have 2 decorations'
            )

            // Select line 1 again. before fix, line 2 will still have 2 decorations
            await driver.page.click('[data-line="1"]')
            await driver.page.waitForSelector('#line-decoration-attachment-1', { timeout })
            assert(
                await driver.page.$('#line-decoration-attachment-1'),
                'Expected line 1 to have a decoration attachment portal after it is reselected'
            )
            assert(
                await driver.page.$('#line-decoration-attachment-2'),
                'Expected line 2 to have a decoration attachment portal after selecting line 1 again'
            )

            // Count child nodes of existing portals
            assert.strictEqual(
                await driver.page.evaluate(
                    () => document.querySelector('#line-decoration-attachment-2')?.childElementCount
                ),
                1,
                'Expected line 2 to have 1 decoration'
            )
        })

        it('sends the latest document to extensions', async () => {
            // This test is meant to prevent regression of
            // "extensions receive wrong text documents": https://github.com/sourcegraph/sourcegraph/issues/14965

            /**
             * How can we verify that extensions receive the latest document?
             * The test extension has to cause some change detectable to the web application, and
             * this change must be dependent on the text document. This extension should be simple to
             * avoid bugs in the extension itself.
             *
             * Simplest possible extension that satisfies these requirements:
             * add attachment to lines that contain a certain word.
             */

            const wordFinder: MockExtension = {
                id: 'TestWordFinderID',
                extensionID: 'test/word-finder',
                extensionManifest: {
                    url: new URL(
                        '/-/static/extension/0001-test-word-finder.js?hash--test-word-finder',
                        driver.sourcegraphBaseUrl
                    ).href,
                    activationEvents: ['*'],
                },
                url: '/extensions/test/word-finder',
                viewerCanAdminister: false,
                bundle: function extensionBundle(): void {
                    // eslint-disable-next-line @typescript-eslint/no-require-imports, @typescript-eslint/no-var-requires
                    const sourcegraph = require('sourcegraph') as typeof import('sourcegraph')

                    const fixedLineDecorationType = sourcegraph.app.createDecorationType()

                    // Match all occurences of 'word', decorate lines
                    function decorateWordLines(editor: sourcegraph.CodeEditor) {
                        const text = editor.document.text ?? ''

                        const wordPattern = /word/g
                        const matchIndices: number[] = []

                        for (const match of text.matchAll(wordPattern)) {
                            if (match.index) {
                                matchIndices.push(match.index)
                            }
                        }

                        editor.setDecorations(
                            fixedLineDecorationType,
                            matchIndices.map(index => {
                                const line = editor.document.positionAt(index).line

                                return {
                                    range: new sourcegraph.Range(
                                        new sourcegraph.Position(line, 0),
                                        new sourcegraph.Position(line, 0)
                                    ),
                                    after: {
                                        contentText: 'found word',
                                        backgroundColor: 'red',
                                    },
                                }
                            })
                        )
                    }

                    function activate(context: sourcegraph.ExtensionContext): void {
                        // check initial viewer in case it was already emitted before extension was activated
                        const initialViewer = sourcegraph.app.activeWindow?.activeViewComponent
                        if (initialViewer?.type === 'CodeEditor') {
                            decorateWordLines(initialViewer)
                        }

                        // subscribe to viewer changes
                        let previousViewerSubscription: sourcegraph.Unsubscribable | undefined
                        context.subscriptions.add(
                            sourcegraph.app.activeWindowChanges.subscribe(activeWindow => {
                                const viewerSubscription = activeWindow?.activeViewComponentChanges.subscribe(
                                    viewer => {
                                        if (viewer?.type === 'CodeEditor') {
                                            decorateWordLines(viewer)
                                        }
                                    }
                                )
                                if (previousViewerSubscription) {
                                    previousViewerSubscription.unsubscribe()
                                }
                                previousViewerSubscription = viewerSubscription
                            })
                        )
                    }

                    exports.activate = activate
                },
            }

            const userSettings: Settings = {
                extensions: {
                    'test/word-finder': true,
                },
            }

            testContext.overrideGraphQL({
                ...commonBlobGraphQlResults,
                TreeEntries: () =>
                    createTreeEntriesResult(repositorySourcegraphUrl, ['README.md', 'test.ts', 'fake.ts']),
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
                Blob: ({ filePath }) => {
                    const file =
                        filePath === 'fake.ts'
                            ? {
                                  content: '// First word line\n// Second line\n// Third word line',
                                  richHTML: '',
                                  highlight: {
                                      aborted: false,
                                      html:
                                          // Note: whitespace in this string is significant.
                                          '<table><tbody><tr><td class="line" data-line="1"></td><td class="code"><div><span style="color: gray">&sol;&sol; First word line\n' +
                                          '</span></div></td></tr><tr><td class="line" data-line="2"></td><td class="code"><div><span style="color: gray">&sol;&sol; Second line</span></td></tr>\n' +
                                          '<tr><td class="line" data-line="3"></td><td class="code"><div><span style="color: gray">&sol;&sol; Third word line</span></td></tr></tbody></table>',
                                  },
                              }
                            : {
                                  content: '// First line\n// Second word line\n// Third line',
                                  richHTML: '',
                                  highlight: {
                                      aborted: false,
                                      html:
                                          // Note: whitespace in this string is significant.
                                          '<table><tbody><tr><td class="line" data-line="1"></td><td class="code"><div><span style="color: gray">&sol;&sol; First line\n' +
                                          '</span></div></td></tr><tr><td class="line" data-line="2"></td><td class="code"><div><span style="color: gray">&sol;&sol; Second word line</span></td></tr>\n' +
                                          '<tr><td class="line" data-line="3"></td><td class="code"><div><span style="color: gray">&sol;&sol; Third line</span></td></tr></tbody></table>',
                                  },
                              }

                    return {
                        repository: {
                            commit: {
                                file,
                            },
                        },
                    }
                },
                Extensions: () => ({
                    extensionRegistry: {
                        extensions: {
                            nodes: [
                                {
                                    ...wordFinder,
                                    manifest: {
                                        raw: JSON.stringify(wordFinder.extensionManifest),
                                    },
                                },
                            ],
                        },
                    },
                }),
            })

            // Serve the word-finder extension bundle
            testContext.server
                .get(new URL(wordFinder.extensionManifest.url, driver.sourcegraphBaseUrl).href)
                .intercept((request, response) => {
                    // Create an immediately-invoked function expression for the extensionBundle function
                    const extensionBundleString = `(${wordFinder.bundle.toString()})()`
                    response.type('application/javascript; charset=utf-8').send(extensionBundleString)
                })

            const timeout = 3000
            await driver.page.goto(`${driver.sourcegraphBaseUrl}/github.com/sourcegraph/test/-/blob/test.ts`)

            // File 1 (test.ts). Only line two contains 'word'
            try {
                await driver.page.waitForSelector('#line-decoration-attachment-2', { timeout })
            } catch {
                // Rethrow with contextual error message
                throw new Error('Timeout waiting for #line-decoration-attachment-2 (test.ts, first time)')
            }
            // await driver.page.waitFor(1000)
            assert(
                !(await driver.page.$('#line-decoration-attachment-1')),
                'Expected line 1 to not have a decoration attachment on test.ts'
            )
            assert(
                await driver.page.$('#line-decoration-attachment-2'),
                'Expected line 2 to have a decoration attachment on test.ts'
            )
            assert(
                !(await driver.page.$('#line-decoration-attachment-3')),
                'Expected line 3 to not have a decoration attachment on test.ts'
            )

            // Go to File 2 (fake.ts). Lines one and three contain 'word'
            await driver.findElementWithText('fake.ts', {
                selector: '.test-tree-file-link',
                action: 'click',
            })
            try {
                await driver.page.waitForSelector('#line-decoration-attachment-1', { timeout })
            } catch {
                throw new Error('Timeout waiting for #line-decoration-attachment-1 (fake.ts)')
            }
            assert(
                await driver.page.$('#line-decoration-attachment-1'),
                'Expected line 1 to have a decoration attachment on fake.ts'
            )
            assert(
                !(await driver.page.$('#line-decoration-attachment-2')),
                'Expected line 2 to not have a decoration attachment on fake.ts'
            )
            assert(
                await driver.page.$('#line-decoration-attachment-3'),
                'Expected line 3 to have a decoration attachment on fake.ts'
            )

            // Come back to File 1 (test.ts)
            await driver.findElementWithText('test.ts', {
                selector: '.test-tree-file-link',
                action: 'click',
            })
            try {
                await driver.page.waitForSelector('#line-decoration-attachment-2', { timeout })
            } catch {
                throw new Error('Timeout waiting for #line-decoration-attachment-2 (test.ts, second time)')
            }
            assert(
                !(await driver.page.$('#line-decoration-attachment-1')),
                'Expected line 1 to not have a decoration attachment on test.ts (second visit)'
            )
            assert(
                await driver.page.$('#line-decoration-attachment-2'),
                'Expected line 2 to have a decoration attachment on test.ts (second visit)'
            )
            assert(
                !(await driver.page.$('#line-decoration-attachment-3')),
                'Expected line 3 to not have a decoration attachment on test.ts (second visit)'
            )
        })

        it.skip('gets displayed when navigating to a URL with a token position', async () => {
            await driver.page.goto(`${driver.sourcegraphBaseUrl}/github.com/sourcegraph/test/-/test.ts#2:9`)
            // TODO
        })

        it('properly displays reference panel for URIs with spaces', async () => {
            const repositoryName = 'github.com/sourcegraph/test%20repo'
            const files = ['test.ts', 'test spaces.ts']
            const commitID = '1234'
            const userSettings: Settings = {
                extensions: {
                    'test/references': true,
                },
            }
            const extensionManifest: ExtensionManifest = {
                url: new URL(
                    '/-/static/extension/0001-test-references.js?hash--test-references',
                    driver.sourcegraphBaseUrl
                ).href,
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
                Extensions: () => ({
                    extensionRegistry: {
                        extensions: {
                            nodes: [
                                {
                                    id: 'TestExtensionID',
                                    extensionID: 'test/references',
                                    manifest: {
                                        raw: JSON.stringify(extensionManifest),
                                    },
                                    url: '/extensions/test/references',
                                    viewerCanAdminister: false,
                                },
                            ],
                        },
                    },
                }),
                Blob: ({ filePath }) => ({
                    repository: {
                        commit: {
                            file: {
                                content: `// file path: ${filePath}\nconsole.log("Hello world")`,
                                richHTML: '',
                                highlight: {
                                    aborted: false,
                                    html:
                                        `<table><tbody><tr><td class="line" data-line="1"></td><td class="code"><div><span style="color: gray">&sol;&sol; file path: ${filePath}</span></div></td></tr>\n` +
                                        '<tr><td class="line" data-line="2"></td><td class="code"><div><span style="color: #859900;" class="test-console-token">console</span><span style="color: #657b83;">.</span><span style="color: #859900;" class="test-log-token">log</span><span style="color: #657b83;">(</span><span style="color: #839496;">&quot;</span><span style="color: #2aa198;">Hello world</span><span style="color: #839496;">&quot;</span><span style="color: #657b83;">)</span></div></td></tr>\n' +
                                        '</tbody></table>',
                                    lineRanges: [],
                                },
                            },
                        },
                    },
                }),
                TreeEntries: () => createTreeEntriesResult(repositorySourcegraphUrl, files),
                ResolveRev: () => createResolveRevisionResult(repositorySourcegraphUrl, commitID),
                HighlightedFile: ({ filePath }) => ({
                    repository: {
                        commit: {
                            file: {
                                isDirectory: false,
                                richHTML: '',
                                highlight: {
                                    aborted: false,
                                    html:
                                        `<table><tbody><tr><td class="line" data-line="1"></td><td class="code"><div><span style="color: gray">&sol;&sol; file path: ${filePath}</span></div></td></tr>\n` +
                                        '<tr><td class="line" data-line="2"></td><td class="code"><div><span style="color: #859900;" class="test-console-token">console</span><span style="color: #657b83;">.</span><span style="color: #859900;" class="test-log-token">log</span><span style="color: #657b83;">(</span><span style="color: #839496;">&quot;</span><span style="color: #2aa198;">Hello world</span><span style="color: #839496;">&quot;</span><span style="color: #657b83;">)</span></div></td></tr>\n' +
                                        '</tbody></table>',
                                    lineRanges: [],
                                },
                            },
                        },
                    },
                }),
                FetchCommits: () => ({
                    node: { __typename: 'GitCommit' },
                }),
                // Required for definition provider,
                ResolveRawRepoName: () => ({
                    repository: {
                        mirrorInfo: {
                            cloned: true,
                        },
                        uri: repositoryName,
                    },
                }),
            })

            // Serve a mock extension bundle with a simple reference provider
            testContext.server
                .get(new URL(extensionManifest.url, driver.sourcegraphBaseUrl).href)
                .intercept((request, response) => {
                    function extensionBundle(): void {
                        // eslint-disable-next-line @typescript-eslint/no-require-imports, @typescript-eslint/no-var-requires
                        const sourcegraph = require('sourcegraph') as typeof import('sourcegraph')

                        function activate(context: sourcegraph.ExtensionContext): void {
                            context.subscriptions.add(
                                sourcegraph.languages.registerReferenceProvider(['*'], {
                                    provideReferences: () => [
                                        new sourcegraph.Location(
                                            new URL('git://github.com/sourcegraph/test%20repo?1234#test%20spaces.ts'),
                                            new sourcegraph.Range(
                                                new sourcegraph.Position(0, 0),
                                                new sourcegraph.Position(1, 0)
                                            )
                                        ),
                                    ],
                                })
                            )

                            // We aren't testing definition providers in this test; we include a definition provider
                            // because the "Find references" action isn't displayed unless a definition is found
                            context.subscriptions.add(
                                sourcegraph.languages.registerDefinitionProvider(['*'], {
                                    provideDefinition: () =>
                                        new sourcegraph.Location(
                                            new URL('git://github.com/sourcegraph/test%20repo?1234#test%20spaces.ts'),
                                            new sourcegraph.Range(
                                                new sourcegraph.Position(0, 0),
                                                new sourcegraph.Position(1, 0)
                                            )
                                        ),
                                })
                            )
                        }

                        exports.activate = activate
                    }
                    // Create an immediately-invoked function expression for the extensionBundle function
                    const extensionBundleString = `(${extensionBundle.toString()})()`
                    response.type('application/javascript; charset=utf-8').send(extensionBundleString)
                })

            await driver.page.goto(`${driver.sourcegraphBaseUrl}/${repositoryName}/-/blob/test.ts`)

            // Click on "log" in "console.log()" in line 2
            await driver.page.waitForSelector('.test-log-token', { visible: true })
            await driver.page.click('.test-log-token')

            // Click 'Find references'
            await driver.page.waitForSelector('.test-tooltip-find-references', { visible: true })
            await driver.page.click('.test-tooltip-find-references')

            // Click on the first reference
            await driver.page.waitForSelector('.test-file-match-children-item')
            await driver.page.click('.test-file-match-children-item')

            // Assert that the first line of code has text content which contains: 'file path: test spaces.ts'
            try {
                await driver.page.waitForFunction(
                    () =>
                        document
                            .querySelector('.test-repo-blob [data-line="1"]')
                            ?.nextElementSibling?.textContent?.includes('file path: test spaces.ts'),
                    { timeout: 5000 }
                )
            } catch {
                throw new Error('Expected to navigate to file after clicking on link in references panel')
            }
        })

        describe('browser extension discoverability', () => {
            const HOVER_THRESHOLD = 5
            const HOVER_COUNT_KEY = 'hover-count'

            it(`shows a popover about the browser extension when the user has seen ${HOVER_THRESHOLD} hovers and clicks "View on [code host]" button`, async () => {
                testContext.server.get('https://github.com/*').intercept((request, response) => {
                    response.sendStatus(200)
                })

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
