import React from 'react'
import renderer from 'react-test-renderer'
import { Path } from './Path'

describe('Path', () => {
    test('no path components', () => {
        expect(renderer.create(<Path path="" />).toJSON()).toMatchSnapshot()
    })

    test('1 path component', () => {
        expect(renderer.create(<Path path="a" />).toJSON()).toMatchSnapshot()
    })

    test('2 path components', () => {
        expect(renderer.create(<Path path="a/b" />).toJSON()).toMatchSnapshot()
    })

    test('3 path components', () => {
        expect(renderer.create(<Path path="a/b/c" />).toJSON()).toMatchSnapshot()
    })
})
