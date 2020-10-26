import { first } from 'lodash'
import { Observable } from 'rxjs'
import { fromFetch } from 'rxjs/fetch'
import { filter, map } from 'rxjs/operators'
import { memoizeObservable } from '../../../../../shared/src/util/memoizeObservable'
import { isDefined } from '../../../../../shared/src/util/types'
import { DiffResolvedRevisionSpec } from '../../repo'
import { BitbucketRepoInfo } from './scrape'
import { checkOk } from '../../../../../shared/src/backend/fetch'

//
// PR API /rest/api/1.0/projects/SG/repos/go-langserver/pull-requests/1

/**
 * Builds the URL to the Bitbucket Server REST API endpoint for the given project/repo/path.
 *
 * `path` should have a leading slash.
 * `project` and `repoSlug` should have neither a leading nor a traling slash.
 */
const buildURL = (project: string, repoSlug: string, path: string): string =>
    // If possible, use the global `AJS.contextPath()` to reliably construct an absolute URL.
    // This is possible in the native integration only - browser extension content scripts cannot
    // access the page's global scope.
    `${window.AJS ? window.AJS.contextPath() : window.location.origin}/rest/api/1.0/projects/${encodeURIComponent(
        project
    )}/repos/${repoSlug}${path}`

const get = <T>(url: string): Observable<T> =>
    fromFetch(url, { selector: response => checkOk(response).json() as Promise<T> })

interface Repo {
    project: { key: string }
    name: string
    public: boolean
}

interface Reference {
    /**
     * The branch name.
     */
    displayId: string
    /**
     * The commit ID.
     */
    latestCommit: string

    repository: Repo
}

interface PRResponse {
    fromRef: Reference
    toRef: Reference
}

/**
 * Get the base commit ID for a merge request.
 */
export const getCommitsForPR: (
    info: BitbucketRepoInfo & { prID: number }
) => Observable<DiffResolvedRevisionSpec> = memoizeObservable(
    ({ project, repoSlug, prID }) =>
        get<PRResponse>(buildURL(project, repoSlug, `/pull-requests/${prID}`)).pipe(
            map(({ fromRef, toRef }) => ({ baseCommitID: toRef.latestCommit, headCommitID: fromRef.latestCommit }))
        ),
    ({ prID }) => prID.toString()
)

interface GetBaseCommitInput extends BitbucketRepoInfo {
    commitID: string
}

interface Commit {
    id: string
}

interface CommitResponse {
    parents: Commit[]
}

// Commit API /rest/api/1.0/projects/SG/repos/go-langserver/commits/b8a948dc75cc9d0c01ece01d0ba9d1eeace573aa
export const getBaseCommit: (info: GetBaseCommitInput) => Observable<string> = memoizeObservable(
    ({ project, repoSlug, commitID }) =>
        get<CommitResponse>(buildURL(project, repoSlug, `/commits/${commitID}`)).pipe(
            map(({ parents }) => first(parents)),
            filter(isDefined),
            map(({ id }) => id)
        ),
    ({ project, repoSlug, commitID }) => `${project}:${repoSlug}:${commitID}`
)
