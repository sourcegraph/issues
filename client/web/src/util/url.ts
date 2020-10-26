import { Position, Range } from '@sourcegraph/extension-api-types'
import {
    encodeRepoRevision,
    LineOrPositionOrRange,
    lprToRange,
    ParsedRepoURI,
    parseHash,
    RepoFile,
    toPositionHashComponent,
} from '../../../shared/src/util/url'

export function toTreeURL(target: RepoFile): string {
    return `/${encodeRepoRevision(target)}/-/tree/${target.filePath}`
}

/**
 * Returns the LineOrPositionOrRange and given URLSearchParams as a string.
 */
export function formatHash(lpr: LineOrPositionOrRange, searchParameters: URLSearchParams): string {
    if (!lpr.line) {
        return `#${searchParameters.toString()}`
    }
    const anyParameters = [...searchParameters].length > 0
    return `#L${formatLineOrPositionOrRange(lpr)}${anyParameters ? '&' + searchParameters.toString() : ''}`
}

/**
 * Returns the textual form of the LineOrPositionOrRange suitable for encoding
 * in a URL fragment' query parameter.
 *
 * @param lpr The `LineOrPositionOrRange`
 */
function formatLineOrPositionOrRange(lpr: LineOrPositionOrRange): string {
    const range = lprToRange(lpr)
    if (!range) {
        return ''
    }
    const emptyRange = range.start.line === range.end.line && range.start.character === range.end.character
    return emptyRange
        ? toPositionHashComponent(range.start)
        : `${toPositionHashComponent(range.start)}-${toPositionHashComponent(range.end)}`
}

/**
 * Replaces the revision in the given URL, or adds one if there is not already
 * one.
 *
 * @param href The URL whose revision should be replaced.
 */
export function replaceRevisionInURL(href: string, newRevision: string): string {
    const parsed = parseBrowserRepoURL(href)
    const repoRevision = `/${encodeRepoRevision(parsed)}`

    const url = new URL(href, window.location.href)
    url.pathname = `/${encodeRepoRevision({ ...parsed, revision: newRevision })}${url.pathname.slice(
        repoRevision.length
    )}`
    return `${url.pathname}${url.search}${url.hash}`
}

/**
 * Parses the properties of a blob URL.
 */
export function parseBrowserRepoURL(href: string): ParsedRepoURI & Pick<ParsedRepoRevision, 'rawRevision'> {
    const url = new URL(href, window.location.href)
    let pathname = url.pathname.slice(1) // trim leading '/'
    if (pathname.endsWith('/')) {
        pathname = pathname.slice(0, -1) // trim trailing '/'
    }

    const indexOfSeparator = pathname.indexOf('/-/')

    // examples:
    // - 'github.com/gorilla/mux'
    // - 'github.com/gorilla/mux@revision'
    // - 'foo/bar' (from 'sourcegraph.mycompany.com/foo/bar')
    // - 'foo/bar@revision' (from 'sourcegraph.mycompany.com/foo/bar@revision')
    // - 'foobar' (from 'sourcegraph.mycompany.com/foobar')
    // - 'foobar@revision' (from 'sourcegraph.mycompany.com/foobar@revision')
    let repoRevision: string
    if (indexOfSeparator === -1) {
        repoRevision = pathname // the whole string
    } else {
        repoRevision = pathname.slice(0, indexOfSeparator) // the whole string leading up to the separator (allows revision to be multiple path parts)
    }
    const { repoName, revision, rawRevision } = parseRepoRevision(repoRevision)
    if (!repoName) {
        throw new Error('unexpected repo url: ' + href)
    }
    const commitID = revision && /^[\da-f]{40}$/i.test(revision) ? revision : undefined

    let filePath: string | undefined
    let commitRange: string | undefined
    const treeSeparator = pathname.indexOf('/-/tree/')
    const blobSeparator = pathname.indexOf('/-/blob/')
    const comparisonSeparator = pathname.indexOf('/-/compare/')
    if (treeSeparator !== -1) {
        filePath = decodeURIComponent(pathname.slice(treeSeparator + '/-/tree/'.length))
    }
    if (blobSeparator !== -1) {
        filePath = decodeURIComponent(pathname.slice(blobSeparator + '/-/blob/'.length))
    }
    if (comparisonSeparator !== -1) {
        commitRange = pathname.slice(comparisonSeparator + '/-/compare/'.length)
    }
    let position: Position | undefined
    let range: Range | undefined
    if (url.hash) {
        const parsedHash = parseHash(url.hash.slice('#'.length))
        if (parsedHash.line) {
            position = {
                line: parsedHash.line,
                character: parsedHash.character || 0,
            }
            if (parsedHash.endLine) {
                range = {
                    start: position,
                    end: {
                        line: parsedHash.endLine,
                        character: parsedHash.endCharacter || 0,
                    },
                }
            }
        }
    }

    return { repoName, revision, rawRevision, commitID, filePath, commitRange, position, range }
}

/** The results of parsing a repo-revision string like "my/repo@my/revision". */
export interface ParsedRepoRevision {
    repoName: string

    /** The URI-decoded revision (e.g., "my#branch" in "my/repo@my%23branch"). */
    revision?: string

    /** The raw revision (e.g., "my%23branch" in "my/repo@my%23branch"). */
    rawRevision?: string
}

/**
 * Parses a repo-revision string like "my/repo@my/revision" to the repo and revision components.
 */
export function parseRepoRevision(repoRevision: string): ParsedRepoRevision {
    const [repository, revision] = repoRevision.split('@', 2) as [string, string | undefined]
    return {
        repoName: decodeURIComponent(repository),
        revision: revision && decodeURIComponent(revision),
        rawRevision: revision,
    }
}
