import { render, RenderResult, cleanup } from '@testing-library/react'
import React from 'react'
import sinon from 'sinon'

import { Steps, Step, StepsProps, StepList, StepPanel, StepPanels } from '.'

describe('Steps', () => {
    let queries: RenderResult

    const renderWithProps = (props: StepsProps): RenderResult =>
        render(<Steps initialStep={props.initialStep}>{props.children}</Steps>)
    const onChangeMock = sinon.spy()

    beforeEach(() => {
        onChangeMock.resetHistory()
    })

    afterEach(cleanup)

    describe('Invalid configuration', () => {
        it('will error when initial step is less than 1', () => {
            expect(() =>
                renderWithProps({
                    initialStep: -1,
                    children: [
                        <StepList key={1} numeric={true}>
                            <Step borderColor="blue">Panel 1 title</Step>
                        </StepList>,
                        <StepPanels key={2}>
                            <StepPanel>Panel 1</StepPanel>
                        </StepPanels>,
                    ],
                })
            ).toThrowErrorMatchingSnapshot()
        })

        it('will error when initial step is more than Steps length', () => {
            expect(() =>
                renderWithProps({
                    initialStep: 10,
                    children: [
                        <StepList key={1} numeric={true}>
                            <Step borderColor="blue">Panel 1 title</Step>
                        </StepList>,
                        <StepPanels key={2}>
                            <StepPanel>Panel 1</StepPanel>
                        </StepPanels>,
                    ],
                })
            ).toThrowErrorMatchingSnapshot()
        })

        it('will error when initial step is more than <Step> components length', () => {
            expect(() =>
                renderWithProps({
                    initialStep: 2,
                    children: [
                        <StepList key={1} numeric={true}>
                            <Step borderColor="blue">Panel 1 title</Step>
                        </StepList>,
                        <StepPanels key={2}>
                            <StepPanel>Panel 1</StepPanel>
                        </StepPanels>,
                    ],
                })
            ).toThrowErrorMatchingSnapshot()
        })

        it('will error when StepPanels does not includes children', () => {
            expect(() =>
                renderWithProps({
                    initialStep: 1,
                    children: [
                        <StepList key={1} numeric={true}>
                            <Step borderColor="blue">Panel 1 title</Step>
                        </StepList>,
                        <StepPanels key={2} />,
                    ],
                })
            ).toThrowErrorMatchingSnapshot()
        })
    })

    describe('Steps navigation', () => {
        const initialStep = 1
        beforeEach(() => {
            queries = renderWithProps({
                initialStep,
                children: [
                    <>
                        <StepList key={1} numeric={true}>
                            <Step key={1} borderColor="blue">
                                Panel 1 title
                            </Step>
                            <Step key={2} borderColor="orange">
                                Panel 2 title
                            </Step>
                            <Step key={3} borderColor="orange">
                                Panel 3 title
                            </Step>
                        </StepList>
                        <StepPanels key={2}>
                            <StepPanel key={2}>Panel 1</StepPanel>
                            <StepPanel key={1}>Panel 2</StepPanel>
                            <StepPanel key={3}>Panel 3</StepPanel>
                        </StepPanels>
                    </>,
                ],
            })
        })

        it('Will render all the <Step> elements', () => {
            expect(queries.getByText('Panel 1 title', { selector: 'button' })).toBeInTheDocument()
            expect(queries.getByText('Panel 2 title', { selector: 'button' })).toBeInTheDocument()
            expect(queries.getByText('Panel 3 title', { selector: 'button' })).toBeInTheDocument()
        })

        it('<Step> is enabled when is visited or current', () => {
            expect(queries.getByText('Panel 1 title', { selector: 'button' })).toBeEnabled()
            expect(queries.getByText('Panel 2 title', { selector: 'button' })).toBeDisabled()
            expect(queries.getByText('Panel 3 title', { selector: 'button' })).toBeDisabled()
        })

        it('Will render the <StepPanel> element associated to their respective <Step> element', () => {
            expect(queries.getByText('Panel 1')).toBeInTheDocument()
            expect(queries.queryByText('Panel 2')).toBe(null)
            expect(queries.queryByText('Panel 3')).toBe(null)
        })
    })
})
