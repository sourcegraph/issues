import React from 'react'
import { mount } from 'enzyme'
import { UserProfileFormFields } from './UserProfileFormFields'

describe('UserProfileFormFields', () => {
    test('simple', () =>
        expect(
            mount(
                <UserProfileFormFields
                    value={{ username: 'u', displayName: 'd', avatarURL: 'https://example.com/image.jpg' }}
                    onChange={() => {}}
                />
            )
        ).toMatchSnapshot())
})
