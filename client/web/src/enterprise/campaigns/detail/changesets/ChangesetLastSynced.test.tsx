import React from 'react'
import { ChangesetLastSynced } from './ChangesetLastSynced'
import { subMinutes } from 'date-fns'
import { mount } from 'enzyme'

describe('ChangesetLastSynced', () => {
    for (const viewerCanAdminister of [false, true]) {
        describe(`ViewerCanAdminister: ${viewerCanAdminister}`, () => {
            test('renders not scheduled', () => {
                expect(
                    mount(
                        <ChangesetLastSynced
                            changeset={{
                                id: '123',
                                nextSyncAt: null,
                                updatedAt: subMinutes(new Date('2020-03-01'), 10).toISOString(),
                            }}
                            viewerCanAdminister={viewerCanAdminister}
                            _now={new Date('2020-03-01')}
                        />
                    )
                ).toMatchSnapshot()
            })
            test('renders scheduled', () => {
                expect(
                    mount(
                        <ChangesetLastSynced
                            changeset={{
                                id: '123',
                                nextSyncAt: new Date('2020-03-02').toISOString(),
                                updatedAt: subMinutes(new Date('2020-03-01'), 10).toISOString(),
                            }}
                            viewerCanAdminister={viewerCanAdminister}
                            _now={new Date('2020-03-01')}
                        />
                    )
                ).toMatchSnapshot()
            })
        })
    }
})
