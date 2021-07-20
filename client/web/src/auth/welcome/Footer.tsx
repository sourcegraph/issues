import React from 'react'

import { LoaderButton } from '@sourcegraph/web/src/components/LoaderButton'
import { useSteps } from '@sourcegraph/wildcard/src/components/Steps/context'

interface Props {
    onFinish: () => void
}

export const Footer: React.FunctionComponent<Props> = ({ onFinish }) => {
    const { setStep, currentIndex, currentStep } = useSteps()

    return (
        <div className="mt-4">
            <LoaderButton
                type="button"
                alwaysShowLabel={true}
                label={currentStep.isLastStep ? 'Start searching' : 'Continue'}
                className="btn btn-primary float-right ml-2"
                disabled={!currentStep.isComplete}
                onClick={currentStep.isLastStep ? onFinish : () => setStep(currentIndex + 1)}
            />

            {!currentStep.isLastStep && (
                <button
                    type="button"
                    className="btn btn-link font-weight-normal text-secondary float-right"
                    onClick={onFinish}
                >
                    Not right now
                </button>
            )}
        </div>
    )
}
