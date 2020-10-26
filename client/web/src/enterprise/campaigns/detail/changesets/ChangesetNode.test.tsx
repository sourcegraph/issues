import * as H from 'history'
import React from 'react'
import { ChangesetNode } from './ChangesetNode'
import { shallow } from 'enzyme'

describe('ChangesetNode', () => {
    const history = H.createMemoryHistory({ keyLength: 0 })
    const location = H.createLocation('/campaigns')
    test('renders ExternalChangeset', () => {
        expect(
            shallow(
                <ChangesetNode
                    isLightTheme={true}
                    history={history}
                    location={location}
                    viewerCanAdminister={false}
                    node={{ __typename: 'ExternalChangeset' } as any}
                />
            )
        ).toMatchSnapshot()
    })
    test('renders HiddenExternalChangeset', () => {
        expect(
            shallow(
                <ChangesetNode
                    isLightTheme={true}
                    history={history}
                    location={location}
                    viewerCanAdminister={false}
                    node={{ __typename: 'HiddenExternalChangeset' } as any}
                />
            )
        ).toMatchSnapshot()
    })
})
