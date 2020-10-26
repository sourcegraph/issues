import { from, of, Subscribable, throwError } from 'rxjs'
import { TestScheduler } from 'rxjs/testing'
import { ConfiguredExtension } from '../../../extensions/extension'
import { EMPTY_SETTINGS_CASCADE, SettingsCascadeOrError } from '../../../settings/settings'
import { ExecutableExtension, ExtensionsService } from './extensionsService'
import { ModelService } from './modelService'
import { PlatformContext } from '../../../platform/context'

const scheduler = (): TestScheduler => new TestScheduler((a, b) => expect(a).toEqual(b))

class TestExtensionsService extends ExtensionsService {
    constructor(
        mockConfiguredExtensions: ConfiguredExtension[],
        modelService: Pick<ModelService, 'activeLanguages'>,
        settings: PlatformContext['settings'],
        extensionActivationFilter: (
            enabledExtensions: ConfiguredExtension[],
            activeLanguages: ReadonlySet<string>
        ) => ConfiguredExtension[],
        sideloadedExtensionURL: Subscribable<string | null>,
        fetchSideloadedExtension: (baseUrl: string) => Subscribable<ConfiguredExtension | null>
    ) {
        super(
            {
                requestGraphQL: () => {
                    throw new Error('not implemented')
                },
                settings,
                getScriptURLForExtension: scriptURL => scriptURL,
                sideloadedExtensionURL,
            },
            modelService,
            extensionActivationFilter,
            fetchSideloadedExtension
        )
        this.configuredExtensions = of(mockConfiguredExtensions)
    }
}

describe('activeExtensions', () => {
    test('emits an empty set', () =>
        scheduler().run(({ cold, expectObservable }) =>
            expectObservable(
                from(
                    new TestExtensionsService(
                        [],
                        {
                            activeLanguages: cold<ReadonlySet<string>>('-a-|', {
                                a: new Set(),
                            }),
                        },
                        cold<SettingsCascadeOrError>('-a-|', { a: EMPTY_SETTINGS_CASCADE }),
                        enabledExtensions => enabledExtensions,
                        cold('-a-|', { a: '' }),
                        () => of(null)
                    ).activeExtensions
                )
            ).toBe('-a-|', {
                a: [],
            })
        ))

    const manifest = { url: 'u', activationEvents: [] }
    test('previously activated extensions remain activated when their activationEvents no longer match', () =>
        scheduler().run(({ cold, expectObservable }) =>
            expectObservable(
                from(
                    new TestExtensionsService(
                        [
                            { id: 'x', manifest, rawManifest: null },
                            { id: 'y', manifest, rawManifest: null },
                        ],
                        {
                            activeLanguages: cold<ReadonlySet<string>>('-a-b-|', {
                                a: new Set(['x']),
                                b: new Set(['y']),
                            }),
                        },

                        cold<SettingsCascadeOrError>('-a-b-|', {
                            a: { final: { extensions: { x: true } }, subjects: [] },
                            b: { final: { extensions: { x: true, y: true } }, subjects: [] },
                        }),
                        (enabledExtensions, activeLanguages) =>
                            enabledExtensions.filter(extension => activeLanguages.has(extension.id)),
                        cold('-a--|', { a: '' }),
                        () => of(null)
                    ).activeExtensions
                )
            ).toBe('-a-b-|', {
                a: [{ id: 'x', manifest, scriptURL: 'u' }],
                b: [
                    { id: 'x', manifest, scriptURL: 'u' },
                    { id: 'y', manifest, scriptURL: 'u' },
                ],
            } as Record<string, ExecutableExtension[]>)
        ))

    test('fetches a sideloaded extension and disables the official extension', () => {
        scheduler().run(({ cold, expectObservable }) => {
            expectObservable(
                from(
                    new TestExtensionsService(
                        [
                            { id: 'a/x', manifest, rawManifest: null },
                            { id: 'y', manifest, rawManifest: null },
                        ],
                        {
                            activeLanguages: cold<ReadonlySet<string>>('a-|', {
                                a: new Set([]),
                            }),
                        },
                        cold<SettingsCascadeOrError>('a-|', {
                            a: {
                                final: {
                                    extensions: {
                                        'a/x': true,
                                        y: true,
                                    },
                                },
                                subjects: [],
                            },
                        }),

                        enabledExtensions => enabledExtensions,
                        cold('a-|', { a: 'x' }),
                        baseUrl =>
                            of({
                                id: baseUrl,
                                manifest: {
                                    url: 'x.js',
                                    activationEvents: [],
                                    publisher: 'a',
                                },
                                rawManifest: null,
                            })
                    ).activeExtensions
                )
            ).toBe('a-|', {
                a: [
                    {
                        id: 'y',
                        manifest,
                        scriptURL: 'u',
                    },
                    {
                        id: 'x',
                        manifest: { url: 'x.js', activationEvents: [], publisher: 'a' },
                        scriptURL: 'x.js',
                    },
                ],
            })
        })
    })

    test('still returns registry extensions even if fetching a sideloaded extension fails', () => {
        scheduler().run(({ cold, expectObservable }) => {
            expectObservable(
                from(
                    new TestExtensionsService(
                        [{ id: 'foo', manifest, rawManifest: null }],
                        {
                            activeLanguages: cold<ReadonlySet<string>>('a-|', {
                                a: new Set([]),
                            }),
                        },

                        cold<SettingsCascadeOrError>('a-|', {
                            a: {
                                final: {
                                    extensions: {
                                        foo: true,
                                    },
                                },
                                subjects: [],
                            },
                        }),

                        enabledExtensions => enabledExtensions,
                        cold('a-|', { a: 'bar' }),
                        () => throwError(new Error('baz'))
                    ).activeExtensions
                )
            ).toBe('a-|', {
                a: [{ id: 'foo', manifest, scriptURL: 'u' }],
            })
        })
    })
})
