import * as H from 'history'

type ActivationID = 'ConnectedCodeHost' | 'EnabledRepository' | 'DidSearch' | 'FoundReferences' | 'EnabledSharing'

/**
 * Represents the activation status of the current user.
 */
export interface Activation {
    /**
     * The steps that make up the activation list
     */
    steps: ActivationStep[]

    /**
     * The completion status of each activation step
     */
    completed?: ActivationCompletionStatus

    /**
     * Updates the activation status with the given steps and their completion status.
     */
    update: (u: ActivationCompletionStatus) => void

    /**
     * Resync the activation status from the server.
     */
    refetch: () => void
}

/**
 * A map indicating which activation steps have been completed
 */
export type ActivationCompletionStatus = { [K in ActivationID]?: boolean }

/**
 * Component props should inherit from this to include activation status.
 */
export interface ActivationProps {
    activation?: Activation
}

/**
 * One step in the activation status.
 */
export interface ActivationStep {
    /**
     * The identifier for the activation step
     */
    id: ActivationID

    /**
     * The title of the step to display in the activation dropdown
     */
    title: string

    /**
     * Description of the step displayed in a popover
     */
    detail: React.ReactNode

    /**
     * If set, the handler should be invoked when the user attempts
     * to complete this step.
     */
    onClick?: (event: React.MouseEvent<HTMLElement>, history: H.History) => void
}

/**
 * Returns the percent of activation checklist items completed.
 */
export const percentageDone = (info?: ActivationCompletionStatus): number => {
    if (!info) {
        return 0
    }
    const values = Object.values(info)
    return (100 * values.filter(done => done).length) / values.length
}
