import { RepoSpec, RevisionSpec } from '../../../../shared/src/util/url'

export interface DiffResolvedRevisionSpec {
    baseCommitID: string
    headCommitID: string
}

export interface OpenInSourcegraphProps extends RepoSpec, RevisionSpec {
    sourcegraphURL: string
    filePath: string
}

export interface OpenDiffInSourcegraphProps
    extends Pick<OpenInSourcegraphProps, Exclude<keyof OpenInSourcegraphProps, 'commit'>> {
    commit: {
        baseRev: string
        headRev: string
    }
}
