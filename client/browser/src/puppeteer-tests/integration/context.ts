import {
    createSharedIntegrationTestContext,
    IntegrationTestContext,
    IntegrationTestOptions,
} from '../../../../shared/src/testing/integration/context'
import { BrowserGraphQlOperations } from '../../graphql-operations'
import { SharedGraphQlOperations } from '../../../../shared/src/graphql-operations'
import { commonBrowserGraphQlResults } from './graphQlResults'

export interface BrowserIntegrationTestContext
    extends IntegrationTestContext<
        BrowserGraphQlOperations & SharedGraphQlOperations,
        string & keyof (BrowserGraphQlOperations & SharedGraphQlOperations)
    > {}

/**
 * Creates the intergation test context for integration tests testing the web app.
 * This should be called in a `beforeEach()` hook and assigned to a variable `testContext` in the test scope.
 */
export const createBrowserIntegrationTestContext = async ({
    driver,
    currentTest,
    directory,
}: IntegrationTestOptions): Promise<BrowserIntegrationTestContext> => {
    const sharedTestContext = await createSharedIntegrationTestContext<
        BrowserGraphQlOperations & SharedGraphQlOperations,
        string & keyof (BrowserGraphQlOperations & SharedGraphQlOperations)
    >({ driver, currentTest, directory })
    sharedTestContext.overrideGraphQL(commonBrowserGraphQlResults)

    return {
        ...sharedTestContext,
    }
}
