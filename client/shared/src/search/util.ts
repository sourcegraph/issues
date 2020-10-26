/**
 * Version contexts are a lists of repos at specific revisions.
 * When a version context is set, searching and browsing is restricted to
 * the repos and revisions in the version context.
 */
export interface VersionContextProps {
    /**
     * The currently selected version context. This is undefined when there is no version context active.
     * When undefined, we use the default behavior, which allows access to all repos.
     */
    versionContext: string | undefined
}
