import { isErrorLike } from '../util/errors'
import { hasProperty } from '../util/types'

const CLONE_IN_PROGRESS_ERROR_NAME = 'CloneInProgressError'
export class CloneInProgressError extends Error {
    public readonly name = CLONE_IN_PROGRESS_ERROR_NAME
    constructor(repoName: string, public readonly progress?: string) {
        super(`${repoName} is clone in progress`)
    }
}
// Will work even for errors that came from GraphQL, background pages, comlink webworkers, etc.
// TODO remove error message assertion after https://github.com/sourcegraph/sourcegraph/issues/9697 and https://github.com/sourcegraph/sourcegraph/issues/9693 are fixed
export const isCloneInProgressErrorLike = (value: unknown): boolean =>
    isErrorLike(value) && (value.name === CLONE_IN_PROGRESS_ERROR_NAME || /clone in progress/i.test(value.message))

const REPO_NOT_FOUND_ERROR_NAME = 'RepoNotFoundError' as const
export class RepoNotFoundError extends Error {
    public readonly name = REPO_NOT_FOUND_ERROR_NAME
    constructor(repoName: string) {
        super(`repo ${repoName} not found`)
    }
}
// Will work even for errors that came from GraphQL, background pages, comlink webworkers, etc.
// TODO remove error message assertion after https://github.com/sourcegraph/sourcegraph/issues/9697 and https://github.com/sourcegraph/sourcegraph/issues/9693 are fixed
export const isRepoNotFoundErrorLike = (value: unknown): boolean =>
    isErrorLike(value) && (value.name === REPO_NOT_FOUND_ERROR_NAME || /repo.*not found/i.test(value.message))

const REVISION_NOT_FOUND_ERROR_NAME = 'RevisionNotFoundError' as const
export class RevisionNotFoundError extends Error {
    public readonly name = REVISION_NOT_FOUND_ERROR_NAME
    constructor(revision?: string) {
        super(`Revision ${String(revision)} not found`)
    }
}
// Will work even for errors that came from GraphQL, background pages, comlink webworkers, etc.
// TODO remove error message assertion after https://github.com/sourcegraph/sourcegraph/issues/9697 and https://github.com/sourcegraph/sourcegraph/issues/9693 are fixed
export const isRevisionNotFoundErrorLike = (value: unknown): boolean =>
    isErrorLike(value) && (value.name === REVISION_NOT_FOUND_ERROR_NAME || /revision.*not found/i.test(value.message))

const REPO_SEE_OTHER_ERROR_NAME = 'RepoSeeOtherError' as const
export class RepoSeeOtherError extends Error {
    public readonly name = REPO_SEE_OTHER_ERROR_NAME
    constructor(public readonly redirectURL: string) {
        super(`Repository not found at this location, but might exist at ${redirectURL}`)
    }
}
// Will work even for errors that came from GraphQL, background pages, comlink webworkers, etc.
// TODO remove error message assertion after https://github.com/sourcegraph/sourcegraph/issues/9697 and https://github.com/sourcegraph/sourcegraph/issues/9693 are fixed
/** Returns the redirect URL if the passed value is like a RepoSeeOtherError, otherwise `false`. */
export const isRepoSeeOtherErrorLike = (value: unknown): string | false => {
    if (!isErrorLike(value)) {
        return false
    }
    if (
        value.name === REPO_SEE_OTHER_ERROR_NAME &&
        hasProperty('redirectURL')(value) &&
        typeof value.redirectURL === 'string'
    ) {
        return value.redirectURL
    }
    const match = value.message.match(/repository not found at this location, but might exist at (\S+)/i)
    if (match) {
        return match[1]
    }
    return false
}

const PRIVATE_REPO_PUBLIC_SOURCEGRAPH_COM_ERROR_NAME = 'PrivateRepoPublicSourcegraphError' as const
/**
 * An Error that means that the current repository is private and the current
 * Sourcegraph URL is Sourcegraph.com. Requests made from a private repository
 * to Sourcegraph.com are blocked unless the `requestMightContainPrivateInfo`
 * argument to `requestGraphQL` is explicitly set to false (defaults to true to
 * be conservative).
 */
export class PrivateRepoPublicSourcegraphComError extends Error {
    public readonly name = PRIVATE_REPO_PUBLIC_SOURCEGRAPH_COM_ERROR_NAME
    constructor(graphQLName: string) {
        super(
            `A ${graphQLName} GraphQL request to the public Sourcegraph.com was blocked because the current repository is private.`
        )
    }
}
// Will work even for errors that came from GraphQL, background pages, comlink webworkers, etc.
// TODO remove error message assertion after https://github.com/sourcegraph/sourcegraph/issues/9697 and https://github.com/sourcegraph/sourcegraph/issues/9693 are fixed
export const isPrivateRepoPublicSourcegraphComErrorLike = (value: unknown): boolean =>
    isErrorLike(value) &&
    (value.name === PRIVATE_REPO_PUBLIC_SOURCEGRAPH_COM_ERROR_NAME ||
        /graphql request to the public sourcegraph.com was blocked because the current repository is private/i.test(
            value.message
        ))
