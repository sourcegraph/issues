import { MarkupKind } from '@sourcegraph/extension-api-classes'
import { take } from 'rxjs/operators'
import { integrationTestContext } from './testHelpers'

describe('content (integration)', () => {
    test('registers a link preview provider', async () => {
        const { services, extensionAPI } = await integrationTestContext()

        // Register the provider and call it.
        extensionAPI.content.registerLinkPreviewProvider('http://example.com', {
            provideLinkPreview: url => ({
                content: { value: `xyz ${url.href}`, kind: MarkupKind.PlainText },
            }),
        })
        await extensionAPI.internal.sync()
        expect(
            await services.linkPreviews.provideLinkPreview('http://example.com/foo').pipe(take(1)).toPromise()
        ).toEqual({
            content: [{ value: 'xyz http://example.com/foo', kind: MarkupKind.PlainText }],
            hover: [],
        })
    })

    test('unregisters a link preview provider', async () => {
        const { services, extensionAPI } = await integrationTestContext()

        // Register the provider and call it.
        const subscription = extensionAPI.content.registerLinkPreviewProvider('http://example.com', {
            provideLinkPreview: () => ({ content: { value: 'bar', kind: MarkupKind.PlainText } }),
        })
        await extensionAPI.internal.sync()

        // Unregister the provider and ensure it's removed.
        subscription.unsubscribe()
        await extensionAPI.internal.sync()
        expect(
            await services.linkPreviews.provideLinkPreview('http://example.com/foo').pipe(take(1)).toPromise()
        ).toEqual(null)
    })

    test('supports multiple link preview providers', async () => {
        const { services, extensionAPI } = await integrationTestContext()

        // Register the provider and call it
        extensionAPI.content.registerLinkPreviewProvider('http://example.com', {
            provideLinkPreview: () => ({ content: { value: 'qux', kind: MarkupKind.PlainText } }),
        })
        extensionAPI.content.registerLinkPreviewProvider('http://example.com', {
            provideLinkPreview: () => ({ content: { value: 'zip', kind: MarkupKind.PlainText } }),
        })
        await extensionAPI.internal.sync()
        expect(
            await services.linkPreviews.provideLinkPreview('http://example.com/foo').pipe(take(1)).toPromise()
        ).toEqual({
            content: [
                { value: 'qux', kind: MarkupKind.PlainText },
                { value: 'zip', kind: MarkupKind.PlainText },
            ],
            hover: [],
        })
    })
})
