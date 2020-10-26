import { startCase } from 'lodash'
import { Omit } from 'utility-types'
import { DOMFunctionsTest, getFixtureBody, testDOMFunctions } from '../shared/codeHostTestUtils'
import { diffDomFunctions, isDomSplitDiff, singleFileDOMFunctions } from './domFunctions'

type GitHubVersion = 'github.com' | 'ghe-2.14.11'

describe('GitHub DOM functions', () => {
    describe('diffDomFunctions', () => {
        type GitHubDiffPage = 'pull-request' | 'pull-request-discussion' | 'commit'

        interface GitHubCodeViewFixture extends Omit<DOMFunctionsTest, 'htmlFixturePath'> {}

        const diffFixtures: Record<GitHubVersion, Record<GitHubDiffPage, GitHubCodeViewFixture>> = {
            'ghe-2.14.11': {
                commit: {
                    url: 'https://ghe.sgdev.org/beyang/mux/commit/1fddf523893b7475951631ed0f7e09edd9ce50d0',
                    lineCases: [
                        { diffPart: 'head', lineNumber: 80, firstCharacterIsDiffIndicator: true }, // not changed
                        { diffPart: 'head', lineNumber: 82, firstCharacterIsDiffIndicator: true }, // added
                        { diffPart: 'base', lineNumber: 82, firstCharacterIsDiffIndicator: true }, // removed
                    ],
                },
                'pull-request': {
                    url: 'http://ghe.sgdev.org/beyang/mux/pull/1',
                    lineCases: [
                        { diffPart: 'head', lineNumber: 63, firstCharacterIsDiffIndicator: true }, // not changed
                        { diffPart: 'head', lineNumber: 64, firstCharacterIsDiffIndicator: true }, // added
                    ],
                },
                'pull-request-discussion': {
                    url: 'http://ghe.sgdev.org/beyang/mux/pull/1',
                    lineCases: [
                        { diffPart: 'head', lineNumber: 64, firstCharacterIsDiffIndicator: true }, // added
                    ],
                },
            },
            'github.com': {
                commit: {
                    url: 'https://github.com/sourcegraph/sourcegraph/commit/d3d0fe7fad2c909e3a2e4de2259dc6604983a092',
                    lineCases: [
                        { diffPart: 'head', lineNumber: 41 }, // not changed
                        { diffPart: 'base', lineNumber: 42 }, // removeed
                        { diffPart: 'head', lineNumber: 42 }, // added
                    ],
                },
                'pull-request': {
                    url: 'https://github.com/sourcegraph/sourcegraph/pull/3272/files',
                    lineCases: [
                        { diffPart: 'head', lineNumber: 570 }, // not changed
                        { diffPart: 'base', lineNumber: 572 }, // removed
                        { diffPart: 'head', lineNumber: 572 }, // added
                    ],
                },
                'pull-request-discussion': {
                    url: 'https://github.com/sourcegraph/sourcegraph/pull/3221',
                    lineCases: [
                        { diffPart: 'head', lineNumber: 13 }, // added
                    ],
                },
            },
        }
        for (const [version, pages] of Object.entries(diffFixtures)) {
            describe(version, () => {
                for (const [page, { lineCases, url }] of Object.entries(pages)) {
                    describe(`${startCase(page)} page`, () => {
                        for (const extension of ['vanilla', 'refined-github']) {
                            describe(startCase(extension), () => {
                                if (page === 'pull-request-discussion') {
                                    const htmlFixturePath = `${__dirname}/__fixtures__/${version}/${page}/${extension}/code-view.html`
                                    testDOMFunctions(diffDomFunctions, {
                                        url,
                                        htmlFixturePath,
                                        lineCases,
                                    })
                                } else {
                                    for (const view of ['split', 'unified']) {
                                        const htmlFixturePath = `${__dirname}/__fixtures__/${version}/${page}/${extension}/${view}/code-view.html`
                                        describe(`${startCase(view)} view`, () => {
                                            testDOMFunctions(diffDomFunctions, {
                                                url,
                                                htmlFixturePath,
                                                lineCases,
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

    describe('singleFileDOMFunctions', () => {
        for (const version of ['github.com', 'ghe-2.14.11']) {
            describe(version, () => {
                for (const extension of ['vanilla', 'refined-github']) {
                    describe(startCase(extension), () => {
                        const htmlFixturePath = `${__dirname}/__fixtures__/${version}/blob/${extension}/code-view.html`
                        testDOMFunctions(singleFileDOMFunctions, {
                            htmlFixturePath,
                            lineCases: [{ lineNumber: 1 }, { lineNumber: 2 }],
                        })
                    })
                }
            })
        }
    })

    describe('isDomSplitDiff()', () => {
        for (const version of ['github.com', 'ghe-2.14.11']) {
            describe(`Version ${version}`, () => {
                const views = [
                    {
                        view: 'pull-request',
                        url: 'https://github.com/sourcegraph/sourcegraph/pull/2672/files',
                    },
                    {
                        view: 'commit',
                        url:
                            'https://github.com/sourcegraph/sourcegraph/commit/2c74f329fd03008fa0b446cd5e53234715dae3dc',
                    },
                    {
                        view: 'pull-request-discussion',
                        url: 'https://github.com/sourcegraph/sourcegraph/pull/2672/',
                    },
                ]
                for (const { view, url } of views) {
                    describe(`${startCase(view)} page`, () => {
                        beforeEach(() => {
                            // TODO ideally DOM functions would not look at global state like the URL.
                            jsdom.reconfigure({ url })
                        })
                        for (const extension of ['vanilla', 'refined-github']) {
                            describe(startCase(extension), () => {
                                if (view === 'pull-request-discussion') {
                                    it('should return false', async () => {
                                        const codeViewElement = await getFixtureBody({
                                            htmlFixturePath: `${__dirname}/__fixtures__/${version}/${view}/${extension}/code-view.html`,
                                            isFullDocument: false,
                                        })
                                        expect(isDomSplitDiff(codeViewElement)).toBe(false)
                                    })
                                } else {
                                    it('should return true for split view', async () => {
                                        const codeViewElement = await getFixtureBody({
                                            htmlFixturePath: `${__dirname}/__fixtures__/${version}/${view}/${extension}/split/code-view.html`,
                                            isFullDocument: false,
                                        })
                                        expect(isDomSplitDiff(codeViewElement)).toBe(true)
                                    })
                                    it('should return false for unified view', async () => {
                                        const codeViewElement = await getFixtureBody({
                                            htmlFixturePath: `${__dirname}/__fixtures__/${version}/${view}/${extension}/unified/code-view.html`,
                                            isFullDocument: false,
                                        })
                                        expect(isDomSplitDiff(codeViewElement)).toBe(false)
                                    })
                                }
                            })
                        }
                    })
                }
            })
        }
    })
})
