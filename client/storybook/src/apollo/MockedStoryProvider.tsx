import { ApolloLink } from '@apollo/client'
import { MockedProvider, MockedResponse, MockLink } from '@apollo/client/testing'
import { getOperationName } from '@apollo/client/utilities'
import React from 'react'

const forceMockVariablesLink = (mocks: readonly MockedResponse[]): ApolloLink =>
    new ApolloLink((operation, forward) => {
        console.log(operation)
        const mock = mocks.find(mock => getOperationName(mock.request.query) === operation.operationName)
        if (mock) {
            operation.variables = mock.request.variables || {}
        } else {
            console.warn(`Unable to find a mock for query: ${operation.operationName}. Did you mean to mock this?`)
        }
        return forward(operation)
    })

interface MockedForcedProvider {
    mocks?: readonly MockedResponse[]
}

/**
 * Similar to MockedProvider, except it forces request variables to match the specified mock.
 */
export const MockedStoryProvider: React.FunctionComponent<MockedForcedProvider> = ({ children, mocks = [] }) => (
    <MockedProvider mocks={mocks} link={ApolloLink.from([forceMockVariablesLink(mocks), new MockLink(mocks)])}>
        {children}
    </MockedProvider>
)
