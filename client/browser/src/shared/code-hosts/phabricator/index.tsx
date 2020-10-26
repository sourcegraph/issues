import { FileSpec, RawRepoSpec, ResolvedRevisionSpec } from '../../../../../shared/src/util/url'

export enum PhabricatorMode {
    Diffusion = 1,
    Differential,
    Revision,
    Change,
}

export interface DiffusionState extends RawRepoSpec, ResolvedRevisionSpec, FileSpec {
    mode: PhabricatorMode.Diffusion
}

export interface RevisionSpec {
    /**
     * The ID of the revision in Differential.
     * A revision is a set of changes up for review in Differential.
     * See https://secure.phabricator.com/book/phabricator/article/differential/#how-review-works
     */
    revisionID: number
}

export interface DiffSpec {
    /**
     * The ID of the 'head' diff that is being viewed in the Differential UI.
     * A Differential revision is made up of one or more 'Diffs' (patches).
     */
    diffID: number
}

export interface BaseDiffSpec {
    /**
     * The ID of the 'base' diff. This is only defined when comparing
     * two states of a revision in the differential UI.
     *
     */
    baseDiffID?: number
}

export interface DifferentialState extends RevisionSpec, DiffSpec, BaseDiffSpec {
    mode: PhabricatorMode.Differential
    baseRawRepoName: string
    headRawRepoName: string
}

export interface RevisionState extends RawRepoSpec {
    mode: PhabricatorMode.Revision
    baseCommitID: string
    headCommitID: string
}

/**
 * Refers to a URL like http://phabricator.aws.sgdev.org/source/nzap/change/master/checked_message_bench_test.go,
 * which a user gets to by clicking "Show Last Change" on a differential page.
 */
export interface ChangeState extends RawRepoSpec, FileSpec, ResolvedRevisionSpec {
    mode: PhabricatorMode.Change
}

export function convertSpacesToTabs(realLineContent: string, domContent: string): boolean {
    return !!realLineContent && !!domContent && realLineContent.startsWith('\t') && !domContent.startsWith('\t')
}

export function spacesToTabsAdjustment(text: string): number {
    let suffix = text
    let adjustment = 0

    while (suffix.length >= 2 && suffix.startsWith('  ')) {
        ++adjustment
        suffix = suffix.slice(2)
    }
    return adjustment
}
