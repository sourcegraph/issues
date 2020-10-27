import { TestScheduler } from 'rxjs/testing'
import { RegisteredProvider, callProviders } from './flatExtensionApi'
import { Observable } from 'rxjs'

const scheduler = (): TestScheduler => new TestScheduler((a, b) => expect(a).toEqual(b))

type Provider = RegisteredProvider<number | Observable<number>>

const getResultsFromProviders = (providersObservable: Observable<Provider[]>) =>
    providersObservable.pipe(
        callProviders(
            value => value,
            results => results,
            false // < -- logErrors
        )
    )

describe('callProviders()', () => {
    const provide = (number: number | Observable<number>, pattern = '*.ts'): Provider => ({
        provider: number,
        selector: [{ pattern }],
    })

    describe('1 provider', () => {
        it('returns empty non loading result with no providers', () => {
            scheduler().run(({ cold, expectObservable }) =>
                expectObservable(
                    getResultsFromProviders(
                        cold<Provider[]>('-a', { a: [] })
                    )
                ).toBe('-a', {
                    a: { isLoading: false, result: [] },
                })
            )
        })

        it('returns a result from a provider synchronously with raw values', () => {
            scheduler().run(({ cold, expectObservable }) =>
                expectObservable(
                    getResultsFromProviders(
                        cold<Provider[]>('-a', { a: [provide(1)] })
                    )
                ).toBe('-(lr)', {
                    l: { isLoading: true, result: [] },
                    r: { isLoading: false, result: [1] },
                })
            )
        })

        it('returns a result from a provider with observables', () => {
            scheduler().run(({ cold, expectObservable }) =>
                expectObservable(
                    getResultsFromProviders(
                        cold<Provider[]>('-a', { a: [provide(cold('--a', { a: 1 }))] })
                    )
                ).toBe('-l-r', {
                    l: { isLoading: true, result: [] },
                    r: { isLoading: false, result: [1] },
                })
            )
        })
    })

    describe('2 providers', () => {
        it("returns a synchronous result from both providers, but doesn't wait for the second to yield", () => {
            scheduler().run(({ cold, expectObservable }) =>
                expectObservable(
                    getResultsFromProviders(
                        cold<Provider[]>('-a', { a: [provide(1), provide(2)] })
                    )
                ).toBe('-(lr)', {
                    l: { isLoading: true, result: [1] },
                    r: { isLoading: false, result: [1, 2] },
                })
            )
        })

        it('returns isLoading: true first if providers return observables', () => {
            scheduler().run(({ cold, expectObservable }) =>
                expectObservable(
                    getResultsFromProviders(
                        cold<Provider[]>('-a', {
                            a: [provide(cold('-a', { a: 1 })), provide(cold('-a', { a: 2 }))],
                        })
                    )
                ).toBe('-lr', {
                    l: { isLoading: true, result: [] },
                    r: { isLoading: false, result: [1, 2] },
                })
            )
        })

        it('replaces errors from a provider with nulls', () => {
            scheduler().run(({ cold, expectObservable }) =>
                expectObservable(
                    getResultsFromProviders(
                        cold<Provider[]>('-a', {
                            a: [provide(cold('-a', { a: 1 })), provide(cold('-#', {}, new Error('boom!')))],
                        })
                    )
                ).toBe('-lr', {
                    l: { isLoading: true, result: [] },
                    r: { isLoading: false, result: [1, null] },
                })
            )
        })
    })

    describe('providers change over time', () => {
        it('requeries providers if they changed', () => {
            scheduler().run(({ cold, expectObservable }) =>
                expectObservable(
                    getResultsFromProviders(
                        cold<Provider[]>('-a-b', {
                            a: [provide(cold('-a', { a: 1 })), provide(cold('-a', { a: 2 }))],
                            b: [provide(cold('-a', { a: 3 }))],
                        })
                    )
                ).toBe('-abcd', {
                    a: { isLoading: true, result: [] },
                    b: { isLoading: false, result: [1, 2] },
                    c: { isLoading: true, result: [1, 2] },
                    d: { isLoading: false, result: [3] },
                })
            )
        })
    })
})
