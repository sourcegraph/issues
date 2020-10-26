import {
    isRepoSeeOtherErrorLike,
    RepoSeeOtherError,
    isPrivateRepoPublicSourcegraphComErrorLike,
    PrivateRepoPublicSourcegraphComError,
    RepoNotFoundError,
    isRevisionNotFoundErrorLike,
    RevisionNotFoundError,
    CloneInProgressError,
    isCloneInProgressErrorLike,
    isRepoNotFoundErrorLike,
} from './errors'

describe('backend errors', () => {
    describe('isCloneInProgressErrorLike()', () => {
        it('returns true for CloneInProgressError', () => {
            expect(isCloneInProgressErrorLike(new CloneInProgressError('foobar'))).toBe(true)
        })
    })
    describe('isRevisionNotFoundErrorLike()', () => {
        it('returns true for RevisionNotFoundError', () => {
            expect(isRevisionNotFoundErrorLike(new RevisionNotFoundError('foobar'))).toBe(true)
        })
    })
    describe('isRepoNotFoundErrorLike()', () => {
        it('returns true for RepoNotFoundError', () => {
            expect(isRepoNotFoundErrorLike(new RepoNotFoundError('foobar'))).toBe(true)
        })
    })
    describe('isRepoSeeOtherErrorLike()', () => {
        it('returns the redirect URL for RepoSeeOtherErrors', () => {
            expect(isRepoSeeOtherErrorLike(new RepoSeeOtherError('https://sourcegraph.test'))).toBe(
                'https://sourcegraph.test'
            )
        })
        it('returns the redirect URL for plain RepoSeeOtherErrors', () => {
            expect(
                isRepoSeeOtherErrorLike({ message: new RepoSeeOtherError('https://sourcegraph.test').message })
            ).toBe('https://sourcegraph.test')
        })
        it('returns false for other errors', () => {
            expect(isRepoSeeOtherErrorLike(new Error())).toBe(false)
        })
        it('returns false for other values', () => {
            expect(isRepoSeeOtherErrorLike('foo')).toBe(false)
        })
    })
    describe('isPrivateRepoPublicSourcegraphComErrorLike()', () => {
        it('returns true for PrivateRepoPublicSourcegraphComError', () => {
            expect(
                isPrivateRepoPublicSourcegraphComErrorLike(new PrivateRepoPublicSourcegraphComError('ResolveFoo'))
            ).toBe(true)
        })
        it('returns true for plain PrivateRepoPublicSourcegraphComError', () => {
            expect(
                isPrivateRepoPublicSourcegraphComErrorLike({
                    message: new PrivateRepoPublicSourcegraphComError('ResolveFoo').message,
                })
            ).toBe(true)
        })
    })
})
