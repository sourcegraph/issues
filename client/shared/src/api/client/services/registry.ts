import { BehaviorSubject, Observable, Subscription } from 'rxjs'
import { map } from 'rxjs/operators'
import { getModeFromPath } from '../../../languages'
import { parseRepoURI } from '../../../util/url'
import { TextDocumentRegistrationOptions } from '../../protocol'
import { match, TextDocumentIdentifier } from '../types/textDocument'

/** A registry entry for a registered provider. */
export interface Entry<O, P> {
    registrationOptions: O
    provider: P
}

/** Base class for provider registries for features. */
export abstract class FeatureProviderRegistry<O, P> {
    protected entries = new BehaviorSubject<Entry<O, P>[]>([])

    public registerProvider(registrationOptions: O, provider: P): Subscription {
        return this.registerProviders([{ registrationOptions, provider }])
    }

    /**
     * Bulk-register providers. This is useful when registering an initial set of providers. Registering them all
     * at once means there is only one emission from {@link FeatureProviderRegistry#entries} (instead of one per
     * entry).
     */
    public registerProviders(entries: Entry<O, P>[]): Subscription {
        this.entries.next([...this.entries.value, ...entries])
        return new Subscription(() => {
            this.entries.next([...this.entries.value.filter(entry => !entries.includes(entry))])
        })
    }

    /** All providers, emitted whenever the set of registered providers changed. */
    public readonly providers: Observable<P[]> = this.entries.pipe(
        map(entries => entries.map(({ provider }) => provider))
    )
}

/**
 * A registry for providers that provide features within a document. Use this class instead of
 * {@link FeatureProviderRegistry} when all calls to the provider are scoped to a document.
 *
 * For example, hovers are scoped to a document (i.e., the document URI is one of the required arguments passed to
 * the hover provider), so this class is used for the hover provider registry.
 */
export abstract class DocumentFeatureProviderRegistry<
    P,
    O extends TextDocumentRegistrationOptions = TextDocumentRegistrationOptions
> extends FeatureProviderRegistry<O, P> {
    /**
     * @param document The text document to find applicable providers for.
     * @param filter An optional function to filter providers based on their registration options.
     * @returns an observable of the set of registered providers that apply to the document. The observable emits
     * initially and whenever the set changes (due to a provider being registered or unregistered).
     */
    public providersForDocument(
        document: TextDocumentIdentifier,
        filter?: (registrationOptions: O) => boolean
    ): Observable<P[]> {
        return this.entries.pipe(
            map(entries =>
                entries
                    .filter(
                        ({ registrationOptions }) =>
                            (filter ? filter(registrationOptions) : true) &&
                            match(registrationOptions.documentSelector, {
                                uri: document.uri,
                                languageId: getModeFromPath(parseRepoURI(document.uri).filePath || ''),
                            })
                    )
                    .map(({ provider }) => provider)
            )
        )
    }
}
