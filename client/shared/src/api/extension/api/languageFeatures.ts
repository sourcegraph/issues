/* eslint-disable no-sync */
import * as comlink from 'comlink'
import * as extensionApiTypes from '@sourcegraph/extension-api-types'
import { Unsubscribable } from 'rxjs'
import { CompletionItemProvider, DocumentSelector, Location, LocationProvider, ReferenceProvider } from 'sourcegraph'
import { ClientLanguageFeaturesAPI } from '../../client/api/languageFeatures'
import { ReferenceParameters, TextDocumentPositionParameters } from '../../protocol'
import { syncSubscription } from '../../util'
import { toProxyableSubscribable } from './common'
import { ExtensionDocuments } from './documents'
import { fromLocation, toPosition, fromDocumentSelector } from './types'

/** @internal */
export class ExtensionLanguageFeatures {
    constructor(private proxy: comlink.Remote<ClientLanguageFeaturesAPI>, private documents: ExtensionDocuments) {}

    public registerReferenceProvider(selector: DocumentSelector, provider: ReferenceProvider): Unsubscribable {
        const providerFunction: comlink.Local<
            Parameters<ClientLanguageFeaturesAPI['$registerReferenceProvider']>[1]
        > = comlink.proxy(async ({ textDocument, position, context }: ReferenceParameters) =>
            toProxyableSubscribable(
                provider.provideReferences(
                    await this.documents.getSync(textDocument.uri),
                    toPosition(position),
                    context
                ),
                toLocations
            )
        )
        return syncSubscription(this.proxy.$registerReferenceProvider(fromDocumentSelector(selector), providerFunction))
    }

    public registerLocationProvider(
        idString: string,
        selector: DocumentSelector,
        provider: LocationProvider
    ): Unsubscribable {
        const providerFunction: comlink.Local<
            Parameters<ClientLanguageFeaturesAPI['$registerLocationProvider']>[2]
        > = comlink.proxy(async ({ textDocument, position }: TextDocumentPositionParameters) =>
            toProxyableSubscribable(
                provider.provideLocations(await this.documents.getSync(textDocument.uri), toPosition(position)),
                toLocations
            )
        )
        return syncSubscription(
            this.proxy.$registerLocationProvider(
                idString,
                fromDocumentSelector(selector),
                comlink.proxy(providerFunction)
            )
        )
    }

    public registerCompletionItemProvider(
        selector: DocumentSelector,
        provider: CompletionItemProvider
    ): Unsubscribable {
        const providerFunction: comlink.Local<
            Parameters<ClientLanguageFeaturesAPI['$registerCompletionItemProvider']>[1]
        > = comlink.proxy(async ({ textDocument, position }: TextDocumentPositionParameters) =>
            toProxyableSubscribable(
                provider.provideCompletionItems(await this.documents.getSync(textDocument.uri), toPosition(position)),
                items => items
            )
        )
        return syncSubscription(
            this.proxy.$registerCompletionItemProvider(fromDocumentSelector(selector), providerFunction)
        )
    }
}

function toLocations(result: Location[] | Location | null | undefined): extensionApiTypes.Location[] {
    return result ? (Array.isArray(result) ? result : [result]).map(location => fromLocation(location)) : []
}
