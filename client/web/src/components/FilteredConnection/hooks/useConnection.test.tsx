import { MockedProvider, MockedResponse } from '@apollo/client/testing'
import { fireEvent } from '@testing-library/react'
import React from 'react'

import { dataOrThrowErrors, getDocumentNode, gql } from '@sourcegraph/shared/src/graphql/graphql'
import { waitForNextApolloResponse } from '@sourcegraph/shared/src/testing/apollo'
import { renderWithRouter, RenderWithRouterResult } from '@sourcegraph/shared/src/testing/render-with-router'

import {
    TestConnectionQueryFields,
    TestConnectionQueryResult,
    TestConnectionQueryVariables,
} from '../../../graphql-operations'

import { useConnection } from './useConnection'

const TEST_CONNECTION_QUERY = gql`
    query TestConnectionQuery($username: String!, $first: Int) {
        user(username: $username) {
            repositories(first: $first) {
                nodes {
                    id
                    name
                }
                totalCount
                pageInfo {
                    endCursor
                    hasNextPage
                }
            }
        }
    }

    fragment TestConnectionQueryFields on Repository {
        name
        id
    }
`

const TestComponent = () => {
    const { connection, fetchMore, hasNextPage } = useConnection<
        TestConnectionQueryResult,
        TestConnectionQueryVariables,
        TestConnectionQueryFields
    >({
        query: TEST_CONNECTION_QUERY,
        variables: {
            first: 1,
            username: 'username',
        },
        getConnection: result => {
            const data = dataOrThrowErrors(result)
            if (!data.user) {
                throw new Error('User not found')
            }
            return data.user.repositories
        },
        options: {
            useURL: true,
        },
    })

    return (
        <>
            <ul>
                {connection?.nodes.map((node, index) => (
                    <li key={index}>{node.name}</li>
                ))}
            </ul>

            {connection?.totalCount && <p>Total count: {connection.totalCount}</p>}
            {hasNextPage && (
                <button type="button" onClick={fetchMore}>
                    Fetch more
                </button>
            )}
        </>
    )
}

const generateMockRequest = ({
    after,
    first,
}: {
    after?: string
    first: number
}): MockedResponse<TestConnectionQueryResult>['request'] => ({
    query: getDocumentNode(TEST_CONNECTION_QUERY),
    variables: {
        username: 'username',
        after,
        first,
    },
})

const generateMockResult = ({
    endCursor,
    hasNextPage,
    nodes,
    totalCount,
}: {
    endCursor: string | null
    hasNextPage: boolean
    nodes: TestConnectionQueryFields[]
    totalCount: number
}): MockedResponse<TestConnectionQueryResult>['result'] => ({
    data: {
        user: {
            repositories: {
                nodes,
                pageInfo: {
                    endCursor,
                    hasNextPage,
                },
                totalCount,
            },
        },
    },
})

describe('useConnection', () => {
    const fetchNextPage = async (renderResult: RenderWithRouterResult) => {
        const fetchMoreButton = renderResult.getByText('Fetch more')
        fireEvent.click(fetchMoreButton)

        // Skip loading state
        await waitForNextApolloResponse()
    }

    const mockResultNodes: TestConnectionQueryFields[] = [
        {
            id: 'A',
            name: 'repo-A',
        },
        {
            id: 'B',
            name: 'repo-B',
        },
        {
            id: 'C',
            name: 'repo-C',
        },
        {
            id: 'D',
            name: 'repo-D',
        },
    ]

    const renderWithMocks = async (mocks: MockedResponse<TestConnectionQueryResult>[], route = '/') => {
        const renderResult = renderWithRouter(
            <MockedProvider mocks={mocks}>
                <TestComponent />
            </MockedProvider>,
            { route }
        )

        // Skip loading state
        await waitForNextApolloResponse()

        return renderResult
    }

    describe('Cursor based pagination', () => {
        const generateMockCursorResponses = (
            nodes: TestConnectionQueryFields[]
        ): MockedResponse<TestConnectionQueryResult>[] =>
            nodes.map((node, index) => {
                const isFirstPage = index === 0
                const cursor = !isFirstPage ? String(index) : undefined
                return {
                    request: generateMockRequest({ after: cursor, first: 1 }),
                    result: generateMockResult({
                        nodes: [node],
                        endCursor: String(index + 1),
                        hasNextPage: index !== nodes.length - 1,
                        totalCount: nodes.length,
                    }),
                }
            })

        const cursorMocks = generateMockCursorResponses(mockResultNodes)

        it('renders correct result', async () => {
            const queries = await renderWithMocks(cursorMocks)
            expect(queries.getAllByRole('listitem').length).toBe(1)
            expect(queries.getByText('repo-A')).toBeVisible()
            expect(queries.getByText('Total count: 4')).toBeVisible()
            expect(queries.getByText('Fetch more')).toBeVisible()
            expect(queries.history.location.search).toBe('')
        })

        it('fetches next page of results correctly', async () => {
            const queries = await renderWithMocks(cursorMocks)

            // Fetch first page
            await fetchNextPage(queries)

            // Both pages are now displayed
            expect(queries.getAllByRole('listitem').length).toBe(2)
            expect(queries.getByText('repo-A')).toBeVisible()
            expect(queries.getByText('repo-B')).toBeVisible()
            expect(queries.getByText('Total count: 4')).toBeVisible()
            expect(queries.getByText('Fetch more')).toBeVisible()

            // URL updates to match visible results
            expect(queries.history.location.search).toBe('?visible=2')
        })

        it('fetches final page of results correctly', async () => {
            const queries = await renderWithMocks(cursorMocks)

            // Fetch all pages
            await fetchNextPage(queries)
            await fetchNextPage(queries)
            await fetchNextPage(queries)

            // All pages of results are displayed
            expect(queries.getAllByRole('listitem').length).toBe(4)
            expect(queries.getByText('repo-A')).toBeVisible()
            expect(queries.getByText('repo-B')).toBeVisible()
            expect(queries.getByText('repo-C')).toBeVisible()
            expect(queries.getByText('repo-D')).toBeVisible()
            expect(queries.getByText('Total count: 4')).toBeVisible()

            // Fetch more button is now no longer visible
            expect(queries.queryByText('Fetch more')).not.toBeInTheDocument()

            // URL updates to match visible results
            expect(queries.history.location.search).toBe('?visible=4')
        })

        it('fetches correct amount of results when navigating directly with a URL', async () => {
            // We need to add an extra mock here, as we will derive a different `first` variable from `visible` in the URL.
            const mockFromVisible: MockedResponse<TestConnectionQueryResult> = {
                request: generateMockRequest({ first: 3 }),
                result: generateMockResult({
                    nodes: [mockResultNodes[0], mockResultNodes[1], mockResultNodes[2]],
                    hasNextPage: true,
                    endCursor: '3',
                    totalCount: 4,
                }),
            }

            const queries = await renderWithMocks([...cursorMocks, mockFromVisible], '/?visible=3')

            // Renders 3 results without having to manually fetch
            expect(queries.getAllByRole('listitem').length).toBe(3)
            expect(queries.getByText('repo-A')).toBeVisible()
            expect(queries.getByText('repo-B')).toBeVisible()
            expect(queries.getByText('repo-C')).toBeVisible()
            expect(queries.getByText('Total count: 4')).toBeVisible()

            // Fetching next page should work as usual
            await fetchNextPage(queries)
            expect(queries.getAllByRole('listitem').length).toBe(4)
            expect(queries.getByText('repo-C')).toBeVisible()
            expect(queries.getByText('repo-D')).toBeVisible()
            expect(queries.getByText('Total count: 4')).toBeVisible()

            // URL should be overidden
            expect(queries.history.location.search).toBe('?visible=4')
        })
    })

    describe('Batch based pagination', () => {
        const batchedMocks: MockedResponse<TestConnectionQueryResult>[] = [
            {
                request: generateMockRequest({ first: 1 }),
                result: generateMockResult({
                    nodes: [mockResultNodes[0]],
                    endCursor: null,
                    hasNextPage: true,
                    totalCount: 4,
                }),
            },
            {
                request: generateMockRequest({ first: 2 }),
                result: generateMockResult({
                    nodes: [mockResultNodes[0], mockResultNodes[1]],
                    endCursor: null,
                    hasNextPage: true,
                    totalCount: 4,
                }),
            },
            {
                request: generateMockRequest({ first: 4 }),
                result: generateMockResult({
                    nodes: mockResultNodes,
                    endCursor: null,
                    hasNextPage: false,
                    totalCount: 4,
                }),
            },
        ]

        it('renders correct result', async () => {
            const queries = await renderWithMocks(batchedMocks)
            expect(queries.getAllByRole('listitem').length).toBe(1)
            expect(queries.getByText('repo-A')).toBeVisible()
            expect(queries.getByText('Total count: 4')).toBeVisible()
            expect(queries.getByText('Fetch more')).toBeVisible()
            expect(queries.history.location.search).toBe('')
        })

        it('fetches next page of results correctly', async () => {
            const queries = await renderWithMocks(batchedMocks)

            // Fetch first page
            await fetchNextPage(queries)

            // Both pages are now displayed
            expect(queries.getAllByRole('listitem').length).toBe(2)
            expect(queries.getByText('repo-A')).toBeVisible()
            expect(queries.getByText('repo-B')).toBeVisible()
            expect(queries.getByText('Total count: 4')).toBeVisible()
            expect(queries.getByText('Fetch more')).toBeVisible()

            // URL updates to match the new request
            expect(queries.history.location.search).toBe('?first=2')
        })

        it('fetches final page of results correctly', async () => {
            const queries = await renderWithMocks(batchedMocks)

            // Fetch both pages
            await fetchNextPage(queries)
            await fetchNextPage(queries)

            // All pages of results are displayed
            expect(queries.getAllByRole('listitem').length).toBe(4)
            expect(queries.getByText('repo-A')).toBeVisible()
            expect(queries.getByText('repo-B')).toBeVisible()
            expect(queries.getByText('repo-C')).toBeVisible()
            expect(queries.getByText('repo-D')).toBeVisible()
            expect(queries.getByText('Total count: 4')).toBeVisible()

            // Fetch more button is now no longer visible
            expect(queries.queryByText('Fetch more')).not.toBeInTheDocument()

            // URL updates to match the new request
            expect(queries.history.location.search).toBe('?first=4')
        })

        it('fetches correct amount of results when navigating directly with a URL', async () => {
            const queries = await renderWithMocks(batchedMocks, '/?first=2')

            // Renders 2 results without having to manually fetch
            expect(queries.getAllByRole('listitem').length).toBe(2)
            expect(queries.getByText('repo-A')).toBeVisible()
            expect(queries.getByText('repo-B')).toBeVisible()
            expect(queries.getByText('Total count: 4')).toBeVisible()

            // Fetching next page should work as usual
            await fetchNextPage(queries)
            expect(queries.getAllByRole('listitem').length).toBe(4)
            expect(queries.getByText('repo-C')).toBeVisible()
            expect(queries.getByText('repo-D')).toBeVisible()
            expect(queries.getByText('Total count: 4')).toBeVisible()

            // URL should be overidden
            expect(queries.history.location.search).toBe('?first=4')
        })
    })
})
