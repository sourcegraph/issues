import { Selection } from '@sourcegraph/extension-api-classes'
import { Location } from '@sourcegraph/extension-api-types'
import { Observable, of } from 'rxjs'
import { TestScheduler } from 'rxjs/testing'
import { TextDocumentRegistrationOptions } from '../../protocol'
import {
    getLocationsFromProviders,
    ProvideTextDocumentLocationSignature,
    TextDocumentLocationProviderRegistry,
} from './location'
import { Entry } from './registry'
import { FIXTURE } from './registry.test'
import { first } from 'rxjs/operators'

const scheduler = (): TestScheduler => new TestScheduler((actual, expected) => expect(actual).toEqual(expected))

const FIXTURE_LOCATION: Location = {
    uri: 'file:///f',
    range: { start: { line: 1, character: 2 }, end: { line: 3, character: 4 } },
}
const FIXTURE_LOCATIONS: Location | Location[] | null = [FIXTURE_LOCATION, FIXTURE_LOCATION]

/**
 * Allow overriding {@link TextDocumentLocationProviderRegistry#entries} for tests.
 */
class TestTextDocumentLocationProviderRegistry extends TextDocumentLocationProviderRegistry {
    constructor(entries?: Observable<Entry<TextDocumentRegistrationOptions, ProvideTextDocumentLocationSignature>[]>) {
        super()
        if (entries) {
            entries.subscribe(entries => this.entries.next(entries))
        }
    }
}

describe('TextDocumentLocationProviderRegistry', () => {
    describe('hasProvidersForActiveTextDocument', () => {
        test('false if no position params', async () => {
            const registry = new TestTextDocumentLocationProviderRegistry(
                of([{ provider: () => of(null), registrationOptions: { documentSelector: ['*'] } }])
            )
            expect(await registry.hasProvidersForActiveTextDocument(undefined).pipe(first()).toPromise()).toBe(false)
        })

        test('true if matching document', () => {
            scheduler().run(({ cold, expectObservable }) => {
                const registry = new TestTextDocumentLocationProviderRegistry(
                    cold<Entry<TextDocumentRegistrationOptions, ProvideTextDocumentLocationSignature>[]>('a', {
                        a: [{ provider: () => of(null), registrationOptions: { documentSelector: ['l'] } }],
                    })
                )
                expectObservable(
                    registry.hasProvidersForActiveTextDocument({
                        isActive: true,
                        viewerId: 'viewer#0',
                        type: 'CodeEditor' as const,
                        selections: [new Selection(1, 2, 3, 4).toPlain()],
                        resource: 'file:///g',
                        model: { languageId: 'l' },
                    })
                ).toBe('a', {
                    a: true,
                })
            })
        })

        test('false if no matching document', () => {
            scheduler().run(({ cold, expectObservable }) => {
                const registry = new TestTextDocumentLocationProviderRegistry(
                    cold<Entry<TextDocumentRegistrationOptions, ProvideTextDocumentLocationSignature>[]>('a', {
                        a: [{ provider: () => of(null), registrationOptions: { documentSelector: ['otherlang'] } }],
                    })
                )
                expectObservable(
                    registry.hasProvidersForActiveTextDocument({
                        isActive: true,
                        viewerId: 'viewer#0',
                        type: 'CodeEditor' as const,
                        selections: [new Selection(1, 2, 3, 4).toPlain()],
                        resource: 'file:///g',
                        model: { languageId: 'l' },
                    })
                ).toBe('a', {
                    a: false,
                })
            })
        })
    })
})

describe('getLocationsFromProviders', () => {
    describe('0 providers', () => {
        it('emits an empty non-loading result', () => {
            scheduler().run(({ cold, expectObservable }) => {
                expectObservable(
                    getLocationsFromProviders(
                        cold<ProvideTextDocumentLocationSignature[]>('-a-|', { a: [] }),
                        FIXTURE.TextDocumentPositionParams
                    )
                ).toBe('-a-|', {
                    a: { isLoading: false, result: [] },
                })
            })
        })
    })

    describe('1 provider', () => {
        it('emits an empty result from provider', () => {
            scheduler().run(({ cold, expectObservable }) => {
                expectObservable(
                    getLocationsFromProviders(
                        cold<ProvideTextDocumentLocationSignature[]>('-a', { a: [() => of(null)] }),
                        FIXTURE.TextDocumentPositionParams
                    )
                ).toBe('-(lr)', {
                    l: { isLoading: true, result: [] },
                    r: { isLoading: false, result: [] },
                })
            })
        })

        it('returns result array from provider', () => {
            scheduler().run(({ cold, expectObservable }) => {
                expectObservable(
                    getLocationsFromProviders(
                        cold<ProvideTextDocumentLocationSignature[]>('-a', {
                            a: [() => cold('-a', { a: FIXTURE_LOCATIONS })],
                        }),
                        FIXTURE.TextDocumentPositionParams
                    )
                ).toBe('-lr', {
                    l: { isLoading: true, result: [] },
                    r: { isLoading: false, result: FIXTURE_LOCATIONS },
                })
            })
        })
    })

    it('returns the results of other providers even if a provider errors', () => {
        scheduler().run(({ cold, expectObservable }) => {
            expectObservable(
                getLocationsFromProviders(
                    cold<ProvideTextDocumentLocationSignature[]>('-a', {
                        a: [() => cold('-a', { a: [FIXTURE_LOCATION] }), () => cold('-#', {}, new Error('x'))],
                    }),
                    FIXTURE.TextDocumentPositionParams,
                    false
                )
            ).toBe('-lr', {
                l: { isLoading: true, result: [] },
                r: { isLoading: false, result: [FIXTURE_LOCATION] },
            })
        })
    })

    describe('2 providers', () => {
        it('returns an empty result if both providers return an empty result', () => {
            scheduler().run(({ cold, expectObservable }) => {
                expectObservable(
                    getLocationsFromProviders(
                        cold<ProvideTextDocumentLocationSignature[]>('-a', {
                            a: [() => of(null), () => of(null)],
                        }),
                        FIXTURE.TextDocumentPositionParams
                    )
                ).toBe('-(lr)', {
                    l: { isLoading: true, result: [] },
                    r: { isLoading: false, result: [] },
                })
            })
        })

        it('omits null result from 1 provider', () => {
            scheduler().run(({ cold, expectObservable }) => {
                expectObservable(
                    getLocationsFromProviders(
                        cold<ProvideTextDocumentLocationSignature[]>('-a', {
                            a: [() => cold('-a', { a: FIXTURE_LOCATIONS }), () => of(null)],
                        }),
                        FIXTURE.TextDocumentPositionParams
                    )
                ).toBe('-lr', {
                    l: { isLoading: true, result: [] },
                    r: { isLoading: false, result: FIXTURE_LOCATIONS },
                })
            })
        })

        it('merges results from providers', () => {
            scheduler().run(({ cold, expectObservable }) => {
                expectObservable(
                    getLocationsFromProviders(
                        cold<ProvideTextDocumentLocationSignature[]>('-a', {
                            a: [
                                () =>
                                    of([
                                        {
                                            uri: 'file:///f1',
                                            range: { start: { line: 1, character: 2 }, end: { line: 3, character: 4 } },
                                        },
                                    ]),
                                () =>
                                    of([
                                        {
                                            uri: 'file:///f2',
                                            range: { start: { line: 5, character: 6 }, end: { line: 7, character: 8 } },
                                        },
                                    ]),
                            ],
                        }),
                        FIXTURE.TextDocumentPositionParams
                    )
                ).toBe('-(ab)', {
                    // Partial result (first provider emitted)
                    a: {
                        isLoading: true,
                        result: [
                            {
                                uri: 'file:///f1',
                                range: { start: { line: 1, character: 2 }, end: { line: 3, character: 4 } },
                            },
                        ],
                    },
                    // Full result
                    b: {
                        isLoading: false,
                        result: [
                            {
                                uri: 'file:///f1',
                                range: { start: { line: 1, character: 2 }, end: { line: 3, character: 4 } },
                            },
                            {
                                uri: 'file:///f2',
                                range: { start: { line: 5, character: 6 }, end: { line: 7, character: 8 } },
                            },
                        ],
                    },
                })
            })
        })
    })

    describe('multiple emissions', () => {
        it('returns stream of results', () => {
            scheduler().run(({ cold, expectObservable }) => {
                expectObservable(
                    getLocationsFromProviders(
                        cold<ProvideTextDocumentLocationSignature[]>('-a----b', {
                            a: [() => of(FIXTURE_LOCATIONS)],
                            b: [() => of(null)],
                        }),
                        FIXTURE.TextDocumentPositionParams
                    )
                ).toBe('-(ab)-(cd)', {
                    a: { isLoading: true, result: [] },
                    b: { isLoading: false, result: FIXTURE_LOCATIONS },
                    c: { isLoading: true, result: [] },
                    d: { isLoading: false, result: [] },
                })
            })
        })
    })
})
