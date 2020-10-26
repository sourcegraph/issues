import { createDriverForTest, Driver } from '../../../../shared/src/testing/driver'
import { Config } from '../../../../shared/src/testing/config'
import { GraphQLClient, createGraphQLClient } from './GraphQlClient'
import { TestResourceManager } from './TestResourceManager'

/**
 * Returns tools for regression tests. The GraphQL client is authenticated with the sudo token, so
 * therefore has admin rights. The driver drives the puppeteer browser page instance. The resource
 * manager manages cleanup of resources created during the test.
 */
export async function getTestTools(
    config: Pick<
        Config,
        | 'sourcegraphBaseUrl'
        | 'logBrowserConsole'
        | 'slowMo'
        | 'headless'
        | 'sudoToken'
        | 'sudoUsername'
        | 'keepBrowser'
    >
): Promise<{
    gqlClient: GraphQLClient
    driver: Driver
    resourceManager: TestResourceManager
}> {
    const driver = await createAndInitializeDriver(config)
    const gqlClient = createGraphQLClient({
        baseUrl: config.sourcegraphBaseUrl,
        token: config.sudoToken,
        sudoUsername: config.sudoUsername,
    })
    const resourceManager = new TestResourceManager()

    return {
        driver,
        gqlClient,
        resourceManager,
    }
}

/**
 * Returns a Puppeteer driver with a 5s command timeout. It is important that none of the Jest test
 * timeouts is under 5s. Otherwise, the timeout error will be a cryptic Jest timeout error, instead
 * of an error pointing to the timed-out Puppeteer command.
 */
export async function createAndInitializeDriver(
    config: Pick<Config, 'sourcegraphBaseUrl' | 'logBrowserConsole' | 'slowMo' | 'headless' | 'keepBrowser'>
): Promise<Driver> {
    const driver = await createDriverForTest(config)
    driver.page.setDefaultNavigationTimeout(5 * 1000) // 5s navigation timeout
    return driver
}
