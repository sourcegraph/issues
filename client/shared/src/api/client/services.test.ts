import { BehaviorSubject, NEVER } from 'rxjs'
import { Services } from './services'

describe('Services', () => {
    test('initializes empty services', () => {
        new Services({
            settings: NEVER,
            updateSettings: () => Promise.reject(new Error('not implemented')),
            requestGraphQL: () => NEVER,
            getScriptURLForExtension: scriptURL => scriptURL,
            clientApplication: 'sourcegraph',
            sideloadedExtensionURL: new BehaviorSubject<string | null>(null),
        })
    })
})
