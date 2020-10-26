import { RawRepoSpec, RevisionSpec } from '../../../../../shared/src/util/url'
import { CodeHostContext } from '../shared/codeHost'

// example pathname: /projects/TEST/repos/some-repo/browse/src/extension.ts
const PATH_REGEX = /\/projects\/([^/]+)\/repos\/([^/]+)\//

function getRawRepoSpecFromLocation(location: Pick<Location, 'hostname' | 'pathname'>): RawRepoSpec {
    const { hostname, pathname } = location
    const match = pathname.match(PATH_REGEX)
    if (!match) {
        throw new Error(`location pathname does not match path regex: ${pathname}`)
    }
    const [, projectName, repoName] = match
    return {
        rawRepoName: `${hostname}/${projectName}/${repoName}`,
    }
}

interface RevisionReferenceInfo {
    latestCommit?: string
}

function getRevisionSpecFromRevisionSelector(): RevisionSpec {
    const branchNameElement = document.querySelector('#repository-layout-revision-selector .name[data-revision-ref]')
    if (!branchNameElement) {
        throw new Error('branchNameElement not found')
    }
    const revisionReferenceString = branchNameElement.getAttribute('data-revision-ref')
    let revisionReferenceInfo: RevisionReferenceInfo | null = null
    if (revisionReferenceString) {
        try {
            revisionReferenceInfo = JSON.parse(revisionReferenceString)
        } catch {
            throw new Error(`Could not parse revisionRefStr: ${revisionReferenceString}`)
        }
    }
    if (revisionReferenceInfo?.latestCommit) {
        return {
            revision: revisionReferenceInfo.latestCommit,
        }
    }
    throw new Error(
        `revisionRefInfo is empty or has no latestCommit (revisionRefStr: ${String(revisionReferenceString)})`
    )
}

export function getContext(): CodeHostContext {
    const repoSpec = getRawRepoSpecFromLocation(window.location)
    let revisionSpec: Partial<RevisionSpec> = {}
    try {
        revisionSpec = getRevisionSpecFromRevisionSelector()
    } catch {
        // RevSpec is optional in CodeHostContext
    }
    return {
        ...repoSpec,
        ...revisionSpec,
        privateRepository: window.location.hostname !== 'bitbucket.org',
    }
}
