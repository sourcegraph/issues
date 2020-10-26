import React from 'react'
import renderer from 'react-test-renderer'
import { Notices } from './Notices'
import * as H from 'history'

describe('Notices', () => {
    test('shows notices for location', () =>
        expect(
            renderer
                .create(
                    <Notices
                        location="home"
                        settingsCascade={{
                            subjects: [],
                            final: {
                                notices: [
                                    { message: 'a', location: 'home' },
                                    { message: 'a', location: 'home', dismissible: true },
                                    { message: 'b', location: 'top' },
                                ],
                            },
                        }}
                        history={H.createMemoryHistory()}
                    />
                )
                .toJSON()
        ).toMatchSnapshot())

    test('no notices', () =>
        expect(
            renderer
                .create(
                    <Notices
                        location="home"
                        history={H.createMemoryHistory()}
                        settingsCascade={{ subjects: [], final: { notices: null } }}
                    />
                )
                .toJSON()
        ).toMatchSnapshot())
})
