import { startCase } from 'lodash'
import { testDOMFunctions } from '../shared/codeHostTestUtils'
import { diffDOMFunctions, singleFileDOMFunctions } from './domFunctions'

describe('Bitbucket DOM functions', () => {
    describe('diffDOMFunctions', () => {
        for (const view of ['split', 'unified']) {
            describe(`${startCase(view)} view`, () => {
                // https://gitlab.com/gitlab-org/gitlab-runner/merge_requests/1058/diffs?view=parallel#diff-content-ca8e0332ce17b2ee630a2ee2c0b56d47a462dadf
                testDOMFunctions(diffDOMFunctions, {
                    htmlFixturePath: `${__dirname}/__fixtures__/code-views/merge-request/${view}.html`,
                    lineCases: [
                        { diffPart: 'head', lineNumber: 733 }, // not changed
                        { diffPart: 'head', lineNumber: 740 }, // added
                        { diffPart: 'base', lineNumber: 735 }, // removed
                    ],
                })
            })
        }
    })

    describe('singleFileDOMFunctions', () => {
        const htmlFixturePath = `${__dirname}/__fixtures__/code-views/blob.html`
        // https://gitlab.com/gitlab-org/gitlab-runner/blob/0362425dc5026417338ac6a823c53fe65b10c4a7/executors/shell/executor_shell.go
        testDOMFunctions(singleFileDOMFunctions, {
            htmlFixturePath,
            lineCases: [{ lineNumber: 1 }, { lineNumber: 22 }],
        })
    })
})
