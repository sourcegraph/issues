import { startCase } from 'lodash'
import { testDOMFunctions } from '../shared/codeHostTestUtils'
import { diffDOMFunctions, singleFileDOMFunctions } from './domFunctions'

describe('Bitbucket DOM functions', () => {
    describe('diffDOMFunctions', () => {
        for (const view of ['split', 'unified']) {
            describe(`${startCase(view)} view`, () => {
                testDOMFunctions(diffDOMFunctions, {
                    htmlFixturePath: `${__dirname}/__fixtures__/code-views/pull-request/${view}/modified.html`,
                    lineCases: [
                        { diffPart: 'head', lineNumber: 54 }, // not changed
                        { diffPart: 'head', lineNumber: 60 }, // added
                        { diffPart: 'base', lineNumber: 102 }, // removed
                    ],
                })
            })
        }
    })

    describe('singleFileDOMFunctions', () => {
        const htmlFixturePath = `${__dirname}/__fixtures__/code-views/single-file-source.html`
        testDOMFunctions(singleFileDOMFunctions, {
            htmlFixturePath,
            lineCases: [{ lineNumber: 1 }, { lineNumber: 18 }],
        })
    })
})
