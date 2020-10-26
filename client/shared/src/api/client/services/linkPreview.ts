import { isEqual } from 'lodash'
import { from, Observable } from 'rxjs'
import { catchError, defaultIfEmpty, distinctUntilChanged, map, switchMap } from 'rxjs/operators'
import * as sourcegraph from 'sourcegraph'
import { renderMarkdown } from '../../../util/markdown'
import { combineLatestOrDefault } from '../../../util/rxjs/combineLatestOrDefault'
import { property, isDefined } from '../../../util/types'
import { FeatureProviderRegistry } from './registry'
import { finallyReleaseProxy } from '../api/common'

interface MarkupContentPlainTextOnly extends Pick<sourcegraph.MarkupContent, 'value'> {
    kind: sourcegraph.MarkupKind.PlainText
}

/**
 * Represents one or more {@link sourcegraph.LinkPreview} values merged together.
 */
export interface LinkPreviewMerged {
    /** The content of the merged {@link sourcegraph.LinkPreview} values. */
    content: sourcegraph.MarkupContent[]

    /** The hover content of the merged {@link sourcegraph.LinkPreview} values. */
    hover: MarkupContentPlainTextOnly[]
}

/**
 * Provider registration options for a {@link sourcegraph.PreviewLinkProvider}.
 */
export interface LinkPreviewProviderRegistrationOptions {
    /**
     * @see {@link sourcegraph.content.registerLinkPreviewProvider#urlMatchPattern}
     */
    urlMatchPattern: string
}

export type ProvideLinkPreviewSignature = (url: string) => Observable<sourcegraph.LinkPreview | null | undefined>

/**
 * Manages {@link sourcegraph.LinkPreviewProvider} registrations.
 */
export class LinkPreviewProviderRegistry extends FeatureProviderRegistry<
    LinkPreviewProviderRegistrationOptions,
    ProvideLinkPreviewSignature
> {
    /**
     * Returns an observable that emits all providers' link previews whenever any of the
     * last-emitted set of providers emits link previews. If any provider emits an error, the error
     * is logged and the provider's result is omitted from the emission of the observable (the
     * observable does not emit the error).
     */
    public provideLinkPreview(url: string): Observable<LinkPreviewMerged | null> {
        return provideLinkPreview(this.observeProvidersForLink(url), url)
    }

    /**
     * Return an observable that emits all providers whose
     * {@link LinkPreviewProviderRegistrationOptions} match `url` upon subscription and whenever the
     * set of registered providers changes.
     */
    protected observeProvidersForLink(url: string): Observable<ProvideLinkPreviewSignature[]> {
        return this.entries.pipe(
            map(entries =>
                entries
                    .filter(entry => url.startsWith(entry.registrationOptions.urlMatchPattern))
                    .map(({ provider }) => provider)
            )
        )
    }
}

/**
 * Returns an observable that emits all providers' link previews whenever any of the last-emitted
 * set of providers emits link previews. If any provider emits an error, the error is logged and the
 * provider's result is omitted from the emission of the observable (the observable does not emit
 * the error).
 *
 * Most callers should use {@link LinkPreviewProviderRegistry#provideLinkPreview} method, which uses
 * the registered providers.
 */
export function provideLinkPreview(
    providers: Observable<ProvideLinkPreviewSignature[]>,
    url: string,
    logErrors = true
): Observable<LinkPreviewMerged | null> {
    return providers.pipe(
        switchMap(providers =>
            combineLatestOrDefault(
                providers.map(provider =>
                    from(
                        provider(url).pipe(
                            finallyReleaseProxy(),
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
                map(mergeLinkPreviews),
                defaultIfEmpty<LinkPreviewMerged | null>(null),
                distinctUntilChanged((a, b) => isEqual(a, b))
            )
        )
    )
}

function mergeLinkPreviews(values: (sourcegraph.LinkPreview | null | undefined)[]): LinkPreviewMerged | null {
    const nonemptyValues = values.filter(isDefined)
    const contentValues = nonemptyValues.filter(property('content', isDefined))
    const hoverValues = nonemptyValues.filter(property('hover', isDefined))
    if (hoverValues.length === 0 && contentValues.length === 0) {
        return null
    }
    return { content: contentValues.map(({ content }) => content), hover: hoverValues.map(({ hover }) => hover) }
}

/**
 * Renders an array of {@link sourcegraph.MarkupContent} to its HTML or plaintext contents. The HTML
 * contents are wrapped in an object `{ html: string }` so that callers can differentiate them from
 * plaintext contents.
 */
export function renderMarkupContents(contents: sourcegraph.MarkupContent[]): ({ html: string } | string)[] {
    return contents.map(({ kind, value }) => {
        if (kind === undefined || kind === 'markdown') {
            const html = renderMarkdown(value)
                .replace(/^<p>/, '')
                .replace(/<\/p>\s*$/, '') // remove <p> wrapper
            return { html }
        }
        return value // plaintext
    })
}
