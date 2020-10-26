import { createAggregateError, isErrorLike } from '../util/errors'
import {
    CustomMergeFunctions,
    gqlToCascade,
    merge,
    mergeSettings,
    Settings,
    SettingsCascade,
    SettingsSubject,
    SubjectSettingsContents,
} from './settings'

const FIXTURE_ORG: SettingsSubject & SubjectSettingsContents = {
    __typename: 'Org',
    name: 'n',
    displayName: 'n',
    id: 'a',
    viewerCanAdminister: true,
    latestSettings: { id: 1, contents: '{"a":1}' },
}

const FIXTURE_USER: SettingsSubject & SubjectSettingsContents = {
    __typename: 'User',
    username: 'n',
    displayName: 'n',
    id: 'b',
    viewerCanAdminister: true,
    latestSettings: { id: 2, contents: '{"b":2}' },
}

const FIXTURE_USER_WITH_SETTINGS_ERROR: SettingsSubject & SubjectSettingsContents = {
    ...FIXTURE_USER,
    id: 'c',
    latestSettings: { id: 3, contents: '.' },
}

const SETTINGS_ERROR_FOR_FIXTURE_USER = createAggregateError([new Error('parse error (code: 0, offset: 0, length: 1)')])

describe('gqlToCascade', () => {
    test('converts a value', () => {
        const expected: SettingsCascade = {
            subjects: [
                { subject: FIXTURE_ORG, settings: { a: 1 }, lastID: 1 },
                { subject: FIXTURE_USER, settings: { b: 2 }, lastID: 2 },
            ],
            final: { a: 1, b: 2 },
        }
        expect(
            gqlToCascade({
                subjects: [FIXTURE_ORG, FIXTURE_USER],
            })
        ).toEqual(expected)
    })
    test('preserves errors', () => {
        const value = gqlToCascade({
            subjects: [FIXTURE_ORG, FIXTURE_USER_WITH_SETTINGS_ERROR, FIXTURE_USER],
        })
        expect(isErrorLike(value.final) && value.final.message).toBe(SETTINGS_ERROR_FOR_FIXTURE_USER.message)
        expect(
            value.subjects &&
                !isErrorLike(value.subjects) &&
                isErrorLike(value.subjects[1].settings) &&
                value.subjects[1].settings.message
        ).toBe(SETTINGS_ERROR_FOR_FIXTURE_USER.message)
    })
})

describe('mergeSettings', () => {
    test('handles an empty array', () => expect(mergeSettings([])).toBe(null))
    test('merges multiple values', () =>
        expect(
            mergeSettings<{ a?: number; b?: number } & Settings>([{ a: 1 }, { b: 2 }, { a: 3 }])
        ).toEqual({
            a: 3,
            b: 2,
        }))
    test('deeply merges extensions property', () =>
        expect(
            mergeSettings<{ a?: { [key: string]: boolean }; b?: { [key: string]: boolean } } & Settings>([
                { extensions: { 'sourcegraph/cpp': true, 'sourcegraph/go': true, 'sourcegraph/typescript': true } },
                { extensions: { 'sourcegraph/cpp': false, 'sourcegraph/go': true, 'sourcegraph/typescript': true } },
                { extensions: { 'sourcegraph/go': false, 'sourcegraph/typescript': true } },
            ])
        ).toEqual({
            extensions: { 'sourcegraph/cpp': false, 'sourcegraph/go': false, 'sourcegraph/typescript': true },
        }))
    test('merges experimentalFeatures property', () =>
        expect(
            mergeSettings<Settings & { experimentalFeatures: { a?: boolean; b?: boolean; c?: boolean } }>([
                { experimentalFeatures: { a: true, b: true } },
                { experimentalFeatures: { b: false, c: true } },
            ])
        ).toEqual({
            experimentalFeatures: { a: true, b: false, c: true },
        }))
    test('merges search.scopes property', () =>
        expect(
            mergeSettings<
                {
                    a?: { [key: string]: { [key: string]: string }[] }
                    b?: { [key: string]: { [key: string]: string }[] }
                } & Settings
            >([
                { 'search.scopes': [{ name: 'test repos', value: 'repogroup:test' }] },
                { 'search.scopes': [{ name: 'sourcegraph repos', value: 'repogroup:sourcegraph' }] },
            ])
        ).toEqual({
            'search.scopes': [
                { name: 'test repos', value: 'repogroup:test' },
                { name: 'sourcegraph repos', value: 'repogroup:sourcegraph' },
            ],
        }))
    test('merges quicklinks property', () =>
        expect(
            mergeSettings<
                {
                    a?: { [key: string]: { [key: string]: string }[] }
                    b?: { [key: string]: { [key: string]: string }[] }
                } & Settings
            >([
                { quicklinks: [{ name: 'main repo', value: '/github.com/org/main-repo' }] },
                { quicklinks: [{ name: 'About Sourcegraph', value: 'https://docs.internal/about-sourcegraph' }] },
                {
                    quicklinks: [
                        { name: 'mycorp extensions', value: 'https://sourcegraph.com/extensions?query=mycorp%2F' },
                    ],
                },
            ])
        ).toEqual({
            quicklinks: [
                { name: 'main repo', value: '/github.com/org/main-repo' },
                { name: 'About Sourcegraph', value: 'https://docs.internal/about-sourcegraph' },
                { name: 'mycorp extensions', value: 'https://sourcegraph.com/extensions?query=mycorp%2F' },
            ],
        }))
    test('merges search.repositoryGroups property', () =>
        expect(
            mergeSettings<{ a?: { [key: string]: string }; b?: { [key: string]: string } } & Settings>([
                {
                    'search.repositoryGroups': {
                        sourcegraph: ['github.com/sourcegraph/sourcegraph', 'github.com/sourcegraph/codeintellify'],
                    },
                },
                {
                    'search.repositoryGroups': {
                        k8s: ['github.com/kubernetes/kubernetes'],
                    },
                },
                {
                    'search.repositoryGroups': {
                        docker: ['github.com/docker/docker'],
                        sourcegraph: [
                            'github.com/sourcegraph/sourcegraph',
                            'github.com/sourcegraph/codeintellify',
                            'github.com/sourcegraph/sourcegraph-typescript',
                        ],
                    },
                },
            ])
        ).toEqual({
            'search.repositoryGroups': {
                k8s: ['github.com/kubernetes/kubernetes'],
                docker: ['github.com/docker/docker'],
                sourcegraph: [
                    'github.com/sourcegraph/sourcegraph',
                    'github.com/sourcegraph/codeintellify',
                    'github.com/sourcegraph/sourcegraph-typescript',
                ],
            },
        }))
    test('merges notices property', () =>
        expect(
            mergeSettings<{ a?: { [key: string]: string }; b?: { [key: string]: string } } & Settings>([
                {
                    notices: [
                        {
                            dismissible: false,
                            location: 'home',
                            message: 'global notice',
                        },
                    ],
                },
                {
                    notices: [
                        {
                            dismissible: false,
                            location: 'top',
                            message: 'org notice',
                        },
                    ],
                },
                {
                    notices: [
                        {
                            dismissible: false,
                            location: 'top',
                            message: 'user notice',
                        },
                    ],
                },
            ])
        ).toEqual({
            notices: [
                {
                    dismissible: false,
                    location: 'home',
                    message: 'global notice',
                },
                {
                    dismissible: false,
                    location: 'top',
                    message: 'org notice',
                },

                {
                    dismissible: false,
                    location: 'top',
                    message: 'user notice',
                },
            ],
        }))
    test('merges search.savedQueries property', () =>
        expect(
            mergeSettings<{ a?: { [key: string]: string }; b?: { [key: string]: string } } & Settings>([
                {
                    'search.savedQueries': [
                        {
                            description: 'global saved query',
                            query: 'type:diff global',
                            notify: true,
                        },
                    ],
                },
                {
                    'search.savedQueries': [
                        {
                            description: 'org saved query',
                            query: 'type:diff org',
                            notify: true,
                        },
                    ],
                },
                {
                    'search.savedQueries': [
                        {
                            description: 'user saved query',
                            query: 'type:diff user',
                            notify: true,
                        },
                    ],
                },
            ])
        ).toEqual({
            'search.savedQueries': [
                {
                    description: 'global saved query',
                    query: 'type:diff global',
                    notify: true,
                },
                {
                    description: 'org saved query',
                    query: 'type:diff org',
                    notify: true,
                },
                {
                    description: 'user saved query',
                    query: 'type:diff user',
                    notify: true,
                },
            ],
        }))
})

describe('merge', () => {
    function assertMerged(base: any, add: any, expected: any, custom?: CustomMergeFunctions): void {
        merge(base, add, custom)
        expect(base).toEqual(expected)
    }

    test('merges with empty', () => {
        assertMerged({ a: 1 }, {}, { a: 1 })
        assertMerged({}, { a: 1 }, { a: 1 })
    })
    test('merges top-level objects deeply', () => assertMerged({ a: 1 }, { b: 2 }, { a: 1, b: 2 }))
    test('does not merge nested objects deeply', () => assertMerged({ a: { b: 1 } }, { a: { c: 2 } }, { a: { c: 2 } }))
    test('overwrites arrays', () => assertMerged({ a: [1] }, { a: [2] }, { a: [2] }))
    test('uses custom merge functions', () =>
        assertMerged({ a: [1] }, { a: [2] }, { a: [1, 2] }, { a: (base, add) => [...base, ...add] }))
})
