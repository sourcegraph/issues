import { of, throwError } from 'rxjs'
import { TestScheduler } from 'rxjs/testing'
import * as sourcegraph from 'sourcegraph'
import { getCompletionItems, ProvideCompletionItemSignature } from './completion'
import { FIXTURE } from './registry.test'

const scheduler = (): TestScheduler => new TestScheduler((a, b) => expect(a).toEqual(b))

const FIXTURE_COMPLETION_LIST: sourcegraph.CompletionList = {
    items: [{ label: 'x' }],
}

describe('getCompletionItems', () => {
    describe('0 providers', () => {
        test('returns null', () =>
            scheduler().run(({ cold, expectObservable }) =>
                expectObservable(
                    getCompletionItems(
                        cold<ProvideCompletionItemSignature[]>('-a-|', { a: [] }),
                        FIXTURE.TextDocumentPositionParams
                    )
                ).toBe('-a-|', {
                    a: null,
                })
            ))
    })

    describe('1 provider', () => {
        test('returns null result from provider', () =>
            scheduler().run(({ cold, expectObservable }) =>
                expectObservable(
                    getCompletionItems(
                        cold<ProvideCompletionItemSignature[]>('-a-|', { a: [() => of(null)] }),
                        FIXTURE.TextDocumentPositionParams
                    )
                ).toBe('-a-|', {
                    a: null,
                })
            ))

        test('returns result from provider', () =>
            scheduler().run(({ cold, expectObservable }) =>
                expectObservable(
                    getCompletionItems(
                        cold<ProvideCompletionItemSignature[]>('-a-|', {
                            a: [() => of(FIXTURE_COMPLETION_LIST)],
                        }),
                        FIXTURE.TextDocumentPositionParams
                    )
                ).toBe('-a-|', {
                    a: FIXTURE_COMPLETION_LIST,
                })
            ))
    })

    test('errors do not propagate', () =>
        scheduler().run(({ cold, expectObservable }) =>
            expectObservable(
                getCompletionItems(
                    cold<ProvideCompletionItemSignature[]>('-a-|', {
                        a: [() => of(FIXTURE_COMPLETION_LIST), () => throwError(new Error('x'))],
                    }),
                    FIXTURE.TextDocumentPositionParams,
                    false
                )
            ).toBe('-a-|', {
                a: FIXTURE_COMPLETION_LIST,
            })
        ))

    describe('2 providers', () => {
        test('returns null result if both providers return null', () =>
            scheduler().run(({ cold, expectObservable }) =>
                expectObservable(
                    getCompletionItems(
                        cold<ProvideCompletionItemSignature[]>('-a-|', {
                            a: [() => of(null), () => of(null)],
                        }),
                        FIXTURE.TextDocumentPositionParams
                    )
                ).toBe('-a-|', {
                    a: null,
                })
            ))

        test('omits null result from 1 provider', () =>
            scheduler().run(({ cold, expectObservable }) =>
                expectObservable(
                    getCompletionItems(
                        cold<ProvideCompletionItemSignature[]>('-a-|', {
                            a: [() => of(FIXTURE_COMPLETION_LIST), () => of(null)],
                        }),
                        FIXTURE.TextDocumentPositionParams
                    )
                ).toBe('-a-|', {
                    a: FIXTURE_COMPLETION_LIST,
                })
            ))

        test('merges results from providers', () =>
            scheduler().run(({ cold, expectObservable }) =>
                expectObservable(
                    getCompletionItems(
                        cold<ProvideCompletionItemSignature[]>('-a-|', {
                            a: [() => of(FIXTURE_COMPLETION_LIST), () => of(FIXTURE_COMPLETION_LIST)],
                        }),
                        FIXTURE.TextDocumentPositionParams
                    )
                ).toBe('-a-|', {
                    a: {
                        items: [...FIXTURE_COMPLETION_LIST.items, ...FIXTURE_COMPLETION_LIST.items],
                    },
                })
            ))
    })

    describe('multiple emissions', () => {
        test('returns stream of results', () =>
            scheduler().run(({ cold, expectObservable }) =>
                expectObservable(
                    getCompletionItems(
                        cold<ProvideCompletionItemSignature[]>('-a-b-|', {
                            a: [() => of(FIXTURE_COMPLETION_LIST)],
                            b: [() => of(null)],
                        }),
                        FIXTURE.TextDocumentPositionParams
                    )
                ).toBe('-a-b-|', {
                    a: FIXTURE_COMPLETION_LIST,
                    b: null,
                })
            ))
    })
})
