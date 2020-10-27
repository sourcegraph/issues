import { from } from 'rxjs'
import { first, takeWhile, bufferCount, map } from 'rxjs/operators'
import { createModelService } from './modelService'

describe('ModelService', () => {
    describe('addModel', () => {
        it('adds', () => {
            const modelService = createModelService()
            modelService.addModel({ uri: 'u', text: 't', languageId: 'l' })
            expect([...modelService.models.values()]).toEqual([
                {
                    uri: 'u',
                    text: 't',
                    languageId: 'l',
                },
            ])
        })
        it('refuses to add model with duplicate URI', () => {
            const modelService = createModelService()
            modelService.addModel({ uri: 'u', text: 't', languageId: 'l' })
            expect(() => {
                modelService.addModel({ uri: 'u', text: 't2', languageId: 'l2' })
            }).toThrowError('model already exists with URI u')
            expect([...modelService.models.values()]).toEqual([
                {
                    uri: 'u',
                    text: 't',
                    languageId: 'l',
                },
            ])
        })
    })

    test('hasModel', () => {
        const modelService = createModelService()
        modelService.addModel({ uri: 'u', text: 't', languageId: 'l' })
        expect(modelService.hasModel('u')).toBeTruthy()
        expect(modelService.hasModel('u2')).toBeFalsy()
    })

    describe('updateModel', () => {
        test('existing model', () => {
            const modelService = createModelService()
            modelService.addModel({ uri: 'u', text: 't', languageId: 'l' })
            modelService.updateModel('u', 't2')
            expect([...modelService.models.values()]).toEqual([{ uri: 'u', text: 't2', languageId: 'l' }])
        })

        test('nonexistent model', () => {
            const modelService = createModelService()
            expect(() => modelService.updateModel('x', 't2')).toThrowError('model does not exist with URI x')
        })
    })

    describe('modelUpdates', () => {
        it('emits when a model is added', async () => {
            const modelService = createModelService()
            const modelAdded = from(modelService.modelUpdates).pipe(first()).toPromise()
            modelService.addModel({ uri: 'u', languageId: 'x', text: 't' })
            expect(await modelAdded).toMatchObject([{ uri: 'u', languageId: 'x', text: 't' }])
        })

        it('emits when a model is removed', async () => {
            const modelService = createModelService()
            const modelRemoved = from(modelService.modelUpdates)
                .pipe(takeWhile(updates => updates.every(({ uri, type }) => uri !== 'u' || type !== 'deleted')))
                .toPromise()
            modelService.addModel({ uri: 'u', languageId: 'x', text: 't' })
            modelService.removeModel('u')
            await modelRemoved
        })
    })

    describe('activeLanguages', () => {
        it('emits when a model with a previously unseen language is added', async () => {
            const modelService = createModelService()
            const values = from(modelService.activeLanguages)
                .pipe(
                    map(activeLanguages => [...activeLanguages]),
                    bufferCount(3),
                    first()
                )
                .toPromise()
            modelService.addModel({ uri: 'u', languageId: 'l1', text: 't' })
            modelService.addModel({ uri: 'u2', languageId: 'l1', text: 't' })
            modelService.addModel({ uri: 'u3', languageId: 'l1', text: 't' })
            modelService.addModel({ uri: 'u4', languageId: 'l1', text: 't' })
            modelService.addModel({ uri: 'u5', languageId: 'l2', text: 't' })
            expect(await values).toMatchObject([[], ['l1'], ['l1', 'l2']])
        })

        it('emits when the last model referencing a language is removed', async () => {
            const modelService = createModelService()
            const values = from(modelService.activeLanguages)
                .pipe(
                    map(activeLanguages => [...activeLanguages]),
                    bufferCount(5),
                    first()
                )
                .toPromise()
            modelService.addModel({ uri: 'u', languageId: 'l1', text: 't' })
            modelService.addModel({ uri: 'u2', languageId: 'l1', text: 't' })
            modelService.addModel({ uri: 'u3', languageId: 'l2', text: 't' })
            modelService.addModel({ uri: 'u4', languageId: 'l2', text: 't' })
            modelService.addModel({ uri: 'u5', languageId: 'l1', text: 't' })
            modelService.removeModel('u3')
            modelService.removeModel('u4')
            modelService.removeModel('u')
            modelService.removeModel('u2')
            modelService.removeModel('u5')
            expect(await values).toMatchObject([[], ['l1'], ['l1', 'l2'], ['l1'], []])
        })
    })
})
