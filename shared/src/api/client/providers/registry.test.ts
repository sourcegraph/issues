import * as assert from 'assert'
import { Subscription } from 'rxjs'
import { TextDocumentPositionParams, TextDocumentRegistrationOptions } from '../../protocol'
import { FeatureProviderRegistry as AbstractFeatureProviderRegistry } from './registry'

/** Useful test fixtures. */
export const FIXTURE = {
    TextDocumentPositionParams: {
        position: { line: 1, character: 2 },
        textDocument: { uri: 'file:///f' },
    } as TextDocumentPositionParams,

    PartialEntry: {
        registrationOptions: {
            documentSelector: ['*'],
        } as TextDocumentRegistrationOptions,
    },
}

class FeatureProviderRegistry extends AbstractFeatureProviderRegistry<TextDocumentRegistrationOptions, {}> {
    /**
     * The current set of providers. Used by callers that do not need to react to providers being
     * registered or unregistered.
     *
     * NOTE: You should usually use the providers property on this class, not providersSnapshot,
     * even when you think you don't need live-updating results. Providers are registered
     * asynchronously after the client connects (or reconnects) to the server. So, the providers
     * list might be empty at the instant you need the results (because the client was just
     * instantiated and is waiting on a network roundtrip before it registers providers, or because
     * there was a temporary network error and the client is reestablishing the connection). By
     * using the providers property, the consumer will get the results it (probably) expects when
     * the client connects and registers the providers.
     */
    public get providersSnapshot(): {}[] {
        return this.entries.value.map(({ provider }) => provider)
    }
}

describe('FeatureProviderRegistry', () => {
    it('is initially empty', () => {
        assert.deepStrictEqual(new FeatureProviderRegistry().providersSnapshot, [])
    })

    it('accepts initial providers', () => {
        const initialEntries = [
            {
                ...FIXTURE.PartialEntry,
                provider: () => ({}),
            },
        ]
        assert.deepStrictEqual(
            new FeatureProviderRegistry(initialEntries).providersSnapshot,
            initialEntries.map(({ provider }) => provider)
        )
    })

    it('registers and unregisters providers', () => {
        const subscriptions = new Subscription()
        const registry = new FeatureProviderRegistry()
        const provider1 = () => ({})
        const provider2 = () => ({})

        const unregister1 = subscriptions.add(
            registry.registerProvider(FIXTURE.PartialEntry.registrationOptions, provider1)
        )
        assert.deepStrictEqual(registry.providersSnapshot, [provider1])

        const unregister2 = subscriptions.add(
            registry.registerProvider(FIXTURE.PartialEntry.registrationOptions, provider2)
        )
        assert.deepStrictEqual(registry.providersSnapshot, [provider1, provider2])

        unregister1.unsubscribe()
        assert.deepStrictEqual(registry.providersSnapshot, [provider2])

        unregister2.unsubscribe()
        assert.deepStrictEqual(registry.providersSnapshot, [])
    })
})
