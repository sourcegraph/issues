import { readFile } from 'mz/fs'
import { getFixtureBody } from '../shared/codeHostTestUtils'
import {
    getFileInfoFromSingleFileSourceCodeView,
    getFileInfoWithoutCommitIDsFromMultiFileDiffCodeView,
    isCommitsView,
    isPullRequestView,
} from './scrape'

describe('Bitbucket scrape.ts', () => {
    describe('getFileInfoFromSingleFileSourceCodeView()', () => {
        afterEach(() => {
            document.body.innerHTML = ''
        })
        it('should get the FileInfo for a single file code view', async () => {
            jsdom.reconfigure({
                url: 'https://bitbucket.test/projects/SOUR/repos/mux/browse/context.go',
            })
            document.body.innerHTML = await readFile(`${__dirname}/__fixtures__/single-file.html`, 'utf-8')
            const codeView = document.querySelector<HTMLElement>('.file-content')
            const fileInfo = getFileInfoFromSingleFileSourceCodeView(codeView!)
            expect(fileInfo).toStrictEqual({
                commitID: '212aa90d7cec051ab29930d5c56f758f6f69a789',
                filePath: 'context.go',
                project: 'SOUR',
                rawRepoName: 'bitbucket.test/SOUR/mux',
                repoSlug: 'mux',
                revision: 'master',
            })
        })
    })
    describe('getDiffFileInfoFromMultiFileDiffCodeView()', () => {
        it('should get the FileInfo for an added file', async () => {
            jsdom.reconfigure({
                url: 'https://bitbucket.test/projects/SOURCEGRAPH/repos/mux/pull-requests/1/diff#dir/new_file.go',
            })
            const codeView = await getFixtureBody({
                htmlFixturePath: `${__dirname}/__fixtures__/code-views/pull-request/split/added.html`,
                isFullDocument: false,
            })
            const fileInfo = getFileInfoWithoutCommitIDsFromMultiFileDiffCodeView(codeView)
            expect(fileInfo).toStrictEqual({
                changeType: 'ADD',
                baseFilePath: 'dir/new_file.go',
                filePath: 'dir/new_file.go',
                project: 'SOURCEGRAPH',
                repoSlug: 'mux',
                rawRepoName: 'bitbucket.test/SOURCEGRAPH/mux',
            })
        })
        it('should get the FileInfo for a modified file', async () => {
            jsdom.reconfigure({
                url: 'https://bitbucket.test/projects/SOURCEGRAPH/repos/mux/pull-requests/1/diff#dir/mux.go',
            })
            const codeView = await getFixtureBody({
                htmlFixturePath: `${__dirname}/__fixtures__/code-views/pull-request/split/modified.html`,
                isFullDocument: false,
            })
            const fileInfo = getFileInfoWithoutCommitIDsFromMultiFileDiffCodeView(codeView)
            expect(fileInfo).toStrictEqual({
                changeType: 'MODIFY',
                baseFilePath: 'dir/mux.go',
                filePath: 'dir/mux.go',
                project: 'SOURCEGRAPH',
                repoSlug: 'mux',
                rawRepoName: 'bitbucket.test/SOURCEGRAPH/mux',
            })
        })
        it('should get the FileInfo for a deleted file', async () => {
            jsdom.reconfigure({
                url: 'https://bitbucket.test/projects/SOURCEGRAPH/repos/mux/pull-requests/1/diff#dir/old_test.go',
            })
            const codeView = await getFixtureBody({
                htmlFixturePath: `${__dirname}/__fixtures__/code-views/pull-request/split/deleted.html`,
                isFullDocument: false,
            })
            const fileInfo = getFileInfoWithoutCommitIDsFromMultiFileDiffCodeView(codeView)
            expect(fileInfo).toStrictEqual({
                changeType: 'DELETE',
                baseFilePath: 'dir/old_test.go',
                filePath: 'dir/old_test.go', // TODO should really be undefined?
                project: 'SOURCEGRAPH',
                repoSlug: 'mux',
                rawRepoName: 'bitbucket.test/SOURCEGRAPH/mux',
            })
        })
        it('should get the FileInfo for a copied file', async () => {
            jsdom.reconfigure({
                url: 'https://bitbucket.test/projects/SOURCEGRAPH/repos/mux/pull-requests/1/diff#dir/mux.1.go',
            })
            const codeView = await getFixtureBody({
                htmlFixturePath: `${__dirname}/__fixtures__/code-views/pull-request/split/copied.html`,
                isFullDocument: false,
            })
            const fileInfo = getFileInfoWithoutCommitIDsFromMultiFileDiffCodeView(codeView)
            expect(fileInfo).toStrictEqual({
                changeType: 'COPY',
                baseFilePath: 'dir/mux.go',
                filePath: 'dir/mux.1.go',
                project: 'SOURCEGRAPH',
                repoSlug: 'mux',
                rawRepoName: 'bitbucket.test/SOURCEGRAPH/mux',
            })
        })
        it('should get the FileInfo for a renamed file', async () => {
            jsdom.reconfigure({
                url: 'https://bitbucket.test/projects/SOURCEGRAPH/repos/mux/pull-requests/1/diff#dir/mux_test_moved.go',
            })
            const codeView = await getFixtureBody({
                htmlFixturePath: `${__dirname}/__fixtures__/code-views/pull-request/split/renamed.html`,
                isFullDocument: false,
            })
            const fileInfo = getFileInfoWithoutCommitIDsFromMultiFileDiffCodeView(codeView)
            expect(fileInfo).toStrictEqual({
                changeType: 'RENAME',
                baseFilePath: 'dir/mux_test.go',
                filePath: 'dir/mux_test_moved.go',
                project: 'SOURCEGRAPH',
                repoSlug: 'mux',
                rawRepoName: 'bitbucket.test/SOURCEGRAPH/mux',
            })
        })
        it('should get the FileInfo for a moved file', async () => {
            jsdom.reconfigure({
                url: 'https://bitbucket.test/projects/SOURCEGRAPH/repos/mux/pull-requests/1/diff#dir/test-dir/route.go',
            })
            const codeView = await getFixtureBody({
                htmlFixturePath: `${__dirname}/__fixtures__/code-views/pull-request/split/moved.html`,
                isFullDocument: false,
            })
            const fileInfo = getFileInfoWithoutCommitIDsFromMultiFileDiffCodeView(codeView)
            expect(fileInfo).toStrictEqual({
                changeType: 'MOVE',
                baseFilePath: 'dir/route.go',
                filePath: 'dir/test-dir/route.go',
                project: 'SOURCEGRAPH',
                repoSlug: 'mux',
                rawRepoName: 'bitbucket.test/SOURCEGRAPH/mux',
            })
        })
    })

    describe('isCommitView()', () => {
        it('detects a commit view when there is no context path', () => {
            expect(
                isCommitsView(
                    new URL(
                        'https://bitbucket.sgdev.org/projects/SOUR/repos/vegeta/commits/e827e02858e8d5d581bac4d57b31fbd275da39c5'
                    )
                )
            ).toBe(true)
        })

        it('detects a commit view when there is a context path', () => {
            expect(
                isCommitsView(
                    new URL(
                        'https://atlassian.company.org/bitbucket/projects/SOUR/repos/mux/commits/8eaa9f13091105874ef3e20c65922e382cef3c64'
                    )
                )
            ).toBe(true)
        })
    })

    describe('isPullRequestView()', () => {
        it('detects a pull request view when there is no context path', () => {
            expect(
                isPullRequestView(
                    new URL('https://bitbucket.sgdev.org/projects/SOUR/repos/mux/pull-requests/1/overview')
                )
            ).toBe(true)
        })

        it('detects a pull request view when there is a context path', () => {
            expect(
                isPullRequestView(
                    new URL('https://atlassian.company.org/bitbucket/projects/SOUR/repos/mux/pull-requests/1/overview')
                )
            ).toBe(true)
        })
    })
})
