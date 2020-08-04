import {
    ChangesetExternalState,
    ChangesetCheckState,
    ChangesetReviewState,
    ChangesetState,
} from '../../graphql-operations'

export function isValidChangesetState(input: string): input is ChangesetState {
    return Object.values<string>(ChangesetState).includes(input)
}

export function isValidChangesetExternalState(input: string): input is ChangesetExternalState {
    return Object.values<string>(ChangesetExternalState).includes(input)
}

export function isValidChangesetReviewState(input: string): input is ChangesetReviewState {
    return Object.values<string>(ChangesetReviewState).includes(input)
}

export function isValidChangesetCheckState(input: string): input is ChangesetCheckState {
    return Object.values<string>(ChangesetCheckState).includes(input)
}
