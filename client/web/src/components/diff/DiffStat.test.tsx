import React from 'react'
import renderer from 'react-test-renderer'

import { DiffStat, DiffStatSquares, DiffStatStack } from './DiffStat'

describe('DiffStat', () => {
    test('standard', () =>
        expect(renderer.create(<DiffStat added={1} changed={2} deleted={3} />).toJSON()).toMatchSnapshot())

    test('expandedCounts', () =>
        expect(
            renderer.create(<DiffStat added={1} changed={2} deleted={3} expandedCounts={true} />).toJSON()
        ).toMatchSnapshot())
})

test('DiffStatSquares', () =>
    expect(renderer.create(<DiffStatSquares added={1} changed={2} deleted={3} />).toJSON()).toMatchSnapshot())

test('DiffStatStack', () =>
    expect(renderer.create(<DiffStatStack added={1} changed={2} deleted={3} />).toJSON()).toMatchSnapshot())
