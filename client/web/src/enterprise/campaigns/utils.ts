import {
    ChangesetExternalState,
    ChangesetCheckState,
    ChangesetReviewState,
    ChangesetFields,
    ChangesetReconcilerState,
    ChangesetPublicationState,
} from '../../graphql-operations'
import { HoveredToken } from '@sourcegraph/codeintellify'
import {
    RepoSpec,
    RevisionSpec,
    FileSpec,
    ResolvedRevisionSpec,
    UIPositionSpec,
    ModeSpec,
} from '../../../../shared/src/util/url'
import { getModeFromPath } from '../../../../shared/src/languages'

export function isValidChangesetExternalState(input: string): input is ChangesetExternalState {
    return Object.values<string>(ChangesetExternalState).includes(input)
}

export function isValidChangesetReviewState(input: string): input is ChangesetReviewState {
    return Object.values<string>(ChangesetReviewState).includes(input)
}

export function isValidChangesetCheckState(input: string): input is ChangesetCheckState {
    return Object.values<string>(ChangesetCheckState).includes(input)
}

export function getLSPTextDocumentPositionParameters(
    hoveredToken: HoveredToken & RepoSpec & RevisionSpec & FileSpec & ResolvedRevisionSpec
): RepoSpec & RevisionSpec & ResolvedRevisionSpec & FileSpec & UIPositionSpec & ModeSpec {
    return {
        repoName: hoveredToken.repoName,
        revision: hoveredToken.revision,
        filePath: hoveredToken.filePath,
        commitID: hoveredToken.commitID,
        position: hoveredToken,
        mode: getModeFromPath(hoveredToken.filePath || ''),
    }
}

export enum ChangesetUIState {
    UNPUBLISHED = 'UNPUBLISHED',
    ERRORED = 'ERRORED',
    PROCESSING = 'PROCESSING',
    OPEN = 'OPEN',
    DRAFT = 'DRAFT',
    CLOSED = 'CLOSED',
    MERGED = 'MERGED',
    DELETED = 'DELETED',
}

export function isValidChangesetUIState(input: string): input is ChangesetUIState {
    return Object.values<string>(ChangesetUIState).includes(input)
}

export function computeChangesetUIState(
    changeset: Pick<ChangesetFields, 'reconcilerState' | 'publicationState' | 'externalState'>
): ChangesetUIState {
    if (changeset.reconcilerState === ChangesetReconcilerState.ERRORED) {
        return ChangesetUIState.ERRORED
    }
    if (changeset.reconcilerState !== ChangesetReconcilerState.COMPLETED) {
        return ChangesetUIState.PROCESSING
    }
    if (changeset.publicationState === ChangesetPublicationState.UNPUBLISHED) {
        return ChangesetUIState.UNPUBLISHED
    }
    // Must be set, because changesetPublicationState !== UNPUBLISHED.
    const externalState = changeset.externalState!
    switch (externalState) {
        case ChangesetExternalState.DRAFT:
            return ChangesetUIState.DRAFT
        case ChangesetExternalState.OPEN:
            return ChangesetUIState.OPEN
        case ChangesetExternalState.CLOSED:
            return ChangesetUIState.CLOSED
        case ChangesetExternalState.MERGED:
            return ChangesetUIState.MERGED
        case ChangesetExternalState.DELETED:
            return ChangesetUIState.DELETED
    }
}
