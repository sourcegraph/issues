import { MarkupKind } from '@sourcegraph/extension-api-classes'
import { Observable, of, throwError } from 'rxjs'
import { TestScheduler } from 'rxjs/testing'
import * as sourcegraph from 'sourcegraph'
import {
    LinkPreviewMerged,
    LinkPreviewProviderRegistrationOptions,
    LinkPreviewProviderRegistry,
    provideLinkPreview,
    ProvideLinkPreviewSignature,
    renderMarkupContents,
} from './linkPreview'
import { Entry } from './registry'

const scheduler = (): TestScheduler => new TestScheduler((a, b) => expect(a).toEqual(b))

const FIXTURE_LINK_PREVIEW: sourcegraph.LinkPreview = {
    content: { value: 'x', kind: MarkupKind.PlainText },
    hover: { value: 'y', kind: MarkupKind.PlainText },
}
const FIXTURE_LINK_PREVIEW_MERGED: LinkPreviewMerged = {
    content: [{ value: 'x', kind: MarkupKind.PlainText }],
    hover: [{ value: 'y', kind: MarkupKind.PlainText }],
}

describe('LinkPreviewProviderRegistry', () => {
    /**
     * Allow overriding {@link LinkPreviewProviderRegistry#entries} for tests.
     */
    class TestLinkPreviewProviderRegistry extends LinkPreviewProviderRegistry {
        constructor(
            entries?: Observable<Entry<LinkPreviewProviderRegistrationOptions, ProvideLinkPreviewSignature>[]>
        ) {
            super()
            if (entries) {
                entries.subscribe(entries => this.entries.next(entries))
            }
        }

        /** Make public for tests. */
        public observeProvidersForLink(url: string): Observable<ProvideLinkPreviewSignature[]> {
            return super.observeProvidersForLink(url)
        }
    }

    describe('observeProvidersForLink', () => {
        const PROVIDER: ProvideLinkPreviewSignature = () =>
            of<sourcegraph.LinkPreview>({ hover: { value: 'x', kind: MarkupKind.PlainText } })

        test('prefix match', () => {
            scheduler().run(({ cold, expectObservable }) => {
                const registry = new TestLinkPreviewProviderRegistry(
                    cold<Entry<LinkPreviewProviderRegistrationOptions, ProvideLinkPreviewSignature>[]>('a', {
                        a: [{ provider: PROVIDER, registrationOptions: { urlMatchPattern: 'http://example.com' } }],
                    })
                )
                expectObservable(registry.observeProvidersForLink('http://example.com')).toBe('a', {
                    a: [PROVIDER],
                })
            })
        })

        test('no prefix match', () => {
            scheduler().run(({ cold, expectObservable }) => {
                const registry = new TestLinkPreviewProviderRegistry(
                    cold<Entry<LinkPreviewProviderRegistrationOptions, ProvideLinkPreviewSignature>[]>('a', {
                        a: [{ provider: PROVIDER, registrationOptions: { urlMatchPattern: 'http://example.com' } }],
                    })
                )
                expectObservable(registry.observeProvidersForLink('http://other.example.com/foo')).toBe('a', {
                    a: [],
                })
            })
        })
    })
})

describe('provideLinkPreview', () => {
    describe('0 providers', () => {
        test('returns null', () =>
            scheduler().run(({ cold, expectObservable }) =>
                expectObservable(
                    provideLinkPreview(
                        cold<ProvideLinkPreviewSignature[]>('-a-|', { a: [] }),
                        'http://example.com/foo'
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
                    provideLinkPreview(
                        cold<ProvideLinkPreviewSignature[]>('-a-|', { a: [() => of(null)] }),
                        'http://example.com/foo'
                    )
                ).toBe('-a-|', {
                    a: null,
                })
            ))

        test('returns result from provider', () =>
            scheduler().run(({ cold, expectObservable }) =>
                expectObservable(
                    provideLinkPreview(
                        cold<ProvideLinkPreviewSignature[]>('-a-|', {
                            a: [() => of(FIXTURE_LINK_PREVIEW)],
                        }),
                        'http://example.com/foo'
                    )
                ).toBe('-a-|', {
                    a: FIXTURE_LINK_PREVIEW_MERGED,
                })
            ))
    })

    test('errors do not propagate', () =>
        scheduler().run(({ cold, expectObservable }) =>
            expectObservable(
                provideLinkPreview(
                    cold<ProvideLinkPreviewSignature[]>('-a-|', {
                        a: [() => of(FIXTURE_LINK_PREVIEW), () => throwError(new Error('x'))],
                    }),
                    'http://example.com/foo',
                    false
                )
            ).toBe('-a-|', {
                a: FIXTURE_LINK_PREVIEW_MERGED,
            })
        ))

    describe('2 providers', () => {
        test('returns null result if both providers return null', () =>
            scheduler().run(({ cold, expectObservable }) =>
                expectObservable(
                    provideLinkPreview(
                        cold<ProvideLinkPreviewSignature[]>('-a-|', {
                            a: [() => of(null), () => of(null)],
                        }),
                        'http://example.com/foo'
                    )
                ).toBe('-a-|', {
                    a: null,
                })
            ))

        test('omits null result from 1 provider', () =>
            scheduler().run(({ cold, expectObservable }) =>
                expectObservable(
                    provideLinkPreview(
                        cold<ProvideLinkPreviewSignature[]>('-a-|', {
                            a: [() => of(FIXTURE_LINK_PREVIEW), () => of(null)],
                        }),
                        'http://example.com/foo'
                    )
                ).toBe('-a-|', {
                    a: FIXTURE_LINK_PREVIEW_MERGED,
                })
            ))

        test('merges results from providers', () =>
            scheduler().run(({ cold, expectObservable }) =>
                expectObservable(
                    provideLinkPreview(
                        cold<ProvideLinkPreviewSignature[]>('-a-|', {
                            a: [() => of(FIXTURE_LINK_PREVIEW), () => of(FIXTURE_LINK_PREVIEW)],
                        }),
                        'http://example.com/foo'
                    )
                ).toBe('-a-|', {
                    a: {
                        content: [FIXTURE_LINK_PREVIEW.content, FIXTURE_LINK_PREVIEW.content],
                        hover: [FIXTURE_LINK_PREVIEW.hover, FIXTURE_LINK_PREVIEW.hover],
                    },
                })
            ))
    })

    describe('multiple emissions', () => {
        test('returns stream of results', () =>
            scheduler().run(({ cold, expectObservable }) =>
                expectObservable(
                    provideLinkPreview(
                        cold<ProvideLinkPreviewSignature[]>('-a-b-|', {
                            a: [() => of(FIXTURE_LINK_PREVIEW)],
                            b: [() => of(null)],
                        }),
                        'http://example.com/foo'
                    )
                ).toBe('-a-b-|', {
                    a: FIXTURE_LINK_PREVIEW_MERGED,
                    b: null,
                })
            ))
    })
})

describe('renderMarkupContents', () => {
    test('renders', () =>
        expect(
            renderMarkupContents([
                { value: '*a*' },
                { kind: MarkupKind.PlainText, value: 'b' },
                { kind: MarkupKind.Markdown, value: '*c*' },
            ])
        ).toEqual([{ html: '<em>a</em>' }, 'b', { html: '<em>c</em>' }]))
})
