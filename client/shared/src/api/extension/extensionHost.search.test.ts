import { initNewExtensionAPI } from './flatExtensionApi'
import { pretendRemote, noopMainThreadAPI } from '../util'
import { SettingsCascade } from '../../settings/settings'
import { Observer } from 'rxjs'
import { ProxyMarked, proxyMarker, Remote } from 'comlink'

const remoteNoopMainThreadAPI = pretendRemote(noopMainThreadAPI)
const emptySettings: SettingsCascade<object> = { subjects: [], final: {} }

const observe = (onValue: (value: string) => void): Remote<Observer<string> & ProxyMarked> =>
    pretendRemote({
        next: onValue,
        error: (error: any) => {
            throw error
        },
        complete: () => {},
        [proxyMarker]: Promise.resolve(true as const),
    })

describe('QueryTransformers', () => {
    it('returns the same query with no registered transformers', () => {
        const { exposedToMain } = initNewExtensionAPI(remoteNoopMainThreadAPI, emptySettings)

        const results: string[] = []
        exposedToMain.transformSearchQuery('a').subscribe(observe(value => results.push(value)))
        expect(results).toEqual(['a'])
    })

    it('can work with Promise based transformers', async () => {
        const { exposedToMain, search } = initNewExtensionAPI(remoteNoopMainThreadAPI, emptySettings)
        search.registerQueryTransformer({ transformQuery: query => Promise.resolve(query + '!') })
        const result = await new Promise(resolve => exposedToMain.transformSearchQuery('a').subscribe(observe(resolve)))
        expect(result).toEqual('a!')
    })

    it('emits a new transformed value if there is a new transformer', () => {
        const { exposedToMain, search } = initNewExtensionAPI(remoteNoopMainThreadAPI, emptySettings)

        const results: string[] = []
        exposedToMain.transformSearchQuery('a').subscribe(observe(value => results.push(value)))
        expect(results).toEqual(['a'])

        search.registerQueryTransformer({ transformQuery: query => query + '!' })
        expect(results).toEqual(['a', 'a!'])
    })

    it('emits new value if a transformer was removed', () => {
        const { exposedToMain, search } = initNewExtensionAPI(remoteNoopMainThreadAPI, emptySettings)

        const transformerSubscription = search.registerQueryTransformer({ transformQuery: query => query + '!' })

        const results: string[] = []
        exposedToMain.transformSearchQuery('a').subscribe(observe(value => results.push(value)))
        expect(results).toEqual(['a!'])
        transformerSubscription.unsubscribe()

        expect(results).toEqual(['a!', 'a'])
    })

    it('emits modified query if there are any transformers registered', () => {
        const { exposedToMain, search } = initNewExtensionAPI(remoteNoopMainThreadAPI, emptySettings)
        search.registerQueryTransformer({ transformQuery: query => query + '!' })

        const results: string[] = []
        exposedToMain.transformSearchQuery('a').subscribe(observe(value => results.push(value)))
        expect(results).toEqual(['a!'])
    })

    it('cancels previous transformer chains', async () => {
        const { exposedToMain, search } = initNewExtensionAPI(remoteNoopMainThreadAPI, emptySettings)

        // collect all pending promises and their triggers from the first transformer
        // to manually manipulate them later
        const resolves: ((q: string) => void)[] = []
        const promises: Promise<string>[] = []

        search.registerQueryTransformer({
            transformQuery: () => {
                const promise = new Promise<string>(resolve => {
                    resolves.push(resolve)
                })
                promises.push(promise)
                return promise
            },
        })

        let secondTransformerCallCount = 0
        search.registerQueryTransformer({
            transformQuery: query => {
                secondTransformerCallCount++
                return query
            },
        })

        const results: string[] = []

        exposedToMain.transformSearchQuery('a').subscribe(observe(value => results.push(value)))
        // we hanged in the first transformer
        expect(results).toEqual([])
        expect(promises.length).toBe(1)

        search.registerQueryTransformer({ transformQuery: query => query + '!' })

        // we reissued the transformation and waiting for a new promise to resolve
        expect(promises.length).toBe(2)
        // means that we haven't executed the second transformer yet
        // because it scheduled after the promise
        expect(secondTransformerCallCount).toBe(0)

        for (const resolve of resolves) {
            resolve('finally')
        }

        await Promise.all(promises)

        // the first chain was aborted
        expect(secondTransformerCallCount).toBe(1)
        expect(results).toEqual(['finally!'])
    })
})
