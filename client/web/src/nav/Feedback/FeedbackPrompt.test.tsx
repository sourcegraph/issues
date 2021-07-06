import { MockedProvider } from '@apollo/client/testing'
import { render, RenderResult, fireEvent, act } from '@testing-library/react'
import { GraphQLError } from 'graphql'
import React from 'react'

import { getDocumentNode } from '@sourcegraph/shared/src/graphql/graphql'

import { SubmitHappinessFeedbackVariables } from '../../graphql-operations'
import { routes } from '../../routes'

import { FeedbackPrompt, HAPPINESS_FEEDBACK_OPTIONS, SUBMIT_HAPPINESS_FEEDBACK_QUERY } from './FeedbackPrompt'

jest.mock('../../hooks', () => ({
    useRoutesMatch: () => '/some-route',
}))

const mockData: SubmitHappinessFeedbackVariables = {
    input: {
        score: 4,
        feedback: 'Lorem ipsum dolor sit amet',
        currentPath: '/some-route',
    },
}

describe('FeedbackPrompt', () => {
    let queries: RenderResult

    beforeAll(() => {
        window.context = { productResearchPageEnabled: true } as any
    })

    describe('layout', () => {
        beforeEach(() => {
            queries = render(
                <MockedProvider>
                    <FeedbackPrompt routes={routes} />
                </MockedProvider>
            )
        })

        test('Renders heading correctly', () => {
            expect(queries.getByText('What’s on your mind?')).toBeVisible()
        })

        test('Renders textarea correctly', () => {
            expect(queries.getByPlaceholderText('What’s going well? What could be better?')).toBeVisible()
        })

        test('Renders correct emoji toggles', () => {
            for (const option of HAPPINESS_FEEDBACK_OPTIONS) {
                expect(queries.getByLabelText(option.name)).toBeVisible()
            }
        })

        test('Send button is initially disabled', () => {
            const sendButton = queries.getByText('Send') as HTMLButtonElement
            expect(sendButton.disabled).toBe(true)
        })

        test('Send button is disabled when a happiness rating is selected and textarea is empty', () => {
            const sendButton = queries.getByText('Send') as HTMLButtonElement
            fireEvent.click(queries.getByLabelText('Very Happy'))
            expect(sendButton.disabled).toBe(true)
        })

        test('Send button is disabled when a textarea is not empty and happiness rating is selected', () => {
            const textArea = queries.getByPlaceholderText('What’s going well? What could be better?')
            const sendButton = queries.getByText('Send') as HTMLButtonElement
            fireEvent.change(textArea, { target: { value: mockData.input.feedback } })
            fireEvent.click(queries.getByLabelText('Very Happy'))
            expect(sendButton.disabled).toBe(false)
        })
    })

    describe('submission', () => {
        const mockRequest = {
            query: getDocumentNode(SUBMIT_HAPPINESS_FEEDBACK_QUERY),
            variables: mockData,
        }

        const submitFeedback = async () => {
            const textArea = queries.getByPlaceholderText('What’s going well? What could be better?')
            const radioButton = queries.getByLabelText('Very Happy')
            const sendButton = queries.getByText('Send')
            fireEvent.change(textArea, { target: { value: mockData.input.feedback } })
            fireEvent.click(radioButton)
            fireEvent.click(sendButton)

            // Wait next tick to skip loading state
            await act(() => new Promise(resolve => setTimeout(resolve, 0)))
        }

        describe('Success', () => {
            const mockResponse = {
                data: {
                    submitHappinessFeedback: {
                        alwaysNill: null,
                    },
                },
            }

            beforeEach(async () => {
                queries = render(
                    <MockedProvider mocks={[{ request: mockRequest, result: mockResponse }]}>
                        <FeedbackPrompt routes={routes} />
                    </MockedProvider>
                )

                await submitFeedback()
            })

            test('Renders success page correctly', () => {
                expect(queries.getByText(/Want to help keep making Sourcegraph better?/)).toBeVisible()
            })
        })

        describe('Error', () => {
            const mockError = new GraphQLError('Something went really wrong')
            const mockResponse = {
                errors: [mockError],
            }
            beforeEach(async () => {
                queries = render(
                    <MockedProvider
                        mocks={[
                            {
                                request: mockRequest,
                                result: mockResponse,
                            },
                        ]}
                        defaultOptions={{
                            mutate: {
                                // Fix errors being thrown globally https://github.com/apollographql/apollo-client/issues/7167
                                errorPolicy: 'all',
                            },
                        }}
                    >
                        <FeedbackPrompt routes={routes} />
                    </MockedProvider>
                )

                await submitFeedback()
            })

            test('Renders error alert correctly', () => {
                expect(queries.getByText('Error submitting feedback:')).toBeVisible()
                expect(queries.getByText(mockError.message)).toBeVisible()
            })
        })
    })
})
