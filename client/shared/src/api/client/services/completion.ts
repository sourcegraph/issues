import { isEqual } from 'lodash'
import { from, Observable } from 'rxjs'
import { catchError, defaultIfEmpty, distinctUntilChanged, map, switchMap } from 'rxjs/operators'
import { CompletionList } from 'sourcegraph'
import { combineLatestOrDefault } from '../../../util/rxjs/combineLatestOrDefault'
import { isDefined } from '../../../util/types'
import { TextDocumentPositionParameters } from '../../protocol'
import { DocumentFeatureProviderRegistry } from './registry'

export type ProvideCompletionItemSignature = (
    parameters: TextDocumentPositionParameters
) => Observable<CompletionList | null | undefined>

/** Provides hovers from all extensions. */
export class CompletionItemProviderRegistry extends DocumentFeatureProviderRegistry<ProvideCompletionItemSignature> {
    /**
     * Returns an observable that emits all providers' results whenever any of the last-emitted set
     * of providers emits hovers. If any provider emits an error, the error is logged and the
     * provider result is omitted from the emission of the observable (the observable does not emit
     * the error).
     */
    public getCompletionItems(parameters: TextDocumentPositionParameters): Observable<CompletionList | null> {
        return getCompletionItems(this.providersForDocument(parameters.textDocument), parameters)
    }
}

/**
 * Returns an observable that emits all providers' completion items whenever any of the last-emitted
 * set of providers emits completion items. If any provider emits an error, the error is logged and
 * the provider is omitted from the emission of the observable (the observable does not emit the
 * error).
 *
 * Most callers should use {@link CompletionItemsProviderRegistry#getCompletionItems}, which uses
 * the registered providers.
 */
export function getCompletionItems(
    providers: Observable<ProvideCompletionItemSignature[]>,
    parameters: TextDocumentPositionParameters,
    logErrors = true
): Observable<CompletionList | null> {
    return providers.pipe(
        switchMap(providers =>
            combineLatestOrDefault(
                providers.map(provider =>
                    from(
                        provider(parameters).pipe(
                            catchError(error => {
                                if (logErrors) {
                                    console.error(error)
                                }
                                return [null]
                            })
                        )
                    )
                )
            ).pipe(
                map(mergeCompletionLists),
                defaultIfEmpty<CompletionList | null>(null),
                distinctUntilChanged((a, b) => isEqual(a, b))
            )
        )
    )
}

function mergeCompletionLists(values: (CompletionList | null | undefined)[]): CompletionList | null {
    const items = values.filter(isDefined).flatMap(({ items }) => items)

    return items.length > 0 ? { items } : null
}
