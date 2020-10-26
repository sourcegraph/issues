import { startCase } from 'lodash'
import { DOMFunctionsTest, testDOMFunctions } from '../shared/codeHostTestUtils'
import { diffDomFunctions, diffusionDOMFns } from './domFunctions'

type PhabricatorPage = 'commit' | 'differential'

type PhabricatorVersion = '2017.09-r1' | '2019.21.0-r25'

interface PhabricatorCodeViewFixture extends Pick<DOMFunctionsTest, 'lineCases'> {}

describe('Phabricator DOM functions', () => {
    describe('diffDOMFunctions', () => {
        const DIFF_FIXTURES: Record<
            PhabricatorVersion,
            Partial<Record<PhabricatorPage, (view: 'split' | 'unified') => PhabricatorCodeViewFixture>>
        > = {
            '2017.09-r1': {
                commit: () => ({
                    lineCases: [
                        { diffPart: 'head', lineNumber: 3 }, // not changed
                        { diffPart: 'head', lineNumber: 7 }, // added
                        { diffPart: 'base', lineNumber: 10 }, // removed
                    ],
                }),
                differential: () => ({
                    lineCases: [
                        { diffPart: 'head', lineNumber: 9 }, // not changed
                        { diffPart: 'head', lineNumber: 10 }, // added
                        // TODO test case for removed line
                    ],
                }),
            },
            '2019.21.0-r25': {
                differential: view => ({
                    lineCases: [
                        { diffPart: 'head', lineNumber: 29 }, // not changed
                        { diffPart: 'head', lineNumber: 64, firstCharacterIsDiffIndicator: view === 'unified' }, // added
                        { diffPart: 'base', lineNumber: 34, firstCharacterIsDiffIndicator: view === 'unified' }, // removed
                    ],
                }),
            },
        }
        for (const [version, testCases] of Object.entries(DIFF_FIXTURES)) {
            for (const [page, testCase] of Object.entries(testCases)) {
                if (!testCase) {
                    continue
                }
                describe(`${startCase(page)} Page`, () => {
                    for (const view of ['split', 'unified'] as const) {
                        const htmlFixturePath = `${__dirname}/__fixtures__/code-views/${version}/${page}/${view}.html`
                        describe(`${startCase(view)} view, version ${version}`, () => {
                            // https://phabricator.sgdev.org/D3#diff-7ddfb3e0
                            testDOMFunctions(diffDomFunctions, {
                                htmlFixturePath,
                                ...testCase(view),
                            })
                        })
                    }
                })
            }
        }
    })

    describe('diffusionDOMFns', () => {
        const htmlFixturePath = `${__dirname}/__fixtures__/code-views/2017.09-r1/diffusion.html`
        // https://phabricator.sgdev.org/source/test/browse/master/main.go;48600480cce9f832f7daacab256fbfdeb7002603
        testDOMFunctions(diffusionDOMFns, {
            htmlFixturePath,
            lineCases: [{ lineNumber: 1 }, { lineNumber: 10 }],
        })
    })
})
