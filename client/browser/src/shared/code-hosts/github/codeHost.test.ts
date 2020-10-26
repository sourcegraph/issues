import { existsSync, readdirSync } from 'fs'
import { startCase } from 'lodash'
import { testCodeHostMountGetters, testToolbarMountGetter } from '../shared/codeHostTestUtils'
import { CodeView } from '../shared/codeViews'
import {
    createFileActionsToolbarMount,
    createFileLineContainerToolbarMount,
    githubCodeHost,
    checkIsGitHubDotCom,
} from './codeHost'
import { readFile } from 'mz/fs'

const testCodeHost = (fixturePath: string): void => {
    if (existsSync(fixturePath)) {
        describe('githubCodeHost', () => {
            testCodeHostMountGetters(githubCodeHost, fixturePath)
        })
    }
}

const testMountGetter = (
    mountGetter: NonNullable<CodeView['getToolbarMount']>,
    mountGetterName: string,
    codeViewFixturePath: string
): void => {
    if (existsSync(codeViewFixturePath)) {
        describe(mountGetterName, () => {
            testToolbarMountGetter(codeViewFixturePath, mountGetter)
        })
    }
}

describe('github/codeHost', () => {
    for (const version of ['github.com', 'ghe-2.14.11']) {
        describe(version, () => {
            for (const page of readdirSync(`${__dirname}/__fixtures__/${version}`)) {
                describe(`${startCase(page)} page`, () => {
                    for (const extension of ['vanilla', 'refined-github']) {
                        describe(startCase(extension), () => {
                            // no split/unified view on blobs, and pull-request-discussion is always unified
                            if (page === 'blob' || page === 'pull-request-discussion') {
                                const directory = `${__dirname}/__fixtures__/${version}/${page}/${extension}`
                                testCodeHost(`${directory}/page.html`)
                                if (page !== 'pull-request-discussion') {
                                    testMountGetter(
                                        createFileLineContainerToolbarMount,
                                        'createSingleFileToolbarMount()',
                                        `${directory}/code-view.html`
                                    )
                                }
                            } else {
                                for (const view of ['split', 'unified']) {
                                    describe(`${startCase(view)} view`, () => {
                                        const directory = `${__dirname}/__fixtures__/${version}/${page}/${extension}/${view}`
                                        testCodeHost(`${directory}/page.html`)
                                        describe('createFileActionsToolbarMount()', () => {
                                            testMountGetter(
                                                createFileActionsToolbarMount,
                                                'createFileActionsToolbarMount()',
                                                `${directory}/code-view.html`
                                            )
                                        })
                                    })
                                }
                            }
                        })
                    }
                })
            }
        })
    }

    describe('githubCodeHost.urlToFile()', () => {
        const urlToFile = githubCodeHost.urlToFile!
        const sourcegraphURL = 'https://sourcegraph.my.org'

        describe('on blob page', () => {
            beforeAll(() => {
                jsdom.reconfigure({
                    url:
                        'https://github.com/sourcegraph/sourcegraph/blob/master/browser/src/shared/code-hosts/code_intelligence.tsx',
                })
            })
            it('returns an URL to the Sourcegraph instance if the location has a viewState', () => {
                expect(
                    urlToFile(
                        sourcegraphURL,
                        {
                            repoName: 'sourcegraph/sourcegraph',
                            rawRepoName: 'github.com/sourcegraph/sourcegraph',
                            revision: 'master',
                            filePath: 'browser/src/shared/code-hosts/code_intelligence.tsx',
                            position: {
                                line: 5,
                                character: 12,
                            },
                            viewState: 'references',
                        },
                        { part: undefined }
                    )
                ).toBe(
                    'https://sourcegraph.my.org/sourcegraph/sourcegraph@master/-/blob/browser/src/shared/code-hosts/code_intelligence.tsx#L5:12&tab=references'
                )
            })

            it('returns an absolute URL if the location is not on the same code host', () => {
                expect(
                    urlToFile(
                        sourcegraphURL,
                        {
                            repoName: 'sourcegraph/sourcegraph',
                            rawRepoName: 'ghe.sgdev.org/sourcegraph/sourcegraph',
                            revision: 'master',
                            filePath: 'browser/src/shared/code-hosts/code_intelligence.tsx',
                            position: {
                                line: 5,
                                character: 12,
                            },
                        },
                        { part: undefined }
                    )
                ).toBe(
                    'https://sourcegraph.my.org/sourcegraph/sourcegraph@master/-/blob/browser/src/shared/code-hosts/code_intelligence.tsx#L5:12'
                )
            })
            it('returns an URL to a blob on the same code host if possible', () => {
                expect(
                    urlToFile(
                        sourcegraphURL,
                        {
                            repoName: 'sourcegraph/sourcegraph',
                            rawRepoName: 'github.com/sourcegraph/sourcegraph',
                            revision: 'master',
                            filePath: 'browser/src/shared/code-hosts/code_intelligence.tsx',
                            position: {
                                line: 5,
                                character: 12,
                            },
                        },
                        { part: undefined }
                    )
                ).toBe(
                    'https://github.com/sourcegraph/sourcegraph/blob/master/browser/src/shared/code-hosts/code_intelligence.tsx#L5:12'
                )
            })
        })
        describe('on pull request page', () => {
            beforeAll(async () => {
                jsdom.reconfigure({ url: 'https://github.com/sourcegraph/sourcegraph/pull/3257/files' })
                document.documentElement.innerHTML = await readFile(
                    __dirname + '/__fixtures__/github.com/pull-request/vanilla/unified/page.html',
                    'utf-8'
                )
            })
            it('returns a URL to the same PR if possible', () => {
                expect(
                    urlToFile(
                        sourcegraphURL,
                        {
                            repoName: 'sourcegraph/sourcegraph',
                            rawRepoName: 'github.com/sourcegraph/sourcegraph',
                            revision: 'core/gitserver-tracing',
                            filePath: 'cmd/gitserver/server/server.go',
                            position: {
                                line: 1335,
                                character: 6,
                            },
                        },
                        { part: 'head' }
                    )
                ).toBe(
                    'https://github.com/sourcegraph/sourcegraph/pull/3257/files#diff-93ceb95cf0be7b7b17815c5638fc4c5cR1335'
                )
            })
        })
    })

    describe('githubCodeHost.checkIsGithubDotCom()', () => {
        it('returns true with a github.com URL', () => {
            expect(checkIsGitHubDotCom('https://www.github.com')).toBe(true)
            expect(checkIsGitHubDotCom('https://github.com')).toBe(true)
            expect(checkIsGitHubDotCom('http://github.com')).toBe(true)
            expect(checkIsGitHubDotCom('http://www.github.com')).toBe(true)
        })

        it('returns false on domains that impersonate github.com', () => {
            expect(checkIsGitHubDotCom('https://wwwwgithub.com')).toBe(false)
            expect(checkIsGitHubDotCom('https://www.githubccom')).toBe(false)
            expect(checkIsGitHubDotCom('http://githubccom')).toBe(false)
        })
    })
})
