import { Driver, createDriverForTest } from '../../../../shared/src/testing/driver'
import { BrowserIntegrationTestContext, createBrowserIntegrationTestContext } from './context'
import { BrowserGraphQlOperations } from '../../graphql-operations'
import { SharedGraphQlOperations } from '../../../../shared/src/graphql-operations'
import { afterEachSaveScreenshotIfFailed } from '../../../../shared/src/testing/screenshotReporter'
import { closeInstallPageTab, testSingleFilePage } from '../shared'
import { commonBrowserGraphQlResults } from './graphQlResults'
import delay from 'delay'

describe('GitHub', () => {
    const commonBlobGraphQlResults: Partial<BrowserGraphQlOperations & SharedGraphQlOperations> = {
        ...commonBrowserGraphQlResults,
        // TODO(tj): common graphql overrides
        SiteProductVersion: () => ({
            site: {
                productVersion: '82560_2020-12-11_a5c30d3',
                buildVersion: '82560_2020-12-11_a5c30d3',
                hasCodeIntelligence: true,
            },
        }),
        ViewerConfiguration: () => ({
            viewerConfiguration: {
                subjects: [],
                merged: { contents: '', messages: [] },
            },
        }),
        ResolveRepo: ({ rawRepoName }) => ({
            repository: {
                name: rawRepoName,
            },
        }),
        ResolveRev: () => ({
            repository: {
                mirrorInfo: {
                    cloned: true,
                },
                commit: {
                    oid: '1'.repeat(40),
                },
            },
        }),
        // RepositoryRedirect: ({ repoName }) => createRepositoryRedirectResult(repoName),
        // ResolveRev: () => createResolveRevisionResult(repositorySourcegraphUrl),
        // FileExternalLinks: ({ filePath }) =>
        //     createFileExternalLinksResult(`https://${repositoryName}/blob/master/${filePath}`),
        // TreeEntries: () => createTreeEntriesResult(repositorySourcegraphUrl, ['README.md', fileName]),
        // Blob: ({ filePath }) => createBlobContentResult(`content for: ${filePath}\nsecond line\nthird line`),
        BlobContent: () => ({
            repository: {
                commit: {
                    file: {
                        content:
                            'package jsonrpc2\n\n// CallOption is an option that can be provided to (*Conn).Call to\n// configure custom behavior. See Meta.\ntype CallOption interface {\n\tapply(r *Request) error\n}\n\ntype callOptionFunc func(r *Request) error\n\nfunc (c callOptionFunc) apply(r *Request) error { return c(r) }\n\n// Meta returns a call option which attaches the given meta object to\n// the JSON-RPC 2.0 request (this is a Sourcegraph extension to JSON\n// RPC 2.0 for carrying metadata).\nfunc Meta(meta interface{}) CallOption {\n\treturn callOptionFunc(func(r *Request) error {\n\t\treturn r.SetMeta(meta)\n\t})\n}\n\n// PickID returns a call option which sets the ID on a request. Care must be\n// taken to ensure there are no conflicts with any previously picked ID, nor\n// with the default sequence ID.\nfunc PickID(id ID) CallOption {\n\treturn callOptionFunc(func(r *Request) error {\n\t\tr.ID = id\n\t\treturn nil\n\t})\n}\n',
                    },
                },
            },
        }),
    }

    let driver: Driver
    before(async () => {
        driver = await createDriverForTest({ loadExtension: true })
    })
    after(() => driver?.close())
    let testContext: BrowserIntegrationTestContext
    beforeEach(async function () {
        testContext = await createBrowserIntegrationTestContext({
            driver,
            currentTest: this.currentTest!,
            directory: __dirname,
        })
        testContext.server.get('https://collector.githubapp.com/*').intercept((request, response) => {
            response.sendStatus(200)
        })
        testContext.server.get('https://github.githubassets.com/favicons/*').intercept((request, response) => {
            response.sendStatus(200)
        })
        testContext.server.any('https://api.github.com/_private/browser/*').intercept((request, response) => {
            response.sendStatus(200)
        })
        // TODO(tj): all bext tests should intercept this
        testContext.server.get('https://storage.googleapis.com/sourcegraph-assets/*').intercept((request, response) => {
            response.sendStatus(200)
        })

        testContext.server.get('http://localhost:8890/*').intercept((request, response) => {
            response.sendStatus(200)
        })

        testContext.overrideGraphQL(commonBlobGraphQlResults)

        await delay(3000) // TODO(tj): Why?
        await closeInstallPageTab(driver.browser)
        await driver.setExtensionSourcegraphUrl()
    })
    afterEachSaveScreenshotIfFailed(() => driver.page)
    afterEach(() => testContext?.dispose())

    testSingleFilePage({
        getDriver: () => driver,
        url: 'https://github.com/sourcegraph/jsonrpc2/blob/4fb7cd90793ee6ab445f466b900e6bffb9b63d78/call_opt.go',
        repoName: 'github.com/sourcegraph/jsonrpc2',
        // Not using '.js-file-line' because it breaks the reliance on :nth-child() in testSingleFilePage()
        lineSelector: '.js-file-line-container tr',
        goToDefinitionURL:
            'https://github.com/sourcegraph/jsonrpc2/blob/4fb7cd90793ee6ab445f466b900e6bffb9b63d78/call_opt.go#L5:6',
    })

    it.skip('hover tooltips', () => {})
    // TODO(tj): mock extensions
})
