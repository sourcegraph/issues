import React from 'react'
import renderer from 'react-test-renderer'

import { UserAvatar } from './UserAvatar'

describe('UserAvatar', () => {
    test('no avatar URL', () =>
        expect(renderer.create(<UserAvatar user={{ avatarURL: null, username: 'test' }} />).toJSON()).toMatchSnapshot())

    test('with avatar URL', () =>
        expect(renderer.create(<UserAvatar user={{ avatarURL: 'u', username: 'test' }} />).toJSON()).toMatchSnapshot())
})
