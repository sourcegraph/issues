import { Meta, Story } from '@storybook/react'
import React from 'react'

import { BrandedStory } from '@sourcegraph/branded/src/components/BrandedStory'
import webStyles from '@sourcegraph/web/src/SourcegraphWebApp.scss'

import { Container } from '..'

import { Steps, Step, StepList, StepPanels, StepPanel, StepActions, useSteps } from '.'

const Actions = () => {
    const { setStep, currentIndex, currentStep } = useSteps()
    return (
        <>
            <button
                disabled={currentStep.isFirstStep}
                className="btn btn-primary"
                onClick={() => setStep(currentIndex - 1)}
            >
                Previous
            </button>
            <button
                disabled={currentStep.isLastStep}
                className="btn btn-primary"
                onClick={() => setStep(currentIndex + 1)}
            >
                Next
            </button>
        </>
    )
}

export const Stepper: Story = () => (
    <BrandedStory styles={webStyles}>
        {() => (
            <Container>
                <Steps initialStep={2}>
                    <StepList numeric={true}>
                        <Step borderColor="blue">Panel 1 title</Step>
                        <Step borderColor="orange">Panel 2 Title</Step>
                        <Step borderColor="purple">Panel 3 Title</Step>
                    </StepList>
                    <StepPanels>
                        <StepPanel>Panel 1</StepPanel>
                        <StepPanel>Panel 2</StepPanel>
                        <StepPanel>Panel 3</StepPanel>
                    </StepPanels>
                    <StepActions>
                        <Actions />
                    </StepActions>
                </Steps>
            </Container>
        )}
    </BrandedStory>
)

Stepper.storyName = 'Steps component top navigation'

// eslint-disable-next-line import/no-default-export
export default {
    title: 'wildcard/Steps',
    component: Stepper,
} as Meta
