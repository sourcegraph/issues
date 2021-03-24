import { BrowserGraphQlOperations } from '../../graphql-operations'
import { SharedGraphQlOperations } from '../../../../shared/src/graphql-operations'
import { sharedGraphQlResults } from '../../../../shared/src/testing/integration/graphQlResults'

/**
 * Predefined results for GraphQL requests that are made on almost every page.
 */
export const commonBrowserGraphQlResults: Partial<BrowserGraphQlOperations & SharedGraphQlOperations> = {
    ...sharedGraphQlResults,
    logEvent: () => ({
        logEvent: {
            alwaysNil: null,
        },
    }),
    logUserEvent: () => ({
        logUserEvent: {
            alwaysNil: null,
        },
    }),
}
