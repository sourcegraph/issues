import React, { useMemo } from 'react'

export interface Step {
    index: number
    isFirstStep: boolean
    isLastStep: boolean
    isVisited: boolean
    isComplete: boolean
}

export interface Steps {
    [key: number]: Step
}

export interface State {
    current: number
    initialStep: number
    steps: Steps
}

export type Action =
    | { type: 'SET_CURRENT_STEP'; payload: { index: number } }
    | { type: 'SET_COMPLETE_STEP'; payload: { index: number; complete: boolean } }
    | { type: 'SET_STEPS'; payload: { steps: Steps } }

export interface StepsContext {
    state: State
    dispatch: React.Dispatch<Action>
}

export interface StepListContext {
    setCurrent: () => void
    stepIndex: number
}

interface UseSteps {
    setStep: (index: number) => void
    setComplete: (index: number, complete: boolean) => void
    currentIndex: number
    currentStep: Step
    steps: State['steps']
}

export const StepsContext = React.createContext<StepsContext | null>(null)
StepsContext.displayName = 'StepsContext'

export const StepListContext = React.createContext<StepListContext | null>(null)
StepsContext.displayName = 'StepListContext'

export const useStepListContext = (): StepListContext => {
    const context = React.useContext(StepListContext)
    if (!context) {
        throw new Error('You are trying to use this component outside the <StepList /> component')
    }
    return context
}

export const useStepsContext = (): StepsContext => {
    const context = React.useContext(StepsContext)
    if (!context) {
        throw new Error('Steps compound components cannot be rendered outside the <Steps> component')
    }
    return context
}

export const useSteps = (): UseSteps => {
    const context = useStepsContext()
    const { state, dispatch } = context

    const getters = {
        currentIndex: state.current,
        currentStep: state.steps[state.current],
        steps: state.steps,
    }

    const setters = useMemo(
        () => ({
            setStep: (index: number): void => dispatch({ type: 'SET_CURRENT_STEP', payload: { index } }),
            setComplete: (index: number, complete = true): void =>
                dispatch({ type: 'SET_COMPLETE_STEP', payload: { index, complete } }),
        }),
        [dispatch]
    )

    return { ...getters, ...setters }
}
