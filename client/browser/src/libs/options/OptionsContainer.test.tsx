import * as React from 'react'
import { render, RenderResult } from 'react-testing-library'
import { noop, Observable, of } from 'rxjs'
import { switchMap } from 'rxjs/operators'
import { TestScheduler } from 'rxjs/testing'
import * as GQL from '../../../../../shared/src/graphql/schema'
import { OptionsContainer, OptionsContainerProps } from './OptionsContainer'

describe('OptionsContainer', () => {
    const stubs: Pick<
        OptionsContainerProps,
        'fetchCurrentUser' | 'ensureValidSite' | 'toggleFeatureFlag' | 'featureFlags'
    > = {
        fetchCurrentUser: () => new Observable<GQL.IUser>(),
        ensureValidSite: (url: string) => new Observable<void>(),
        toggleFeatureFlag: noop,
        featureFlags: [],
    }

    test('checks the connection status when it mounts', () => {
        const scheduler = new TestScheduler((a, b) => expect(a).toEqual(b))

        scheduler.run(({ cold, expectObservable }) => {
            const values = { a: 'https://test.com' }

            const siteFetches = cold('a', values).pipe(
                switchMap(
                    url =>
                        new Observable<string>(observer => {
                            const ensureValidSite = (url: string) => {
                                observer.next(url)

                                return of(undefined)
                            }

                            render(
                                <OptionsContainer
                                    {...stubs}
                                    sourcegraphURL={url}
                                    ensureValidSite={ensureValidSite}
                                    setSourcegraphURL={noop}
                                />
                            )
                        })
                )
            )

            expectObservable(siteFetches).toBe('a', values)
        })
    })

    test('checks the connection status when it the url updates', () => {
        const scheduler = new TestScheduler((a, b) => expect(a).toEqual(b))

        const buildRenderer = () => {
            let rerender: RenderResult['rerender'] | undefined

            return (ui: React.ReactElement<any>) => {
                if (rerender) {
                    rerender(ui)
                } else {
                    const renderedRes = render(ui)

                    rerender = renderedRes.rerender
                }
            }
        }

        const renderOrRerender = buildRenderer()

        scheduler.run(({ cold, expectObservable }) => {
            const values = { a: 'https://test.com', b: 'https://test1.com' }

            const siteFetches = cold('ab', values).pipe(
                switchMap(
                    url =>
                        new Observable<string>(observer => {
                            const ensureValidSite = (url: string) => {
                                observer.next(url)

                                return of(undefined)
                            }

                            renderOrRerender(
                                <OptionsContainer
                                    {...stubs}
                                    sourcegraphURL={url}
                                    ensureValidSite={ensureValidSite}
                                    setSourcegraphURL={noop}
                                />
                            )
                        })
                )
            )

            expectObservable(siteFetches).toBe('ab', values)
        })
    })

    test('handles when an error is thrown checking the site connection', () => {
        const ensureValidSite = () => {
            throw new Error('no site, woops')
        }

        try {
            render(
                <OptionsContainer
                    {...stubs}
                    sourcegraphURL={'https://test.com'}
                    ensureValidSite={ensureValidSite}
                    setSourcegraphURL={noop}
                />
            )
        } catch (err) {
            throw new Error("shouldn't be hit")
        }
    })
})
