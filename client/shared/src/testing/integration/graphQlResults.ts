import { SharedGraphQlOperations } from '../../graphql-operations'

export const testUserID = 'TestUserID'
export const settingsID = 123

/**
 * Predefined results for GraphQL requests that are made on almost every page.
 */
export const sharedGraphQlResults: Partial<SharedGraphQlOperations> = {}

export const emptyResponse = {
    alwaysNil: null,
}
