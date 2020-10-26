import { testCodeHostMountGetters, testToolbarMountGetter } from '../shared/codeHostTestUtils'
import { bitbucketServerCodeHost, getToolbarMount } from './codeHost'

describe('bitbucketServerCodeHost', () => {
    testCodeHostMountGetters(bitbucketServerCodeHost, {
        getCommandPaletteMount: `${__dirname}/__fixtures__/browse.html`,
        getViewContextOnSourcegraphMount: `${__dirname}/__fixtures__/browse.html`,
    })
    describe('getToolbarMount()', () => {
        testToolbarMountGetter(`${__dirname}/__fixtures__/code-views/pull-request/split/modified.html`, getToolbarMount)
    })
})
