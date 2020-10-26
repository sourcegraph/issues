import React from 'react'
import { MemoryRouter } from 'react-router'
import renderer from 'react-test-renderer'
import { PersonLink } from './PersonLink'

describe('PersonLink', () => {
    test('no user account', () =>
        expect(
            renderer
                .create(
                    <PersonLink
                        person={{ displayName: 'alice', email: 'alice@example.com', user: null }}
                        className="a"
                        userClassName="b"
                    />
                )
                .toJSON()
        ).toMatchSnapshot())

    test('with user account', () =>
        expect(
            renderer
                .create(
                    <MemoryRouter>
                        <PersonLink
                            person={{
                                displayName: 'Alice',
                                email: 'alice@example.com',
                                user: { username: 'alice', displayName: 'Alice Smith', url: 'u' },
                            }}
                            className="a"
                            userClassName="b"
                        />
                    </MemoryRouter>
                )
                .toJSON()
        ).toMatchSnapshot())
})
