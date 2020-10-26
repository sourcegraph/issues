import * as H from 'history'
import React from 'react'
import { mount } from 'enzyme'
import { VersionContextDropdown, VersionContextDropdownProps } from './VersionContextDropdown'
import sinon from 'sinon'
import { SearchPatternType } from '../graphql-operations'

const commonProps: VersionContextDropdownProps = {
    setVersionContext: sinon.spy((_versionContext: string | undefined) => {}),
    availableVersionContexts: [
        { name: '3.0', description: '3.0', revisions: [{ repo: 'github.com/sourcegraph/sourcegraph', rev: '3.0' }] },
        { name: '3.15', description: '3.15', revisions: [{ repo: 'github.com/sourcegraph/sourcegraph', rev: '3.15' }] },
    ],
    history: H.createMemoryHistory({ keyLength: 0 }),
    navbarSearchQuery: 'test',
    patternType: SearchPatternType.literal,
    caseSensitive: false,
    versionContext: undefined,
    filtersInQuery: undefined,
}
describe('VersionContextDropdown', () => {
    it('renders the version context dropdown with no context selected', () => {
        expect(mount(<VersionContextDropdown {...commonProps} />)).toMatchSnapshot()
    })

    it('renders the expanded version context dropdown, with 3.15 selected and displayed first', () => {
        expect(
            mount(
                <VersionContextDropdown {...commonProps} versionContext="3.15" alwaysExpanded={true} portal={false} />
            )
        ).toMatchSnapshot()
    })
})
