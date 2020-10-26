import { Location } from '@sourcegraph/extension-api-types'
import { Observable, of, concat } from 'rxjs'
import { catchError, map, switchMap, defaultIfEmpty } from 'rxjs/operators'
import { combineLatestOrDefault } from '../../../util/rxjs/combineLatestOrDefault'
import { TextDocumentPositionParameters, TextDocumentRegistrationOptions } from '../../protocol'
import { match, TextDocumentIdentifier } from '../types/textDocument'
import { CodeEditorWithPartialModel } from './viewerService'
import { DocumentFeatureProviderRegistry } from './registry'
import { MaybeLoadingResult, LOADING } from '@sourcegraph/codeintellify'
import { finallyReleaseProxy } from '../api/common'

/**
 * Function signature for retrieving related locations given a location (e.g., definition, implementation, and type
 * definition).
 */
export type ProvideTextDocumentLocationSignature<
    P extends TextDocumentPositionParameters = TextDocumentPositionParameters,
    L extends Location = Location
> = (parameters: P) => Observable<L[] | null>

/**
 * Provides location results from matching registered providers for definition, implementation, and type definition
 * requests.
 */
export class TextDocumentLocationProviderRegistry<
    P extends TextDocumentPositionParameters = TextDocumentPositionParameters,
    L extends Location = Location
> extends DocumentFeatureProviderRegistry<ProvideTextDocumentLocationSignature<P, L>> {
    /**
     * Returns an observable that, initially and upon the registered provider set changing, emits an
     * observable of the combined location results from all providers.
     *
     * Using a higher-order observable here lets the caller display a loading indicator when some
     * providers have not yet completed (i.e., when the inner observable is not yet completed). The
     * outer observable never completes because providers may be registered and unregistered at any
     * time.
     */
    public getLocations(parameters: P): Observable<MaybeLoadingResult<L[]>> {
        return getLocationsFromProviders(this.providersForDocument(parameters.textDocument), parameters)
    }

    /**
     * Reports whether there are any location providers registered for the active text document.
     * This can be used, for example, to selectively show a "Find references" button if there are
     * any reference providers registered.
     */
    public hasProvidersForActiveTextDocument(
        activeEditor: CodeEditorWithPartialModel | undefined
    ): Observable<boolean> {
        if (!activeEditor) {
            return of(false)
        }
        return this.entries.pipe(
            map(
                entries =>
                    entries.filter(({ registrationOptions }) =>
                        match(registrationOptions.documentSelector, {
                            uri: activeEditor.resource,
                            languageId: activeEditor.model.languageId,
                        })
                    ).length > 0
            )
        )
    }
}

/**
 * Registration options for a text document provider that has an ID (such as {@link sourcegraph.LocationProvider}).
 */
export interface TextDocumentProviderIDRegistrationOptions extends TextDocumentRegistrationOptions {
    /**
     * The identifier of the provider, used to distinguish it among other providers.
     *
     * This corresponds to, e.g., the `id` parameter in {@link sourcegraph.languages.registerLocationProvider}.
     */
    id: string
}

/**
 * The registry for text document location providers with a distinguishing ID (i.e., registered using
 * {@link TextDocumentProviderIDRegistrationOptions}).
 *
 * @template P The param type of the text document location signature provider.
 * @template L The result type of the text document location signature provider.
 */
export class TextDocumentLocationProviderIDRegistry extends DocumentFeatureProviderRegistry<
    ProvideTextDocumentLocationSignature<TextDocumentPositionParameters, Location>,
    TextDocumentProviderIDRegistrationOptions
> {
    /**
     * @param id The provider ID.
     * @returns an observable of the set of registered providers that apply to the document with the given ID.
     * (Usually there is at most 1 such provider.) The observable emits initially and whenever the set changes (due
     * to a provider being registered or unregistered).
     */
    public providersForDocumentWithID(
        id: string,
        document: TextDocumentIdentifier
    ): Observable<ProvideTextDocumentLocationSignature<TextDocumentPositionParameters, Location>[]> {
        return this.providersForDocument(document, registrationOptions => registrationOptions.id === id)
    }

    /**
     * Gets locations from the provider with the given ID (i.e., the `id` parameter to
     * {@link sourcegraph.languageFeatures.registerLocationProvider}). Returns an observable that,
     * initially and upon the provider changing, emits an observable of the provider's location
     * results.
     *
     * Using a higher-order observable here lets the caller display a loading indicator when the
     * inner observable has not yet completed. The outer observable never completes because
     * providers may be registered and unregistered at any time.
     *
     * @param id The provider ID.
     */
    public getLocations(
        id: string,
        parameters: TextDocumentPositionParameters
    ): Observable<MaybeLoadingResult<Location[]>> {
        return getLocationsFromProviders(this.providersForDocumentWithID(id, parameters.textDocument), parameters)
    }
}

/**
 * Returns the combined results of invoking multiple location providers and whether any of them are loading.
 *
 * @internal
 */
export function getLocationsFromProviders<
    P extends TextDocumentPositionParameters = TextDocumentPositionParameters,
    L extends Location = Location
>(
    providers: Observable<ProvideTextDocumentLocationSignature<P, L>[]>,
    parameters: P,
    logErrors = true
): Observable<MaybeLoadingResult<L[]>> {
    return providers.pipe(
        switchMap(providers =>
            combineLatestOrDefault(
                providers.map(provider =>
                    concat(
                        [LOADING],
                        provider(parameters).pipe(
                            finallyReleaseProxy(),
                            defaultIfEmpty<typeof LOADING | L[] | null>([]),
                            catchError(error => {
                                if (logErrors) {
                                    console.error('Location provider errored:', error)
                                }
                                return [null]
                            })
                        )
                    )
                )
            ).pipe(
                map(locationsFromProviders => ({
                    isLoading: locationsFromProviders.some(locations => locations === LOADING),
                    result: locationsFromProviders.filter<L[]>(Array.isArray).flat(),
                }))
            )
        )
    )
}
