import React from 'react'
import renderer from 'react-test-renderer'
import { Timestamp } from './Timestamp'

describe('Timestamp', () => {
    test('mocked current time', () =>
        expect(renderer.create(<Timestamp date="2006-01-02" />).toJSON()).toMatchSnapshot())

    test('noAbout', () =>
        expect(renderer.create(<Timestamp date="2006-01-02" noAbout={true} />).toJSON()).toMatchSnapshot())
})
