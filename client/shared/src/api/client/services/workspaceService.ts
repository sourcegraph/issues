import { WorkspaceRoot } from '@sourcegraph/extension-api-types'
import { BehaviorSubject, NextObserver, Subscribable } from 'rxjs'

// TODO (simon) This is essentially global mutable object that is proxied to extensions
// try to rework this pattern later

/**
 * A workspace root with additional metadata that is not exposed to extensions.
 */
export interface WorkspaceRootWithMetadata extends WorkspaceRoot {
    /**
     * The original input Git revision that the user requested. The {@link WorkspaceRoot#uri} value will contain
     * the Git commit SHA resolved from the input revision, but it is useful to also know the original revision
     * (e.g., to construct URLs for the user that don't result in them navigating from a branch view to a commit
     * SHA view).
     *
     * For example, if the user is viewing the web page https://github.com/alice/myrepo/blob/master/foo.js (note
     * that the URL contains a Git revision "master"), the input revision is "master".
     *
     * The empty string is a valid value (meaning that the default should be used, such as "HEAD" in Git) and is
     * distinct from undefined. If undefined, the Git commit SHA from {@link WorkspaceRoot#uri} should be used.
     */
    inputRevision?: string
}

/**
 * The workspace service manages the workspace root folders.
 */
export interface WorkspaceService {
    /**
     * An observable whose value is the list of workspace root folder URIs.
     */
    readonly roots: Subscribable<readonly WorkspaceRootWithMetadata[]> & {
        readonly value: readonly WorkspaceRootWithMetadata[]
    } & NextObserver<readonly WorkspaceRootWithMetadata[]>

    /**
     * An observable whose value is the current version context.
     */
    readonly versionContext: Subscribable<string | undefined> & NextObserver<string | undefined>
}

/**
 * Creates a {@link WorkspaceService} instance.
 */
export function createWorkspaceService(): WorkspaceService {
    const roots = new BehaviorSubject<readonly WorkspaceRootWithMetadata[]>([])
    const versionContext = new BehaviorSubject<string | undefined>(undefined)
    return {
        roots,
        versionContext,
    }
}
