import { testToolbarMountGetter } from '../shared/codeHostTestUtils'
import { commitCodeView, diffCodeView } from './codeHost'

describe('phabricator/codeHost', () => {
    describe('diffCodeView', () => {
        describe('getToolbarMount()', () => {
            for (const view of ['split', 'unified']) {
                testToolbarMountGetter(
                    `${__dirname}/__fixtures__/code-views/2017.09-r1/differential/${view}.html`,
                    diffCodeView.getToolbarMount
                )
            }
        })
    })
    describe('commitCodeView', () => {
        describe('getToolbarMount()', () => {
            for (const view of ['split', 'unified']) {
                testToolbarMountGetter(
                    `${__dirname}/__fixtures__/code-views/2017.09-r1/commit/${view}.html`,
                    commitCodeView.getToolbarMount
                )
            }
        })
    })
    // TODO sourceCodeView, currently not possible because code view element does not contain toolbar
})
