import { Location } from '@sourcegraph/extension-api-types'
import { GroupedLocations, groupLocations } from './locations'

type TestLocation = string

type TestGroup = string

const LOCATIONS: TestLocation[] = ['a/a/0', 'b/a/0', 'a/a/1', 'a/b/0']

const GROUP_KEYS: ((location: TestLocation) => TestGroup | undefined)[] = [
    location => location.split('/')[0],
    location => location.split('/')[1],
]

describe('groupLocations', () => {
    test('groups 1 levels', () =>
        expect(groupLocations<TestLocation, TestGroup>(LOCATIONS, null, GROUP_KEYS.slice(0, 1), LOCATIONS[0])).toEqual({
            groups: [
                [
                    { key: 'a', count: 3 },
                    { key: 'b', count: 1 },
                ],
            ],
            selectedGroups: ['a'],
            visibleLocations: ['a/a/0', 'a/a/1', 'a/b/0'],
        } as GroupedLocations<TestLocation, TestGroup>))

    test('groups 2 levels', () =>
        expect(groupLocations<TestLocation, TestGroup>(LOCATIONS, null, GROUP_KEYS, LOCATIONS[0])).toEqual({
            groups: [
                [
                    { key: 'a', count: 3 },
                    { key: 'b', count: 1 },
                ],
                [
                    { key: 'a', count: 2 },
                    { key: 'b', count: 1 },
                ],
            ],
            selectedGroups: ['a', 'a'],
            visibleLocations: ['a/a/0', 'a/a/1'],
        } as GroupedLocations<TestLocation, TestGroup>))

    test('supports initial selectedGroups', () =>
        expect(groupLocations<TestLocation, TestGroup>(LOCATIONS, ['b', 'a'], GROUP_KEYS, LOCATIONS[0])).toEqual({
            groups: [
                [
                    { key: 'a', count: 3 },
                    { key: 'b', count: 1 },
                ],
                [{ key: 'a', count: 1 }],
            ],
            selectedGroups: ['b', 'a'],
            visibleLocations: ['b/a/0'],
        } as GroupedLocations<TestLocation, TestGroup>))

    test('handles selectedGroups element that does not exist', () =>
        expect(groupLocations<TestLocation, TestGroup>(LOCATIONS, ['b', 'x'], GROUP_KEYS, LOCATIONS[0])).toEqual({
            groups: [
                [
                    { key: 'a', count: 3 },
                    { key: 'b', count: 1 },
                ],
                [{ key: 'a', count: 1 }],
            ],
            selectedGroups: ['b', 'x'],
            visibleLocations: [],
        } as GroupedLocations<TestLocation, TestGroup>))

    test('resolves selectedGroups undefined element', () =>
        expect(groupLocations<TestLocation, TestGroup>(LOCATIONS, ['a'], GROUP_KEYS, LOCATIONS[0])).toEqual({
            groups: [
                [
                    { key: 'a', count: 3 },
                    { key: 'b', count: 1 },
                ],
                [
                    { key: 'a', count: 2 },
                    { key: 'b', count: 1 },
                ],
            ],
            selectedGroups: ['a', 'a'],
            visibleLocations: ['a/a/0', 'a/a/1'],
        } as GroupedLocations<TestLocation, TestGroup>))

    test('dedupes locations', () =>
        expect(
            groupLocations<Location, TestGroup>(
                [
                    { uri: 'a/a' },
                    { uri: 'a/a', range: { start: { line: 1, character: 2 }, end: { line: 3, character: 4 } } },
                    { uri: 'a/a', range: { start: { line: 1, character: 2 }, end: { line: 3, character: 4 } } },
                    { uri: 'a/a', range: { start: { line: 1, character: 2 }, end: { line: 3, character: 4 } } },
                    { uri: 'a/a', range: { start: { line: 11, character: 22 }, end: { line: 33, character: 44 } } },
                ],
                null,
                [() => 'a'],
                { uri: 'a/a' }
            )
        ).toEqual({
            groups: [[{ key: 'a', count: 3 }]],
            selectedGroups: ['a'],
            visibleLocations: [
                { uri: 'a/a' },
                { uri: 'a/a', range: { start: { line: 1, character: 2 }, end: { line: 3, character: 4 } } },
                { uri: 'a/a', range: { start: { line: 11, character: 22 }, end: { line: 33, character: 44 } } },
            ],
        } as GroupedLocations<Location, TestGroup>))
})
