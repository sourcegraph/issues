import assert from 'assert'
import { createDriverForTest, Driver, percySnapshot } from '../../../shared/src/testing/driver'
import { commonWebGraphQlResults } from './graphQlResults'
import { createWebIntegrationTestContext, WebIntegrationTestContext } from './context'
import {
    createRepositoryRedirectResult,
    createResolveRevisionResult,
    createFileExternalLinksResult,
    createTreeEntriesResult,
    createBlobContentResult,
} from './graphQlResponseHelpers'
import { afterEachSaveScreenshotIfFailed } from '../../../shared/src/testing/screenshotReporter'
import * as path from 'path'
import { DiffHunkLineType } from '../graphql-operations'

describe('Repository', () => {
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

    async function assertSelectorHasText(selector: string, text: string) {
        assert.strictEqual(
            await driver.page.evaluate(
                selector => document.querySelector<HTMLButtonElement>(selector)?.textContent,
                selector
            ),
            text
        )
    }

    describe('index page', () => {
        it('loads when accessed with a repo url', async () => {
            const shortRepositoryName = 'sourcegraph/jsonrpc2'
            const repositoryName = `github.com/${shortRepositoryName}`
            const repositorySourcegraphUrl = `/${repositoryName}`
            const clickedFileName = 'async.go'
            const clickedCommit = ''
            const fileEntries = ['jsonrpc2.go', clickedFileName]

            testContext.overrideGraphQL({
                ...commonWebGraphQlResults,
                RepositoryRedirect: ({ repoName }) => createRepositoryRedirectResult(repoName),
                ResolveRev: () => createResolveRevisionResult(repositorySourcegraphUrl),
                FileExternalLinks: ({ filePath }) => createFileExternalLinksResult(filePath),
                TreeEntries: () => createTreeEntriesResult(repositorySourcegraphUrl, fileEntries),
                Blob: () => createBlobContentResult('mock file blob'),
                TreeCommits: () => ({
                    node: {
                        __typename: 'Repository',
                        commit: {
                            ancestors: {
                                nodes: [
                                    {
                                        id: 'CommitID1',
                                        oid: '15c2290dcb37731cc4ee5a2a1c1e5a25b4c28f81',
                                        abbreviatedOID: '15c2290',
                                        message: 'update LSIF indexing CI workflow\n',
                                        subject: 'update LSIF indexing CI workflow',
                                        body: null,
                                        author: {
                                            person: {
                                                avatarURL: '',
                                                name: 'garo (they/them)',
                                                email: 'gbrik@users.noreply.github.com',
                                                displayName: 'garo (they/them)',
                                                user: null,
                                            },
                                            date: '2020-04-29T18:40:54Z',
                                        },
                                        committer: {
                                            person: {
                                                avatarURL: '',
                                                name: 'GitHub',
                                                email: 'noreply@github.com',
                                                displayName: 'GitHub',
                                                user: null,
                                            },
                                            date: '2020-04-29T18:40:54Z',
                                        },
                                        parents: [
                                            {
                                                oid: '96c4efab7ee28f3d1cf1d248a0139cea37368b18',
                                                abbreviatedOID: '96c4efa',
                                                url:
                                                    '/github.com/sourcegraph/jsonrpc2/-/commit/96c4efab7ee28f3d1cf1d248a0139cea37368b18',
                                            },
                                            {
                                                oid: '9e615b1c32cc519130575e8d10d0d0fee8a5eb6c',
                                                abbreviatedOID: '9e615b1',
                                                url:
                                                    '/github.com/sourcegraph/jsonrpc2/-/commit/9e615b1c32cc519130575e8d10d0d0fee8a5eb6c',
                                            },
                                        ],
                                        url:
                                            '/github.com/sourcegraph/jsonrpc2/-/commit/15c2290dcb37731cc4ee5a2a1c1e5a25b4c28f81',
                                        canonicalURL:
                                            '/github.com/sourcegraph/jsonrpc2/-/commit/15c2290dcb37731cc4ee5a2a1c1e5a25b4c28f81',
                                        externalURLs: [
                                            {
                                                url:
                                                    'https://github.com/sourcegraph/jsonrpc2/commit/15c2290dcb37731cc4ee5a2a1c1e5a25b4c28f81',
                                                serviceType: 'github',
                                            },
                                        ],
                                        tree: {
                                            canonicalURL:
                                                '/github.com/sourcegraph/jsonrpc2@15c2290dcb37731cc4ee5a2a1c1e5a25b4c28f81',
                                        },
                                    },
                                    {
                                        id: 'CommitID2',
                                        oid: '9e615b1c32cc519130575e8d10d0d0fee8a5eb6c',
                                        abbreviatedOID: '9e615b1',
                                        message: 'LSIF Indexing Campaign',
                                        subject: 'LSIF Indexing Campaign',
                                        body: null,
                                        author: {
                                            person: {
                                                avatarURL: '',
                                                name: 'Sourcegraph Bot',
                                                email: 'campaigns@sourcegraph.com',
                                                displayName: 'Sourcegraph Bot',
                                                user: null,
                                            },
                                            date: '2020-04-29T16:57:20Z',
                                        },
                                        committer: {
                                            person: {
                                                avatarURL: '',
                                                name: 'Sourcegraph Bot',
                                                email: 'campaigns@sourcegraph.com',
                                                displayName: 'Sourcegraph Bot',
                                                user: null,
                                            },
                                            date: '2020-04-29T16:57:20Z',
                                        },
                                        parents: [
                                            {
                                                oid: '96c4efab7ee28f3d1cf1d248a0139cea37368b18',
                                                abbreviatedOID: '96c4efa',
                                                url:
                                                    '/github.com/sourcegraph/jsonrpc2/-/commit/96c4efab7ee28f3d1cf1d248a0139cea37368b18',
                                            },
                                        ],
                                        url:
                                            '/github.com/sourcegraph/jsonrpc2/-/commit/9e615b1c32cc519130575e8d10d0d0fee8a5eb6c',
                                        canonicalURL:
                                            '/github.com/sourcegraph/jsonrpc2/-/commit/9e615b1c32cc519130575e8d10d0d0fee8a5eb6c',
                                        externalURLs: [
                                            {
                                                url:
                                                    'https://github.com/sourcegraph/jsonrpc2/commit/9e615b1c32cc519130575e8d10d0d0fee8a5eb6c',
                                                serviceType: 'github',
                                            },
                                        ],
                                        tree: {
                                            canonicalURL:
                                                '/github.com/sourcegraph/jsonrpc2@9e615b1c32cc519130575e8d10d0d0fee8a5eb6c',
                                        },
                                    },
                                    {
                                        id: 'CommitID3',
                                        oid: '96c4efab7ee28f3d1cf1d248a0139cea37368b18',
                                        abbreviatedOID: '96c4efa',
                                        message:
                                            'Produce LSIF data for each commit for fast/precise code nav (#35)\n\n* Produce LSIF data for each commit for fast/precise code nav\r\n\r\n* Update lsif.yml\r',
                                        subject: 'Produce LSIF data for each commit for fast/precise code nav (#35)',
                                        body:
                                            '* Produce LSIF data for each commit for fast/precise code nav\r\n\r\n* Update lsif.yml',
                                        author: {
                                            person: {
                                                avatarURL: '',
                                                name: 'Quinn Slack',
                                                email: 'qslack@qslack.com',
                                                displayName: 'Quinn Slack',
                                                user: {
                                                    id: 'VXNlcjo2',
                                                    username: 'sqs',
                                                    url: '/users/sqs',
                                                    displayName: 'sqs',
                                                },
                                            },
                                            date: '2019-12-22T04:34:38Z',
                                        },
                                        committer: {
                                            person: {
                                                avatarURL: '',
                                                name: 'GitHub',
                                                email: 'noreply@github.com',
                                                displayName: 'GitHub',
                                                user: null,
                                            },
                                            date: '2019-12-22T04:34:38Z',
                                        },
                                        parents: [
                                            {
                                                oid: 'cee7209801bf50cee868f8e0696ba0b76ae21792',
                                                abbreviatedOID: 'cee7209',
                                                url:
                                                    '/github.com/sourcegraph/jsonrpc2/-/commit/cee7209801bf50cee868f8e0696ba0b76ae21792',
                                            },
                                        ],
                                        url:
                                            '/github.com/sourcegraph/jsonrpc2/-/commit/96c4efab7ee28f3d1cf1d248a0139cea37368b18',
                                        canonicalURL:
                                            '/github.com/sourcegraph/jsonrpc2/-/commit/96c4efab7ee28f3d1cf1d248a0139cea37368b18',
                                        externalURLs: [
                                            {
                                                url:
                                                    'https://github.com/sourcegraph/jsonrpc2/commit/96c4efab7ee28f3d1cf1d248a0139cea37368b18',
                                                serviceType: 'github',
                                            },
                                        ],
                                        tree: {
                                            canonicalURL:
                                                '/github.com/sourcegraph/jsonrpc2@96c4efab7ee28f3d1cf1d248a0139cea37368b18',
                                        },
                                    },
                                ],
                                pageInfo: { hasNextPage: false },
                            },
                        },
                    },
                }),
                RepositoryCommit: () => ({
                    node: {
                        __typename: 'Repository',
                        commit: {
                            __typename: 'GitCommit',
                            id: 'CommitID1',
                            oid: '15c2290dcb37731cc4ee5a2a1c1e5a25b4c28f81',
                            abbreviatedOID: '15c2290',
                            message: 'update LSIF indexing CI workflow\n',
                            subject: 'update LSIF indexing CI workflow',
                            body: null,
                            author: {
                                person: {
                                    avatarURL: '',
                                    name: 'garo (they/them)',
                                    email: 'gbrik@users.noreply.github.com',
                                    displayName: 'garo (they/them)',
                                    user: null,
                                },
                                date: '2020-04-29T18:40:54Z',
                            },
                            committer: {
                                person: {
                                    avatarURL: '',
                                    name: 'GitHub',
                                    email: 'noreply@github.com',
                                    displayName: 'GitHub',
                                    user: null,
                                },
                                date: '2020-04-29T18:40:54Z',
                            },
                            parents: [
                                {
                                    oid: '96c4efab7ee28f3d1cf1d248a0139cea37368b18',
                                    abbreviatedOID: '96c4efa',
                                    url:
                                        '/github.com/sourcegraph/jsonrpc2/-/commit/96c4efab7ee28f3d1cf1d248a0139cea37368b18',
                                },
                                {
                                    oid: '9e615b1c32cc519130575e8d10d0d0fee8a5eb6c',
                                    abbreviatedOID: '9e615b1',
                                    url:
                                        '/github.com/sourcegraph/jsonrpc2/-/commit/9e615b1c32cc519130575e8d10d0d0fee8a5eb6c',
                                },
                            ],
                            url: '/github.com/sourcegraph/jsonrpc2/-/commit/15c2290dcb37731cc4ee5a2a1c1e5a25b4c28f81',
                            canonicalURL:
                                '/github.com/sourcegraph/jsonrpc2/-/commit/15c2290dcb37731cc4ee5a2a1c1e5a25b4c28f81',
                            externalURLs: [
                                {
                                    url:
                                        'https://github.com/sourcegraph/jsonrpc2/commit/15c2290dcb37731cc4ee5a2a1c1e5a25b4c28f81',
                                    serviceType: 'github',
                                },
                            ],
                            tree: {
                                canonicalURL:
                                    '/github.com/sourcegraph/jsonrpc2@15c2290dcb37731cc4ee5a2a1c1e5a25b4c28f81',
                            },
                        },
                    },
                }),
                RepositoryComparisonDiff: () => ({
                    node: {
                        comparison: {
                            fileDiffs: {
                                nodes: [
                                    {
                                        __typename: 'FileDiff',
                                        oldPath: '.github/workflows/lsif.yml',
                                        oldFile: { __typename: 'GitBlob', binary: false, byteSize: 381 },
                                        newFile: { __typename: 'GitBlob', binary: false, byteSize: 304 },
                                        newPath: '.github/workflows/lsif.yml',
                                        mostRelevantFile: {
                                            __typename: 'GitBlob',
                                            url:
                                                '/github.com/sourcegraph/jsonrpc2@15c2290dcb37731cc4ee5a2a1c1e5a25b4c28f81/-/blob/.github/workflows/lsif.yml',
                                        },
                                        hunks: [
                                            {
                                                oldRange: { startLine: 2, lines: 15 },
                                                oldNoNewlineAt: false,
                                                newRange: { startLine: 2, lines: 12 },
                                                section: 'name: LSIF',
                                                highlight: {
                                                    aborted: false,
                                                    lines: [
                                                        {
                                                            kind: DiffHunkLineType.DELETED,
                                                            html:
                                                                '<div><span style="color:#657b83;">  </span><span style="color:#268bd2;">build</span><span style="color:#657b83;">:\n</span></div>',
                                                        },
                                                        {
                                                            kind: DiffHunkLineType.ADDED,
                                                            html:
                                                                '<div><span style="color:#657b83;">  </span><span style="color:#268bd2;">lsif-go</span><span style="color:#657b83;">:\n</span></div>',
                                                        },
                                                    ],
                                                },
                                            },
                                        ],
                                        stat: { added: 1, changed: 3, deleted: 4 },
                                        internalID: '084bcb27838a8adbbbe10f664420f2d2',
                                    },
                                ],
                                totalCount: 1,
                                pageInfo: { endCursor: null, hasNextPage: false },
                                diffStat: { added: 1, changed: 3, deleted: 4 },
                            },
                        },
                    },
                }),
            })

            // Mock `Date.now` to stabilize timestamps
            await driver.page.evaluateOnNewDocument(() => {
                // Number of ms between Unix epoch and July 1, 2020 (arbitrary)
                const mockMs = new Date('July 1, 2020 00:00:00 UTC').getTime()
                Date.now = () => mockMs
            })

            await driver.page.goto(driver.sourcegraphBaseUrl + repositorySourcegraphUrl)

            await driver.page.waitForSelector('h2.tree-page__title')

            // Assert that the directory listing displays properly
            await driver.page.waitForSelector('.test-tree-entries')
            await percySnapshot(driver.page, 'Repository index page')

            const numberOfFileEntries = await driver.page.evaluate(
                () => document.querySelectorAll<HTMLButtonElement>('.test-tree-entry-file')?.length
            )

            assert.strictEqual(numberOfFileEntries, fileEntries.length, 'Number of files in directory listing')

            await testContext.waitForGraphQLRequest(async () => {
                await driver.findElementWithText(clickedFileName, {
                    selector: '.test-tree-entry-file',
                    action: 'click',
                })
            }, 'Blob')

            await driver.page.waitForSelector('.test-repo-blob')
            await driver.assertWindowLocation(`${repositorySourcegraphUrl}/-/blob/${clickedFileName}`)

            // Assert breadcrumb order
            await driver.page.waitForSelector('.test-breadcrumb')
            const breadcrumbTexts = await driver.page.evaluate(() =>
                [...document.querySelectorAll('.test-breadcrumb')].map(breadcrumb => breadcrumb.textContent)
            )
            assert.deepStrictEqual(breadcrumbTexts, [
                'Home',
                'Repositories',
                shortRepositoryName,
                '@master',
                clickedFileName,
            ])

            // Return to repo page
            await driver.page.waitForSelector('a.repo-header__repo')
            await driver.page.click('a.repo-header__repo')
            await driver.page.waitForSelector('h2.tree-page__title')
            await assertSelectorHasText('h2.tree-page__title', ' ' + shortRepositoryName)
            await driver.assertWindowLocation(repositorySourcegraphUrl)

            await driver.findElementWithText(clickedCommit, { selector: '.git-commit-node__oid', action: 'click' })
            await driver.page.waitForSelector('.git-commit-node__message-subject')
            await assertSelectorHasText('.git-commit-node__message-subject', 'update LSIF indexing CI workflow')
        })

        it('works with files with spaces in the name', async () => {
            const shortRepositoryName = 'ggilmore/q-test'
            const fileName = '% token.4288249258.sql'
            const directoryName = "Geoffrey's random queries.32r242442bf"
            const filePath = path.posix.join(directoryName, fileName)

            testContext.overrideGraphQL({
                ...commonWebGraphQlResults,
                RepositoryRedirect: ({ repoName }) => createRepositoryRedirectResult(repoName),
                ResolveRev: ({ repoName }) => createResolveRevisionResult(repoName),
                FileExternalLinks: ({ filePath, repoName, revision }) =>
                    createFileExternalLinksResult(
                        `https://${repoName}/blob/${revision}/${filePath.split('/').map(encodeURIComponent).join('/')}`
                    ),
                TreeEntries: () => ({
                    repository: {
                        commit: {
                            tree: {
                                isRoot: false,
                                url: '/github.com/ggilmore/q-test/-/tree/Geoffrey%27s%20random%20queries.32r242442bf',
                                entries: [
                                    {
                                        name: fileName,
                                        path: filePath,
                                        isDirectory: false,
                                        url:
                                            '/github.com/ggilmore/q-test/-/blob/Geoffrey%27s%20random%20queries.32r242442bf/%25%20token.4288249258.sql',
                                        submodule: null,
                                        isSingleChild: false,
                                    },
                                ],
                            },
                        },
                    },
                }),
                TreeCommits: () => ({
                    node: {
                        __typename: 'Repository',
                        commit: { ancestors: { nodes: [], pageInfo: { hasNextPage: false } } },
                    },
                }),
                Blob: ({ filePath }) => createBlobContentResult(`content for: ${filePath}`),
            })

            await driver.page.goto(
                `${driver.sourcegraphBaseUrl}/github.com/ggilmore/q-test/-/tree/Geoffrey's%20random%20queries.32r242442bf`
            )
            await driver.page.waitForSelector('.test-tree-file-link')
            assert.strictEqual(
                await driver.page.evaluate(() => document.querySelector('.test-tree-file-link')?.textContent),
                fileName
            )

            // page.click() fails for some reason with Error: Node is either not visible or not an HTMLElement
            await driver.page.$eval('.test-tree-file-link', linkElement => (linkElement as HTMLElement).click())
            await driver.page.waitForSelector('.test-repo-blob')

            await driver.page.waitForSelector('.test-breadcrumb')
            const breadcrumbTexts = await driver.page.evaluate(() =>
                [...document.querySelectorAll('.test-breadcrumb')].map(breadcrumb => breadcrumb.textContent)
            )
            assert.deepStrictEqual(breadcrumbTexts, ['Home', 'Repositories', shortRepositoryName, '@master', filePath])

            await driver.page.waitForSelector('#monaco-query-input .view-lines')
            // TODO: find a more reliable way to get the current search query,
            // to account for the fact that it may _actually_ contain non-breaking spaces
            // (and not just have spaces rendered as non-breaking in the DOM by Monaco)
            // https://github.com/sourcegraph/sourcegraph/issues/14756
            const searchQuery = (
                await driver.page.evaluate(() => document.querySelector('#monaco-query-input .view-lines')?.textContent)
            )?.replace(/\u00A0/g, ' ')
            assert.strictEqual(
                searchQuery,
                'repo:^github\\.com/ggilmore/q-test$ file:"^Geoffrey\'s random queries\\\\.32r242442bf/% token\\\\.4288249258\\\\.sql"'
            )

            await driver.page.waitForSelector('.test-go-to-code-host')
            assert.strictEqual(
                await driver.page.evaluate(
                    () => document.querySelector<HTMLAnchorElement>('.test-go-to-code-host')?.href
                ),
                "https://github.com/ggilmore/q-test/blob/master/Geoffrey's%20random%20queries.32r242442bf/%25%20token.4288249258.sql"
            )

            const blobContent = await driver.page.evaluate(() => document.querySelector('.test-repo-blob')?.textContent)
            assert.strictEqual(blobContent, `content for: ${filePath}`)
        })

        it('works with spaces in the repository name', async () => {
            const shortRepositoryName = 'my org/repo with spaces'
            const repositorySourcegraphUrl = '/github.com/my%20org/repo%20with%20spaces'

            testContext.overrideGraphQL({
                ...commonWebGraphQlResults,
                RepositoryRedirect: ({ repoName }) => createRepositoryRedirectResult(repoName),
                ResolveRev: ({ repoName }) => createResolveRevisionResult(repoName),
                FileExternalLinks: ({ filePath }) => createFileExternalLinksResult(filePath),
                TreeEntries: () => createTreeEntriesResult(repositorySourcegraphUrl, ['readme.md']),

                TreeCommits: () => ({
                    node: {
                        __typename: 'Repository',
                        commit: { ancestors: { nodes: [], pageInfo: { hasNextPage: false } } },
                    },
                }),
                Blob: ({ filePath }) => createBlobContentResult(`content for: ${filePath}`),
            })

            await driver.page.goto(driver.sourcegraphBaseUrl + repositorySourcegraphUrl)

            await driver.page.waitForSelector('h2.tree-page__title')
            await assertSelectorHasText('h2.tree-page__title', ' my org/repo with spaces')
            await assertSelectorHasText('.test-tree-entry-file', 'readme.md')

            // page.click() fails for some reason with Error: Node is either not visible or not an HTMLElement
            await driver.page.$eval('.test-tree-file-link', linkElement => (linkElement as HTMLElement).click())
            await driver.page.waitForSelector('.test-repo-blob')

            await driver.page.waitForSelector('.test-breadcrumb')
            const breadcrumbTexts = await driver.page.evaluate(() =>
                [...document.querySelectorAll('.test-breadcrumb')].map(breadcrumb => breadcrumb.textContent)
            )
            assert.deepStrictEqual(breadcrumbTexts, [
                'Home',
                'Repositories',
                shortRepositoryName,
                '@master',
                'readme.md',
            ])
        })
    })
})
