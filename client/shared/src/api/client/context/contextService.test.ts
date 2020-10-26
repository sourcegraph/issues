import * as sinon from 'sinon'
import { createContextService } from './contextService'

describe('createContextService()', () => {
    describe('updateContext()', () => {
        it('adds properties', () => {
            const service = createContextService({ clientApplication: 'other' })
            service.updateContext({ a: 1, c: 2, d: 3 })
            const spy = sinon.spy()
            service.data.subscribe(spy)
            sinon.assert.calledOnce(spy)
            expect(service.data.value).toEqual({
                'clientApplication.isSourcegraph': false,
                'clientApplication.extensionAPIVersion.major': 3,
                a: 1,
                c: 2,
                d: 3,
            })
            service.updateContext({ b: 1, e: 4 })
            sinon.assert.calledTwice(spy)
            expect(service.data.value).toEqual({
                'clientApplication.isSourcegraph': false,
                'clientApplication.extensionAPIVersion.major': 3,
                a: 1,
                b: 1,
                c: 2,
                d: 3,
                e: 4,
            })
        })
        it('modifies properties', () => {
            const service = createContextService({ clientApplication: 'other' })
            service.updateContext({ a: 1, c: 2, d: 3 })
            const spy = sinon.spy()
            service.data.subscribe(spy)
            sinon.assert.calledOnce(spy)
            expect(service.data.value).toEqual({
                'clientApplication.isSourcegraph': false,
                'clientApplication.extensionAPIVersion.major': 3,
                a: 1,
                c: 2,
                d: 3,
            })
            service.updateContext({ a: 2 })
            sinon.assert.calledTwice(spy)
            expect(service.data.value).toEqual({
                'clientApplication.isSourcegraph': false,
                'clientApplication.extensionAPIVersion.major': 3,
                a: 2,
                c: 2,
                d: 3,
            })
        })
        it('merges properties', () => {
            const service = createContextService({ clientApplication: 'other' })
            service.updateContext({ a: 1, c: 2, d: 3 })
            const spy = sinon.spy()
            service.data.subscribe(spy)
            sinon.assert.calledOnce(spy)
            expect(service.data.value).toEqual({
                'clientApplication.isSourcegraph': false,
                'clientApplication.extensionAPIVersion.major': 3,
                a: 1,
                c: 2,
                d: 3,
            })
            service.updateContext({ b: 1, c: 3 })
            sinon.assert.calledTwice(spy)
            expect(service.data.value).toEqual({
                'clientApplication.isSourcegraph': false,
                'clientApplication.extensionAPIVersion.major': 3,
                a: 1,
                b: 1,
                c: 3,
                d: 3,
            })
        })
        it('removes a key if it is null', () => {
            const service = createContextService({ clientApplication: 'other' })
            service.updateContext({ a: 1, b: 2 })
            const spy = sinon.spy()
            service.data.subscribe(spy)
            sinon.assert.calledOnce(spy)
            expect(service.data.value).toEqual({
                'clientApplication.isSourcegraph': false,
                'clientApplication.extensionAPIVersion.major': 3,
                a: 1,
                b: 2,
            })
            service.updateContext({ a: 1, b: null })
            sinon.assert.calledTwice(spy)
            expect(service.data.value).toEqual({
                'clientApplication.isSourcegraph': false,
                'clientApplication.extensionAPIVersion.major': 3,
                a: 1,
            })
        })
        it('does not emit if values are the same', () => {
            const service = createContextService({ clientApplication: 'other' })
            service.updateContext({ a: 1, c: 2 })
            const spy = sinon.spy()
            service.data.subscribe(spy)
            sinon.assert.calledOnce(spy)
            expect(service.data.value).toEqual({
                'clientApplication.isSourcegraph': false,
                'clientApplication.extensionAPIVersion.major': 3,
                a: 1,
                c: 2,
            })
            service.updateContext({ a: 1 })
            sinon.assert.calledOnce(spy)
            expect(service.data.value).toEqual({
                'clientApplication.isSourcegraph': false,
                'clientApplication.extensionAPIVersion.major': 3,
                a: 1,
                c: 2,
            })
        })
    })
})
