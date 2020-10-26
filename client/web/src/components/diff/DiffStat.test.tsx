import React from 'react'
import renderer from 'react-test-renderer'
import { DiffStat } from './DiffStat'

describe('DiffStat', () => {
    test('standard', () =>
        expect(renderer.create(<DiffStat added={1} changed={2} deleted={3} />).toJSON()).toMatchSnapshot())

    test('expandedCounts', () =>
        expect(
            renderer.create(<DiffStat added={1} changed={2} deleted={3} expandedCounts={true} />).toJSON()
        ).toMatchSnapshot())
})
