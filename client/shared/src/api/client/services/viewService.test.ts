import { NEVER } from 'rxjs'
import { TestScheduler } from 'rxjs/testing'
import { createViewService, View, getView, ViewService } from './viewService'
import { Evaluated, Contributions, ContributableViewContainer } from '../../protocol'
import { MarkupKind } from '@sourcegraph/extension-api-classes'

const scheduler = (): TestScheduler => new TestScheduler((actual, expected) => expect(actual).toEqual(expected))

describe('ViewService', () => {
    test('throws on ID conflict', () => {
        const viewService = createViewService()
        viewService.register('v', ContributableViewContainer.GlobalPage, () => NEVER)
        expect(() => viewService.register('v', ContributableViewContainer.GlobalPage, () => NEVER)).toThrow()
    })

    test('get  nonexistent view', () => {
        const viewService = createViewService()
        scheduler().run(({ expectObservable }) => {
            expectObservable(viewService.get('v', {})).toBe('a', {
                a: null,
            })
        })
    })

    test('register then get', () => {
        const viewService = createViewService()
        scheduler().run(({ cold, expectObservable }) => {
            viewService.register('v', ContributableViewContainer.GlobalPage, () =>
                cold<View>('bc', { b: { title: 'b', content: [] }, c: { title: 'c', content: [] } })
            )
            expectObservable(viewService.get('v', {})).toBe('bc', {
                b: { title: 'b', content: [] },
                c: { title: 'c', content: [] },
            })
        })
    })

    test('register, unsubscribe, then get', () => {
        const viewService = createViewService()
        scheduler().run(({ cold, expectObservable }) => {
            const subscription = viewService.register('v', ContributableViewContainer.GlobalPage, () =>
                cold<View>('bc', { b: { title: 'b', content: [] }, c: { title: 'c', content: [] } })
            )
            subscription.unsubscribe()
            expectObservable(viewService.get('v', {})).toBe('d', {
                d: null,
            })
        })
    })
})

describe('getView', () => {
    const VIEW: View = { title: 't', content: [{ kind: MarkupKind.Markdown, value: 'c' }] }

    test('emits loading then view', () => {
        scheduler().run(({ cold, expectObservable }) => {
            const contributions = cold<Evaluated<Contributions>>('a-b 6s c', {
                a: { views: [] },
                b: {
                    views: [{ id: 'v', where: ContributableViewContainer.GlobalPage }],
                },
                c: {
                    views: [{ id: 'v', where: ContributableViewContainer.GlobalPage }],
                },
            })
            const viewService: Pick<ViewService, 'get'> = {
                get: () =>
                    cold<View | null>('ab- 6s c', {
                        a: VIEW,
                        b: null,
                        c: VIEW,
                    }),
            }
            expectObservable(getView('v', ContributableViewContainer.GlobalPage, {}, contributions, viewService)).toBe(
                'a-- 4997ms c 1002ms d',
                {
                    a: undefined,
                    c: null,
                    d: VIEW,
                }
            )
        })
    })
})
