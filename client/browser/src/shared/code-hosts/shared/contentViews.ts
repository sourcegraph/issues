import { asyncScheduler, merge, Observable, of, Subject, Subscription, Unsubscribable } from 'rxjs'
import { distinctUntilChanged, map, mapTo, mergeMap, observeOn, tap, throttleTime } from 'rxjs/operators'
import { LinkPreviewProviderRegistry } from '../../../../../shared/src/api/client/services/linkPreview'
import { applyLinkPreview } from '../../../../../shared/src/components/linkPreviews/linkPreviews'
import { ExtensionsControllerProps } from '../../../../../shared/src/extensions/controller'
import { MutationRecordLike, observeMutations } from '../../util/dom'
import { CodeHost } from './codeHost'
import { trackViews } from './views'

/**
 * Defines a content view that is present on a page and exposes operations for manipulating it.
 */
export interface ContentView {
    /** The content view HTML element. */
    element: HTMLElement
}

/**
 * Handles added and removed content views according to the {@link CodeHost} configuration.
 */
export function handleContentViews(
    mutations: Observable<MutationRecordLike[]>,
    {
        extensionsController,
    }:
        | ExtensionsControllerProps
        | {
              extensionsController: {
                  services: { linkPreviews: Pick<LinkPreviewProviderRegistry, 'provideLinkPreview'> }
              }
          },
    {
        contentViewResolvers,
        linkPreviewContentClass,
        setElementTooltip,
    }: Pick<CodeHost, 'contentViewResolvers' | 'linkPreviewContentClass' | 'setElementTooltip'>
): Unsubscribable {
    /** A stream of added or removed content views. */
    const contentViews = mutations.pipe(trackViews<ContentView>(contentViewResolvers || []), observeOn(asyncScheduler))

    /** Pause DOM MutationObserver while we are making changes to avoid duplicating work. */
    const pauseMutationObserver = new Subject<boolean>()

    /**
     * Map from content view element to linkPreview subscriptions
     *
     * These subscriptions are maintained separately from `contentViewEvent.subscription`,
     * as they need to be unsubscribed when a content view is updated.
     */
    const linkPreviewSubscriptions = new Map<HTMLElement, Subscription>()

    return contentViews
        .pipe(
            mergeMap(contentViewEvent =>
                merge(
                    of(contentViewEvent).pipe(
                        tap(() => {
                            console.log('Content view added', { contentViewEvent })
                            linkPreviewSubscriptions.set(contentViewEvent.element, new Subscription())
                            contentViewEvent.subscriptions.add(() => {
                                console.log('Content view removed', { contentViewEvent })

                                // Clean up current link preview subscriptions when the content view is removed
                                const subscriptions = linkPreviewSubscriptions.get(contentViewEvent.element)
                                if (!subscriptions) {
                                    throw new Error('No linkPreview subscriptions')
                                }
                                subscriptions.unsubscribe()
                            })
                        })
                    ),

                    /**
                     * Observe updates to the element. Only emit on mutations that actually
                     * change the innerHTML so that our own {@link applyLinkPreview} updates
                     * don't trigger needless work. It is not sufficient to suppress observing
                     * these changes using {@link MutationObserver#disconnect} because that does
                     * not actually seem to suppress mutation notifications in tests when using
                     * jsdom.
                     */
                    observeMutations(contentViewEvent.element, { childList: true }, pauseMutationObserver).pipe(
                        observeOn(asyncScheduler),
                        map(() => contentViewEvent.element.innerHTML),
                        distinctUntilChanged(),
                        tap(() => console.log('Content view updated', { contentViewEvent })),
                        mapTo(contentViewEvent),
                        throttleTime(2000, undefined, { leading: true, trailing: true }) // reduce the harm from an infinite loop bug
                    )
                )
            ),
            tap(({ element }) => {
                // Reset link preview subscriptions
                let subscriptions = linkPreviewSubscriptions.get(element)
                if (!subscriptions) {
                    throw new Error('No linkPreview subscriptions')
                }
                subscriptions.unsubscribe()
                subscriptions = new Subscription()
                linkPreviewSubscriptions.set(element, subscriptions)

                // Add link preview content.
                for (const link of element.querySelectorAll<HTMLAnchorElement>('a[href]')) {
                    subscriptions.add(
                        extensionsController.services.linkPreviews
                            .provideLinkPreview(link.href)
                            // The nested subscribe cannot be replaced with a switchMap()
                            // because we are managing a stateful Map. The subscription is
                            // managed correctly.
                            //
                            // eslint-disable-next-line rxjs/no-nested-subscribe
                            .subscribe(linkPreview => {
                                try {
                                    pauseMutationObserver.next(true) // ignore DOM mutations we make
                                    applyLinkPreview({ setElementTooltip, linkPreviewContentClass }, link, linkPreview)
                                } finally {
                                    pauseMutationObserver.next(false) // stop ignoring DOM mutations
                                }
                            })
                    )
                }
            })
        )
        .subscribe()
}
