import { render, RenderResult, act, within, BoundFunction, GetByRole, cleanup, fireEvent } from '@testing-library/react'
import * as React from 'react'
import { MemoryRouter } from 'react-router-dom'
import sinon from 'sinon'

import { asError } from '@sourcegraph/shared/src/util/errors'

import { FORM_ERROR } from '../../../../../../components/form/hooks/useForm'
import { InsightsApiContext } from '../../../../../../core/backend/api-provider'
import { createMockInsightAPI } from '../../../../../../core/backend/insights-api'
import { SupportedInsightSubject } from '../../../../../../core/types/subjects'

import { SearchInsightCreationContent, SearchInsightCreationContentProps } from './SearchInsightCreationContent'

const USER_TEST_SUBJECT: SupportedInsightSubject = {
    __typename: 'User' as const,
    id: 'user_test_id',
    username: 'testusername',
    displayName: 'test',
    viewerCanAdminister: true,
}

const SITE_TEST_SUBJECT: SupportedInsightSubject = {
    __typename: 'Site' as const,
    viewerCanAdminister: true,
    allowSiteSettingsEdits: true,
    id: 'global_id',
}

describe('CreateInsightContent', () => {
    const mockAPI = createMockInsightAPI({
        getRepositorySuggestions: () => Promise.resolve([]),
    })

    const renderWithProps = (props: SearchInsightCreationContentProps): RenderResult =>
        render(
            <MemoryRouter>
                <InsightsApiContext.Provider value={mockAPI}>
                    <SearchInsightCreationContent {...props} subjects={[USER_TEST_SUBJECT, SITE_TEST_SUBJECT]} />
                </InsightsApiContext.Provider>
            </MemoryRouter>
        )
    const onSubmitMock = sinon.spy()

    beforeEach(() => onSubmitMock.resetHistory())
    afterEach(cleanup)

    const getFormFields = (getByRole: BoundFunction<GetByRole>) => {
        const title = getByRole('textbox', { name: /title/i })
        const repoGroup = getByRole('group', { name: /list of repositories/i })
        const repositories = within(repoGroup).getByRole('combobox')

        const personalVisibility = getByRole('radio', { name: /private/i })
        const organisationVisibility = getByRole('radio', { name: /organization/i })

        const dataSeriesGroup = getByRole('group', { name: /data series/i })
        const seriesName = within(dataSeriesGroup).getByRole('textbox', { name: /name/i })
        const seriesQuery = within(dataSeriesGroup).getByRole('textbox', { name: /query/i })

        const seriesColorGroup = within(dataSeriesGroup).getByRole('group', { name: /color/i })

        const seriesColorRadioButtons = within(seriesColorGroup).getAllByRole('radio') as HTMLInputElement[]

        const stepGroup = getByRole('group', { name: /granularity/i })

        const stepValue = within(stepGroup).getByRole('spinbutton')

        const stepRadioButtons = within(stepGroup).getAllByRole('radio')

        return {
            title,
            repositories,
            personalVisibility,
            organisationVisibility,
            dataSeriesGroup,
            seriesName,
            seriesQuery,
            seriesColorRadioButtons,
            stepValue,
            stepRadioButtons,
        }
    }

    describe('with common fill flow', () => {
        it('will render standard package of form fields', () => {
            const { getByRole } = renderWithProps({ onSubmit: onSubmitMock })

            const {
                title,
                repositories,
                personalVisibility,
                organisationVisibility,
                seriesName,
                seriesQuery,
                seriesColorRadioButtons,
                stepValue,
                stepRadioButtons,
            } = getFormFields(getByRole)

            expect(title).toBeInTheDocument()
            expect(repositories).toBeInTheDocument()
            expect(personalVisibility).toBeInTheDocument()
            expect(organisationVisibility).toBeInTheDocument()
            expect(seriesName).toBeInTheDocument()
            expect(seriesQuery).toBeInTheDocument()

            // Since we use 12 standard open color as a default colors for color picker
            expect(seriesColorRadioButtons.length).toBe(12)

            expect(stepValue).toBeInTheDocument()
            // Since we have 5 options for insight step (hours, days, weeks, months, years)
            expect(stepRadioButtons.length).toBe(5)
        })

        it('will fire onSubmit if all fields have been filled with valid value', async () => {
            const { getByRole } = renderWithProps({ onSubmit: onSubmitMock })
            const {
                title,
                repositories,
                organisationVisibility,
                dataSeriesGroup,
                seriesName,
                seriesQuery,
                stepValue,
            } = getFormFields(getByRole)

            // eslint-disable-next-line @typescript-eslint/require-await
            await act(async () => {
                fireEvent.change(title, { target: { value: 'First code insight' } })
                fireEvent.change(repositories, { target: { value: 'github.com/sourcegraph/sourcegraph' } })
                fireEvent.click(organisationVisibility)

                const submitSeriesButton = within(dataSeriesGroup).getByRole('button', { name: /submit/i })

                const yellowColorRadio = within(dataSeriesGroup).getByRole('radio', { name: /yellow/i })

                fireEvent.change(seriesName, { target: { value: 'First code insight series' } })
                fireEvent.change(seriesQuery, { target: { value: 'patternType:regex case:yes \\*\\sas\\sGQL' } })
                fireEvent.click(yellowColorRadio)
                fireEvent.click(submitSeriesButton)

                const monthsRadio = getByRole('radio', { name: /months/i })

                fireEvent.change(stepValue, { target: { value: 2 } })
                fireEvent.click(monthsRadio)

                const submitButton = getByRole('button', { name: /create code insight/i })

                fireEvent.click(submitButton)

                // Since async repositories validation didn't pass
                sinon.assert.notCalled(onSubmitMock)
            })
        })
    })

    describe('show error massage', () => {
        it('with invalid title field', async () => {
            const { getByRole, getByText } = renderWithProps({
                onSubmit: onSubmitMock,
            })
            const repoGroup = getByRole('group', { name: /list of repositories/i })
            const repositories = within(repoGroup).getByRole('combobox')
            const submitButton = getByRole('button', { name: /create code insight/i })

            // eslint-disable-next-line @typescript-eslint/require-await
            await act(async () => {
                fireEvent.click(submitButton)
            })

            sinon.assert.notCalled(onSubmitMock)

            expect(repositories).toHaveFocus()
            expect(getByText(/title is a required/i)).toBeInTheDocument()
        })

        it('with invalid repository field', async () => {
            const { getByRole, getByText } = renderWithProps({
                onSubmit: onSubmitMock,
            })
            const title = getByRole('textbox', { name: /title/i })

            const repoGroup = getByRole('group', { name: /list of repositories/i })
            const repositories = within(repoGroup).getByRole('combobox')
            const submitButton = getByRole('button', { name: /create code insight/i })

            fireEvent.change(title, { target: { value: 'First code insight' } })

            // eslint-disable-next-line @typescript-eslint/require-await
            await act(async () => {
                fireEvent.click(submitButton)
            })

            sinon.assert.notCalled(onSubmitMock)

            expect(repositories).toHaveFocus()
            expect(getByText(/repositories is a required/i)).toBeInTheDocument()
        })

        it('with invalid data series field', async () => {
            const { getByRole, getByText } = renderWithProps({
                onSubmit: onSubmitMock,
            })
            const title = getByRole('textbox', { name: /title/i })
            const repoGroup = getByRole('group', { name: /list of repositories/i })
            const repositories = within(repoGroup).getByRole('combobox')
            const submitButton = getByRole('button', { name: /create code insight/i })
            const dataSeriesGroup = getByRole('group', { name: /data series/i })
            const seriesName = within(dataSeriesGroup).getByRole('textbox', { name: /name/i })

            fireEvent.change(title, { target: { value: 'First code insight' } })
            fireEvent.change(repositories, { target: { value: 'github.com/sourcegraph/sourcegraph' } })

            // eslint-disable-next-line @typescript-eslint/require-await
            await act(async () => {
                fireEvent.click(submitButton)
            })

            sinon.assert.notCalled(onSubmitMock)

            expect(seriesName).toHaveFocus()
            expect(getByText(/series is invalid/i)).toBeInTheDocument()
        })

        // Get it back when https://github.com/sourcegraph/sourcegraph/issues/21907 will be resolved
        // Since we don't have control over async validation handler for repositories field
        // we can mock validation response in unit.
        it.skip('when onSubmit threw submit error', async () => {
            const onSubmit = () => ({ [FORM_ERROR]: asError(new Error('Submit error')) })
            const { getByRole, getByText } = renderWithProps({ onSubmit })
            const {
                title,
                repositories,
                organisationVisibility,
                dataSeriesGroup,
                seriesName,
                seriesQuery,
                stepValue,
            } = getFormFields(getByRole)

            fireEvent.change(title, { target: { value: 'First code insight' } })
            fireEvent.change(repositories, { target: { value: 'github.com/sourcegraph/sourcegraph' } })
            fireEvent.click(organisationVisibility)

            const submitSeriesButton = within(dataSeriesGroup).getByRole('button', { name: /submit/i })

            const yellowColorRadio = within(dataSeriesGroup).getByRole('radio', { name: /yellow/i })

            fireEvent.change(seriesName, { target: { value: 'First code insight series' } })
            fireEvent.change(seriesQuery, { target: { value: 'patternType:regex case:yes \\*\\sas\\sGQL' } })
            fireEvent.click(yellowColorRadio)
            fireEvent.click(submitSeriesButton)

            const monthsRadio = getByRole('radio', { name: /months/i })

            fireEvent.change(stepValue, { target: { value: 2 } })
            fireEvent.click(monthsRadio)

            const submitButton = getByRole('button', { name: /create code insight/i })

            // eslint-disable-next-line @typescript-eslint/require-await
            await act(async () => {
                fireEvent.click(submitButton)
            })

            expect(getByText(/submit error/i)).toBeInTheDocument()
        })
    })

    // TODO [VK] Add test case for async validation
    // TODO [VK] Add test case for live-preview showing data logic
    // TODO [VK] Add test case for title validation
})
