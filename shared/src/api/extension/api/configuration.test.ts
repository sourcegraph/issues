import * as sinon from 'sinon'
import { ExtConfiguration } from './configuration'
import { ProxyResult, proxyValueSymbol } from '@sourcegraph/comlink'
import { ClientConfigurationAPI } from '../../client/api/configuration'

describe('ExtConfiguration', () => {
    const PROXY: ProxyResult<ClientConfigurationAPI> = {
        [proxyValueSymbol]: Promise.resolve(true),
        $acceptConfigurationUpdate: sinon.stub(),
    }
    describe('get()', () => {
        test("throws if initial settings haven't been received", () => {
            const configuration = new ExtConfiguration(PROXY)
            expect(() => configuration.get()).toThrow('unexpected internal error: settings data is not yet available')
        })
        test('returns the latest settings', () => {
            const configuration = new ExtConfiguration<{ a: string }>(PROXY)
            configuration.$acceptConfigurationData({ subjects: [], final: { a: 'b' } })
            configuration.$acceptConfigurationData({ subjects: [], final: { a: 'c' } })
            expect(configuration.get().get('a')).toBe('c')
        })
    })
    describe('changes', () => {
        test('emits as soon as initial settings are received', () => {
            const configuration = new ExtConfiguration(PROXY)
            const subscriber = sinon.spy()
            configuration.changes.subscribe(subscriber)
            sinon.assert.notCalled(subscriber)
            configuration.$acceptConfigurationData({ subjects: [], final: { a: 'b' } })
            sinon.assert.calledOnce(subscriber)
        })
        test('emits immediately on subscription if initial settings have already been received', () => {
            const configuration = new ExtConfiguration(PROXY)
            configuration.$acceptConfigurationData({ subjects: [], final: { a: 'b' } })
            const subscriber = sinon.spy()
            configuration.changes.subscribe(subscriber)
            sinon.assert.calledOnce(subscriber)
        })

        test('emits when settings are updated', () => {
            const configuration = new ExtConfiguration(PROXY)
            const subscriber = sinon.spy()
            configuration.changes.subscribe(subscriber)
            configuration.$acceptConfigurationData({ subjects: [], final: { a: 'b' } })
            configuration.$acceptConfigurationData({ subjects: [], final: { a: 'c' } })
            configuration.$acceptConfigurationData({ subjects: [], final: { a: 'd' } })
            sinon.assert.calledThrice(subscriber)
        })
    })
})
