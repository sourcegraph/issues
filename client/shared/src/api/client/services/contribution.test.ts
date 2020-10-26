import { noop } from 'lodash'
import { Observable, of, Subscription } from 'rxjs'
import { TestScheduler } from 'rxjs/testing'
import { EMPTY_SETTINGS_CASCADE, SettingsCascadeOrError } from '../../../settings/settings'
import { ContributableMenu, Contributions, Evaluated } from '../../protocol'
import { Context, ContributionScope } from '../context/context'
import { parse, parseTemplate } from '../context/expr/evaluator'
import {
    ContributionRegistry,
    ContributionsEntry,
    evaluateContributions,
    filterContributions,
    mergeContributions,
    parseContributionExpressions,
} from './contribution'
import { createTestViewerService } from './viewerService.test'
import { ModelService } from './modelService'

const scheduler = (): TestScheduler => new TestScheduler((a, b) => expect(a).toEqual(b))

const FIXTURE_CONTRIBUTIONS_1: Contributions = {
    actions: [
        { id: '1.a', command: 'c', title: parseTemplate('1.A') },
        { id: '1.b', command: 'c', title: parseTemplate('1.B') },
    ],
    menus: {
        [ContributableMenu.CommandPalette]: [{ action: '1.a' }],
        [ContributableMenu.GlobalNav]: [{ action: '1.a' }, { action: '1.b' }],
    },
}
const FIXTURE_CONTRIBUTIONS_1_EVALUATED: Evaluated<Contributions> = {
    actions: [
        { id: '1.a', command: 'c', title: '1.A' },
        { id: '1.b', command: 'c', title: '1.B' },
    ],
    menus: {
        [ContributableMenu.CommandPalette]: [{ action: '1.a', when: undefined }],
        [ContributableMenu.GlobalNav]: [
            { action: '1.a', when: undefined },
            { action: '1.b', when: undefined },
        ],
    },
}

const FIXTURE_CONTRIBUTIONS_2: Contributions = {
    actions: [
        { id: '2.a', command: 'c', title: parseTemplate('2.A') },
        { id: '2.b', command: 'c', title: parseTemplate('2.B') },
    ],
    menus: {
        [ContributableMenu.CommandPalette]: [{ action: '2.a' }],
        [ContributableMenu.EditorTitle]: [{ action: '2.a' }, { action: '2.b' }],
    },
}
const FIXTURE_CONTRIBUTIONS_2_EVALUATED: Evaluated<Contributions> = {
    actions: [
        { id: '2.a', command: 'c', title: '2.A' },
        { id: '2.b', command: 'c', title: '2.B' },
    ],
    menus: {
        [ContributableMenu.CommandPalette]: [{ action: '2.a', when: undefined }],
        [ContributableMenu.EditorTitle]: [
            { action: '2.a', when: undefined },
            { action: '2.b', when: undefined },
        ],
    },
}

const FIXTURE_CONTRIBUTIONS_MERGED: Evaluated<Contributions> = {
    actions: [
        { id: '1.a', command: 'c', title: '1.A' },
        { id: '1.b', command: 'c', title: '1.B' },
        { id: '2.a', command: 'c', title: '2.A' },
        { id: '2.b', command: 'c', title: '2.B' },
    ],
    menus: {
        [ContributableMenu.CommandPalette]: [
            { action: '1.a', when: undefined },
            { action: '2.a', when: undefined },
        ],
        [ContributableMenu.GlobalNav]: [
            { action: '1.a', when: undefined },
            { action: '1.b', when: undefined },
        ],
        [ContributableMenu.EditorTitle]: [
            { action: '2.a', when: undefined },
            { action: '2.b', when: undefined },
        ],
    },
}

const TEST_MODEL_SERVICE: Pick<ModelService, 'getPartialModel'> = {
    getPartialModel: () => ({ languageId: 'x' }),
}

describe('ContributionRegistry', () => {
    test('is initially empty', () => {
        expect(
            new ContributionRegistry(
                createTestViewerService({}),
                TEST_MODEL_SERVICE,
                of(EMPTY_SETTINGS_CASCADE),
                of({})
            ).entries.value
        ).toEqual([])
    })

    test('registers and unregisters contributions', () => {
        const subscriptions = new Subscription()
        const registry = new ContributionRegistry(
            createTestViewerService({}),
            TEST_MODEL_SERVICE,
            of(EMPTY_SETTINGS_CASCADE),
            of({})
        )
        const entry1: ContributionsEntry = { contributions: FIXTURE_CONTRIBUTIONS_1 }
        const entry2: ContributionsEntry = { contributions: FIXTURE_CONTRIBUTIONS_2 }

        const unregister1 = subscriptions.add(registry.registerContributions(entry1))
        expect(registry.entries.value).toEqual([entry1])

        const unregister2 = subscriptions.add(registry.registerContributions(entry2))
        expect(registry.entries.value).toEqual([entry1, entry2])

        unregister1.unsubscribe()
        expect(registry.entries.value).toEqual([entry2])

        unregister2.unsubscribe()
        expect(registry.entries.value).toEqual([])
    })

    test('replaces contributions', () => {
        const registry = new ContributionRegistry(
            createTestViewerService({}),
            TEST_MODEL_SERVICE,
            of(EMPTY_SETTINGS_CASCADE),
            of({})
        )
        const entry1: ContributionsEntry = { contributions: FIXTURE_CONTRIBUTIONS_1 }
        const entry2: ContributionsEntry = { contributions: FIXTURE_CONTRIBUTIONS_2 }

        const unregister1 = registry.registerContributions(entry1)
        expect(registry.entries.value).toEqual([entry1])

        const unregister2 = registry.replaceContributions(unregister1, entry2)
        expect(registry.entries.value).toEqual([entry2])

        unregister1.unsubscribe()
        expect(registry.entries.value).toEqual([entry2])

        unregister2.unsubscribe()
        expect(registry.entries.value).toEqual([])
    })

    describe('contributions observable', () => {
        test('emits stream of results of registrations', () => {
            scheduler().run(({ cold, expectObservable }) => {
                const registry = new (class extends ContributionRegistry {
                    public getContributionsFromEntries(
                        entries: Observable<ContributionsEntry[]>
                    ): Observable<Evaluated<Contributions>> {
                        return super.getContributionsFromEntries(entries, undefined)
                    }
                })(
                    {
                        activeViewerUpdates: of(undefined),
                    },
                    TEST_MODEL_SERVICE,
                    of(EMPTY_SETTINGS_CASCADE),
                    of({})
                )
                type Marble = 'a' | 'b' | 'c'
                const values: Record<Marble, ContributionsEntry[]> = {
                    a: [{ contributions: FIXTURE_CONTRIBUTIONS_1 }],
                    b: [{ contributions: FIXTURE_CONTRIBUTIONS_1 }, { contributions: {} }],
                    c: [{ contributions: FIXTURE_CONTRIBUTIONS_1 }, { contributions: FIXTURE_CONTRIBUTIONS_2 }],
                }
                const expected: Record<Exclude<Marble, 'b'>, Evaluated<Contributions>> = {
                    a: FIXTURE_CONTRIBUTIONS_1_EVALUATED,
                    c: FIXTURE_CONTRIBUTIONS_MERGED,
                }
                expectObservable(registry.getContributionsFromEntries(cold('-a-b-c-|', values))).toBe(
                    '-a---c-|',
                    expected
                )
            })
        })

        it('supports registration of an observable', () => {
            scheduler().run(({ cold, expectObservable }) => {
                const registry = new (class extends ContributionRegistry {
                    public getContributionsFromEntries(
                        entries: Observable<ContributionsEntry[]>
                    ): Observable<Evaluated<Contributions>> {
                        return super.getContributionsFromEntries(entries, undefined)
                    }
                })(
                    {
                        activeViewerUpdates: of(undefined),
                    },
                    TEST_MODEL_SERVICE,
                    of(EMPTY_SETTINGS_CASCADE),
                    of({})
                )
                type Marble = 'b' | 'c'
                const values: Record<Marble, Contributions | Contributions[]> = {
                    b: FIXTURE_CONTRIBUTIONS_1,
                    c: [FIXTURE_CONTRIBUTIONS_2],
                }
                const entries = cold<ContributionsEntry[]>('-a-----|', {
                    a: [
                        {
                            contributions: cold('--b-c-|', values),
                        },
                    ],
                })
                const expected: Record<Marble, Evaluated<Contributions>> = {
                    b: FIXTURE_CONTRIBUTIONS_1_EVALUATED,
                    c: FIXTURE_CONTRIBUTIONS_2_EVALUATED,
                }
                expectObservable(registry.getContributionsFromEntries(entries)).toBe('---b-c-|', expected)
            })
        })

        test('emits when context changes and filters on context', () => {
            scheduler().run(({ cold, expectObservable }) => {
                const registry = new (class extends ContributionRegistry {
                    constructor() {
                        super(
                            {
                                activeViewerUpdates: of(undefined),
                            },
                            TEST_MODEL_SERVICE,

                            cold<SettingsCascadeOrError>('-a-b-|', {
                                a: EMPTY_SETTINGS_CASCADE,
                                b: EMPTY_SETTINGS_CASCADE,
                            }),

                            cold<Context>('-a-b-|', { a: { x: 1, y: 2 }, b: { x: 1, y: 1 } })
                        )
                    }

                    public getContributionsFromEntries(
                        entries: Observable<ContributionsEntry[]>
                    ): Observable<Evaluated<Contributions>> {
                        return super.getContributionsFromEntries(entries, undefined)
                    }
                })()
                const values: Record<'a' | 'b', Evaluated<Contributions>> = {
                    a: { menus: { commandPalette: [] } },
                    b: { menus: { commandPalette: [{ action: 'a', when: true }] } },
                }
                expectObservable(
                    registry.getContributionsFromEntries(
                        of<ContributionsEntry[]>([
                            {
                                contributions: {
                                    menus: { commandPalette: [{ action: 'a', when: parse('x == y') }] },
                                },
                            },
                        ])
                    )
                ).toBe('-a-b-|', values)
            })
        })

        test('continues after error thrown during evaluation', () => {
            scheduler().run(({ cold, expectObservable }) => {
                const registry = new (class extends ContributionRegistry {
                    constructor() {
                        super(
                            createTestViewerService({}),
                            TEST_MODEL_SERVICE,
                            cold<SettingsCascadeOrError>('a', { a: EMPTY_SETTINGS_CASCADE }),
                            cold<Context>('a', {})
                        )
                    }

                    public getContributionsFromEntries(
                        entries: Observable<ContributionsEntry[]>,
                        scope?: ContributionScope
                    ): Observable<Evaluated<Contributions>> {
                        return super.getContributionsFromEntries(entries, scope, undefined, noop)
                    }
                })()
                expectObservable(
                    registry.getContributionsFromEntries(
                        of<ContributionsEntry[]>([
                            {
                                // Expression "!" will cause an error to be thrown.
                                contributions: {
                                    menus: { commandPalette: [{ action: 'a1', when: parse('nonExistingVar') }] },
                                },
                            },
                            {
                                contributions: {
                                    menus: { commandPalette: [{ action: 'a2' }] },
                                },
                            },
                        ])
                    )
                ).toBe('a', {
                    a: { menus: { commandPalette: [{ action: 'a2' }] } },
                })
            })
        })
    })
})

describe('mergeContributions()', () => {
    const FIXTURE_CONTRIBUTIONS_1: Evaluated<Contributions> = {
        actions: [
            { id: '1.a', command: 'c', title: '1.A' },
            { id: '1.b', command: 'c', title: '1.B' },
        ],
        menus: {
            [ContributableMenu.CommandPalette]: [{ action: '1.a' }],
            [ContributableMenu.GlobalNav]: [{ action: '1.a' }, { action: '1.b' }],
        },
    }

    const FIXTURE_CONTRIBUTIONS_2: Evaluated<Contributions> = {
        actions: [
            { id: '2.a', command: 'c', title: '2.A' },
            { id: '2.b', command: 'c', title: '2.B' },
        ],
        menus: {
            [ContributableMenu.CommandPalette]: [{ action: '2.a' }],
            [ContributableMenu.EditorTitle]: [{ action: '2.a' }, { action: '2.b' }],
        },
    }
    const FIXTURE_CONTRIBUTIONS_MERGED: Evaluated<Contributions> = {
        actions: [
            { id: '1.a', command: 'c', title: '1.A' },
            { id: '1.b', command: 'c', title: '1.B' },
            { id: '2.a', command: 'c', title: '2.A' },
            { id: '2.b', command: 'c', title: '2.B' },
        ],
        menus: {
            [ContributableMenu.CommandPalette]: [{ action: '1.a' }, { action: '2.a' }],
            [ContributableMenu.GlobalNav]: [{ action: '1.a' }, { action: '1.b' }],
            [ContributableMenu.EditorTitle]: [{ action: '2.a' }, { action: '2.b' }],
        },
    }

    test('handles an empty array', () => {
        expect(mergeContributions([])).toEqual({})
    })
    test('handles a single item', () => {
        expect(mergeContributions([FIXTURE_CONTRIBUTIONS_1])).toEqual(FIXTURE_CONTRIBUTIONS_1)
    })
    test('handles multiple items', () => {
        expect(
            mergeContributions([FIXTURE_CONTRIBUTIONS_1, FIXTURE_CONTRIBUTIONS_2, {}, { actions: [] }, { menus: {} }])
        ).toEqual(FIXTURE_CONTRIBUTIONS_MERGED)
    })
})

const FIXTURE_CONTEXT: Context = {
    a: true,
    b: false,
    replaceMe: 'x',
}

describe('filterContributions()', () => {
    it('handles empty contributions', () => {
        expect(filterContributions({})).toEqual({})
    })

    it('handles empty menu contributions', () => {
        const expected: Evaluated<Contributions> = {
            menus: {},
        }
        expect(filterContributions({ menus: {} })).toEqual(expected)
    })

    it('handles empty array of menu contributions', () => {
        const expected: Evaluated<Contributions> = {
            menus: { commandPalette: [] },
        }
        expect(filterContributions({ menus: { commandPalette: [] } })).toEqual(expected)
    })

    it('handles non-empty contributions', () => {
        const expected: Evaluated<Contributions> = {
            actions: [
                { id: 'a1', command: 'c' },
                { id: 'a2', command: 'c' },
                { id: 'a3', command: 'c' },
            ],
            menus: {
                [ContributableMenu.CommandPalette]: [{ action: 'a1', when: true }, { action: 'a3' }],
                [ContributableMenu.GlobalNav]: [{ action: 'a1', when: true }, { action: 'a2' }],
            },
        }
        expect(
            filterContributions({
                actions: [
                    { id: 'a1', command: 'c' },
                    { id: 'a2', command: 'c' },
                    { id: 'a3', command: 'c' },
                ],
                menus: {
                    [ContributableMenu.CommandPalette]: [
                        { action: 'a1', when: true },
                        { action: 'a2', when: false },
                        { action: 'a3' },
                    ],
                    [ContributableMenu.GlobalNav]: [
                        { action: 'a1', when: true },
                        { action: 'a2' },
                        { action: 'a3', when: false },
                    ],
                },
            })
        ).toEqual(expected)
    })
})

/* eslint-disable no-template-curly-in-string */
describe('evaluateContributions()', () => {
    test('handles empty contributions', () => {
        const expected: Evaluated<Contributions> = {}
        expect(evaluateContributions({}, {})).toEqual(expected)
    })

    test('handles empty array of command contributions', () => {
        const expected: Evaluated<Contributions> = {
            actions: [],
        }
        expect(evaluateContributions({}, { actions: [] })).toEqual(expected)
    })

    test('handles non-empty contributions', () => {
        const input: Contributions = {
            actions: [
                {
                    id: 'a1',
                    command: 'c1',
                    commandArguments: [
                        parseTemplate('${replaceMe}'),
                        parseTemplate('b'),
                        parseTemplate('${replaceMe}'),
                    ],
                    title: parseTemplate('${replaceMe}'),
                    category: parseTemplate('${replaceMe}'),
                    description: parseTemplate('${replaceMe}'),
                    iconURL: parseTemplate('${replaceMe}'),
                    actionItem: {
                        label: parseTemplate('${replaceMe}'),
                        description: parseTemplate('${replaceMe}'),
                        iconDescription: parseTemplate('${replaceMe}'),
                        iconURL: parseTemplate('${replaceMe}'),
                    },
                },
                {
                    id: 'a2',
                    command: 'c2',
                    title: parseTemplate('${replaceMe}'),
                    category: parseTemplate('b'),
                },
                {
                    id: 'a3',
                    command: 'c3',
                    title: parseTemplate('b'),
                    category: parseTemplate('b'),
                    actionItem: {
                        label: parseTemplate('${replaceMe}'),
                        description: parseTemplate('b'),
                    },
                },
            ],
        }
        const origInput = JSON.parse(JSON.stringify(input))
        const expected: Evaluated<Contributions> = {
            actions: [
                {
                    id: 'a1',
                    command: 'c1',
                    commandArguments: ['x', 'b', 'x'],
                    title: 'x',
                    category: 'x',
                    description: 'x',
                    iconURL: 'x',
                    actionItem: {
                        label: 'x',
                        description: 'x',
                        iconDescription: 'x',
                        iconURL: 'x',
                    },
                },
                { id: 'a2', command: 'c2', title: 'x', category: 'b' },
                { id: 'a3', command: 'c3', title: 'b', category: 'b', actionItem: { label: 'x', description: 'b' } },
            ],
        }
        expect(evaluateContributions(FIXTURE_CONTEXT, input)).toEqual(expected)
        expect(input).toEqual(origInput)
    })

    test('supports commandArguments with the first element non-evaluated', () => {
        const expected: Evaluated<Contributions> = {
            actions: [
                {
                    id: 'x',
                    command: 'c',
                    commandArguments: ['b', 'x', 'b', 'x'],
                },
            ],
        }
        expect(
            evaluateContributions(FIXTURE_CONTEXT, {
                actions: [
                    {
                        id: 'x',
                        command: 'c',
                        commandArguments: [
                            parseTemplate('b'),
                            parseTemplate('${replaceMe}'),
                            parseTemplate('b'),
                            parseTemplate('${replaceMe}'),
                        ],
                    },
                ],
            })
        ).toEqual(expected)
    })

    test('evaluates `actionItem.pressed` if present', () => {
        const input: Contributions = {
            actions: [
                {
                    id: 'a',
                    command: 'c',
                    title: parseTemplate('a'),
                    actionItem: {
                        pressed: parse('a'),
                    },
                },
            ],
        }
        expect(evaluateContributions(FIXTURE_CONTEXT, input)).toEqual({
            actions: [{ id: 'a', command: 'c', title: 'a', actionItem: { pressed: true } }],
        })
    })
})

describe('parseContributionExpressions()', () => {
    it('should not parse the `id` or `command` values', () => {
        const expected: Contributions = {
            actions: [{ id: '${replaceMe}', command: '${c}' }],
        }
        expect(
            parseContributionExpressions({
                actions: [{ id: '${replaceMe}', command: '${c}' }],
            })
        ).toEqual(expected)
    })
})
/* eslint-enable no-template-curly-in-string */
