import { startCase } from 'lodash'
import { parseURL, getDiffFileName } from './util'
import { getFixtureBody } from '../shared/codeHostTestUtils'

describe('util', () => {
    describe('parseURL()', () => {
        const testcases: {
            name: string
            url: string
        }[] = [
            {
                name: 'tree page',
                url: 'https://github.com/sourcegraph/sourcegraph/tree/master/client',
            },
            {
                name: 'blob page',
                url: 'https://github.com/sourcegraph/sourcegraph/blob/3.3/shared/src/hover/HoverOverlay.tsx',
            },
            {
                name: 'commit page',
                url: 'https://github.com/sourcegraph/sourcegraph/commit/fb054666c12b40f180c794db4829cbfd1a5fabae',
            },
            {
                name: 'pull request page',
                url: 'https://github.com/sourcegraph/sourcegraph/pull/3849',
            },
            {
                name: 'compare page',
                url:
                    'https://github.com/sourcegraph/sourcegraph-basic-code-intel/compare/new-extension-api-usage...fuzzy-locations',
            },
            {
                name: 'selections - single line',
                url: 'https://github.com/sourcegraph/sourcegraph/blob/master/jest.config.base.js#L5',
            },
            {
                name: 'selections - range',
                url: 'https://github.com/sourcegraph/sourcegraph/blob/master/jest.config.base.js#L5-L12',
            },
            {
                name: 'snippet permalink',
                url:
                    'https://github.com/sourcegraph/sourcegraph/blob/6a91ccec97a46bfb511b7ff58d790554a7d075c8/client/browser/src/shared/repo/backend.tsx#L128-L151',
            },
            {
                name: 'pull request list',
                url: 'https://github.com/sourcegraph/sourcegraph/pulls',
            },
            {
                name: 'wiki page',
                url: 'https://github.com/sourcegraph/sourcegraph/pulls',
            },
            {
                name: 'branch name with forward slashes',
                url: 'http://ghe.sgdev.org/beyang/mux/blob/jr/branch/mux.go',
            },
        ]
        for (const { name, url } of testcases) {
            test(name, () => {
                expect(parseURL(new URL(url))).toMatchSnapshot()
            })
        }
    })
})

type GithubVersion = 'github.com' | 'ghe-2.14.11'
type GithubDiffPage = 'commit' | 'pull-request' | 'pull-request-discussion'

describe('getDiffFileName()', () => {
    const tests: Record<GithubVersion, Record<GithubDiffPage, string>> = {
        'github.com': {
            commit: 'doc/dev/incidents.md',
            'pull-request-discussion': 'web/src/regression/util/TestResourceManager.ts',
            'pull-request': 'packages/sourcegraph-extension-api/src/sourcegraph.d.ts',
        },
        'ghe-2.14.11': {
            commit: 'mux.go',
            'pull-request': 'mux.go',
            'pull-request-discussion': 'mux.go',
        },
    }
    const testGetDeltaFilename = ({
        expectedFilePath,
        htmlFixturePath,
    }: {
        expectedFilePath: string
        htmlFixturePath: string
    }) => {
        test('extracts the filename', async () => {
            const container = await getFixtureBody({
                htmlFixturePath,
                isFullDocument: false,
            })
            // TODO add examples with renamed files and check that getDeltaFilename() doesn't return
            // identical headFilePath & baseFilePath
            expect(getDiffFileName(container)).toStrictEqual({
                baseFilePath: expectedFilePath,
                headFilePath: expectedFilePath,
            })
        })
    }
    for (const gitHubVersion of ['github.com', 'ghe-2.14.11'] as const) {
        describe(gitHubVersion, () => {
            for (const [gitHubDiffPage, expectedFilePath] of Object.entries(tests[gitHubVersion])) {
                describe(`${startCase(gitHubDiffPage)} page`, () => {
                    for (const gitHubFlavor of ['vanilla', 'refined-github']) {
                        describe(startCase(gitHubFlavor), () => {
                            if (gitHubDiffPage === 'pull-request-discussion') {
                                testGetDeltaFilename({
                                    expectedFilePath,
                                    htmlFixturePath: `${__dirname}/__fixtures__/${gitHubVersion}/${gitHubDiffPage}/${gitHubFlavor}/code-view.html`,
                                })
                            } else {
                                for (const view of ['split', 'unified']) {
                                    describe(`${view}`, () => {
                                        testGetDeltaFilename({
                                            expectedFilePath,
                                            htmlFixturePath: `${__dirname}/__fixtures__/${gitHubVersion}/${gitHubDiffPage}/${gitHubFlavor}/${view}/code-view.html`,
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
})
