import { Location } from '@sourcegraph/extension-api-types'
import React from 'react'
import { Observable } from 'rxjs'
import { catchError, map, switchMap } from 'rxjs/operators'
import * as sourcegraph from 'sourcegraph'
import { combineLatestOrDefault } from '../../../util/rxjs/combineLatestOrDefault'
import { ContributableViewContainer } from '../../protocol'
import { Entry, FeatureProviderRegistry } from './registry'
import { MaybeLoadingResult } from '@sourcegraph/codeintellify'
import { isDefined } from '../../../util/types'

export interface PanelViewProviderRegistrationOptions {
    id: string
    container: ContributableViewContainer
}

export interface PanelViewWithComponent extends Pick<sourcegraph.PanelView, 'title' | 'content' | 'priority'> {
    /**
     * The location provider whose results to render in the panel view.
     */
    locationProvider?: Observable<MaybeLoadingResult<Location[]>>

    /**
     * The React element to render in the panel view.
     */
    reactElement?: React.ReactFragment
}

export type ProvidePanelViewSignature = Observable<PanelViewWithComponent | null>

/** Provides panel views from all extensions. */
export class PanelViewProviderRegistry extends FeatureProviderRegistry<
    PanelViewProviderRegistrationOptions,
    ProvidePanelViewSignature
> {
    /**
     * Returns an observable that emits the specified panel view whenever it or the set of
     * registered panel view providers changes. If the provider emits an error, the returned
     * observable also emits an error (and completes).
     */
    public getPanelView(
        id: string
    ): Observable<(PanelViewWithComponent & PanelViewProviderRegistrationOptions) | null> {
        return getPanelView(this.entries, id)
    }

    /**
     * Returns an observable that emits all panel views whenever the set of registered panel view
     * providers or their properties change. If any provider emits an error, the error is logged and
     * the provider is omitted from the emission of the observable (the observable does not emit the
     * error).
     */
    public getPanelViews(
        container: ContributableViewContainer
    ): Observable<(PanelViewWithComponent & PanelViewProviderRegistrationOptions)[]> {
        return getPanelViews(this.entries, container)
    }
}

/**
 * Exported for testing only. Most callers should use {@link ViewProviderRegistry#getPanelView},
 * which uses the registered panel view providers.
 *
 * @internal
 */
export function getPanelView(
    entries: Observable<Entry<PanelViewProviderRegistrationOptions, Observable<PanelViewWithComponent | null>>[]>,
    id: string
): Observable<(PanelViewWithComponent & PanelViewProviderRegistrationOptions) | null> {
    return entries.pipe(
        map(entries => entries.find(entry => entry.registrationOptions.id === id)),
        switchMap(entry => (entry ? addRegistrationOptions(entry) : [null]))
    )
}

/**
 * Exported for testing only. Most callers should use {@link ViewProviderRegistry#getPanelViews},
 * which uses the registered panel view providers.
 *
 * @internal
 */
export function getPanelViews(
    entries: Observable<Entry<PanelViewProviderRegistrationOptions, Observable<PanelViewWithComponent | null>>[]>,
    container: ContributableViewContainer,
    logErrors = true
): Observable<(PanelViewWithComponent & PanelViewProviderRegistrationOptions)[]> {
    return entries.pipe(
        switchMap(entries =>
            combineLatestOrDefault(
                entries
                    .filter(entry => entry.registrationOptions.container === container)
                    .map(entry =>
                        addRegistrationOptions(entry).pipe(
                            catchError(error => {
                                if (logErrors) {
                                    console.error(error)
                                }
                                return [null]
                            })
                        )
                    )
            ).pipe(map(entries => entries.filter(isDefined)))
        )
    )
}

function addRegistrationOptions(
    entry: Entry<PanelViewProviderRegistrationOptions, Observable<PanelViewWithComponent | null>>
): Observable<(PanelViewWithComponent & PanelViewProviderRegistrationOptions) | null> {
    return entry.provider.pipe(map(view => view && { ...view, ...entry.registrationOptions }))
}
