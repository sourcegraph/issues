import { afterEachSaveScreenshotIfFailed } from '../../../shared/src/testing/screenshotReporter'
import { createDriverForTest, Driver } from '../../../shared/src/testing/driver'
import { testSingleFilePage } from './shared'
import { getConfig } from '../../../shared/src/testing/config'
import { ExternalServiceKind } from '../../../shared/src/graphql-operations'

// By default, these tests run against gitlab.com and a local Sourcegraph instance.
// You can run them against other instances by setting the below env vars in addition to SOURCEGRAPH_BASE_URL.

const GITLAB_BASE_URL = process.env.GITLAB_BASE_URL || 'https://gitlab.com'
const GITLAB_TOKEN = process.env.GITLAB_TOKEN
const REPO_PATH_PREFIX = new URL(GITLAB_BASE_URL).hostname

const { sourcegraphBaseUrl, ...restConfig } = getConfig('sourcegraphBaseUrl')

describe('Sourcegraph browser extension on Gitlab Server', () => {
    let driver: Driver

    before(async function () {
        this.timeout(4 * 60 * 1000)
        driver = await createDriverForTest({ loadExtension: true, sourcegraphBaseUrl })

        if (sourcegraphBaseUrl !== 'https://sourcegraph.com') {
            if (restConfig.testUserPassword) {
                await driver.ensureLoggedIn({ username: 'test', password: restConfig.testUserPassword })
            }
            await driver.setExtensionSourcegraphUrl()
            await driver.ensureHasExternalService({
                kind: ExternalServiceKind.GITLAB,
                displayName: 'Gitlab',
                config: JSON.stringify({
                    url: GITLAB_BASE_URL,
                    token: GITLAB_TOKEN,
                    projectQuery: ['groups/sourcegraph/projects?search=jsonrpc2'],
                }),
                ensureRepos: [REPO_PATH_PREFIX + '/sourcegraph/jsonrpc2'],
            })
            await driver.ensureHasCORSOrigin({ corsOriginURL: GITLAB_BASE_URL })
        }
    })

    after(async () => {
        await driver.close()
    })

    // Take a screenshot when a test fails.
    afterEachSaveScreenshotIfFailed(() => driver.page)

    const url = new URL(
        '/sourcegraph/jsonrpc2/blob/4fb7cd90793ee6ab445f466b900e6bffb9b63d78/call_opt.go',
        GITLAB_BASE_URL
    )
    testSingleFilePage({
        getDriver: () => driver,
        url: url.href,
        // Other than GitHub, the URL must not include the column in the hash.
        goToDefinitionURL: new URL('#L5', url.href).href,
        repoName: `${REPO_PATH_PREFIX}/sourcegraph/jsonrpc2`,
        sourcegraphBaseUrl,
        lineSelector: '.line',
    })
})
