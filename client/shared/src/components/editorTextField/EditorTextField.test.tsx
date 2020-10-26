import { of } from 'rxjs'
import { ViewerId, CodeEditorData } from '../../api/client/services/viewerService'
import { EditorTextFieldUtils } from './EditorTextField'
import { Selection } from '@sourcegraph/extension-api-types'
import * as sinon from 'sinon'
import { noop } from 'lodash'
import { TextModel } from '../../api/client/services/modelService'

describe('EditorTextFieldUtils', () => {
    describe('getEditorDataFromElement', () => {
        test('empty selection', () => {
            const textArea = document.createElement('textarea')
            textArea.value = 'abc'
            textArea.setSelectionRange(2, 2)
            expect(EditorTextFieldUtils.getEditorDataFromElement(textArea)).toEqual({
                text: 'abc',
                selections: [
                    {
                        anchor: { line: 0, character: 2 },
                        active: { line: 0, character: 2 },
                        start: { line: 0, character: 2 },
                        end: { line: 0, character: 2 },
                        isReversed: false,
                    },
                ],
            })
        })
        test('forward selection', () => {
            const textArea = document.createElement('textarea')
            textArea.value = 'abc'
            textArea.setSelectionRange(2, 3)
            expect(EditorTextFieldUtils.getEditorDataFromElement(textArea)).toEqual({
                text: 'abc',
                selections: [
                    {
                        anchor: { line: 0, character: 2 },
                        active: { line: 0, character: 3 },
                        start: { line: 0, character: 2 },
                        end: { line: 0, character: 3 },
                        isReversed: false,
                    },
                ],
            })
        })
        test('backward selection', () => {
            const textArea = document.createElement('textarea')
            textArea.value = 'abc'
            textArea.setSelectionRange(2, 3, 'backward')
            expect(EditorTextFieldUtils.getEditorDataFromElement(textArea)).toEqual({
                text: 'abc',
                selections: [
                    {
                        anchor: { line: 0, character: 3 },
                        active: { line: 0, character: 2 },
                        start: { line: 0, character: 3 },
                        end: { line: 0, character: 2 },
                        isReversed: true,
                    },
                ],
            })
        })
    })

    test('updateEditorSelectionFromElement', () => {
        const textArea = document.createElement('textarea')
        textArea.value = 'abc'
        textArea.setSelectionRange(2, 3, 'backward')
        const setSelections = sinon.spy<(editor: ViewerId, selections: Selection[]) => void>(noop)
        EditorTextFieldUtils.updateEditorSelectionFromElement({ setSelections }, { viewerId: 'e' }, textArea)
        sinon.assert.calledOnce(setSelections)
        expect(setSelections.args[0][0]).toEqual({ viewerId: 'e' })
        expect(setSelections.args[0][1]).toEqual([
            {
                anchor: { line: 0, character: 3 },
                active: { line: 0, character: 2 },
                start: { line: 0, character: 3 },
                end: { line: 0, character: 2 },
                isReversed: true,
            },
        ])
    })

    test('updateModelFromElement', () => {
        const textArea = document.createElement('textarea')
        textArea.value = 'abc'
        textArea.setSelectionRange(2, 3, 'backward')
        const updateModel = sinon.spy<(uri: string, text: string) => void>(noop)
        EditorTextFieldUtils.updateModelFromElement({ updateModel }, 'u', textArea)
        sinon.assert.calledOnce(updateModel)
        expect(updateModel.args[0][0]).toEqual('u')
        expect(updateModel.args[0][1]).toEqual('abc')
    })

    describe('updateElementOnEditorOrModelChanges', () => {
        test('forward selection', () => {
            const textArea = document.createElement('textarea')
            textArea.value = 'abc'
            const setValue = sinon.spy<(value: string) => void>(noop)
            const subscription = EditorTextFieldUtils.updateElementOnEditorOrModelChanges(
                {
                    observeViewer: () =>
                        of<CodeEditorData>({
                            type: 'CodeEditor',
                            resource: 'u',
                            selections: [
                                {
                                    anchor: { line: 0, character: 2 },
                                    active: { line: 0, character: 3 },
                                    start: { line: 0, character: 2 },
                                    end: { line: 0, character: 3 },
                                    isReversed: false,
                                },
                            ],
                            isActive: true,
                        }),
                },
                {
                    observeModel: () => of<TextModel>({ uri: 'u', languageId: 'l', text: 'xyz' }),
                },
                { viewerId: 'e' },
                setValue,
                { current: textArea }
            )
            sinon.assert.calledOnce(setValue)
            expect(setValue.args[0][0]).toEqual('xyz')
            expect(textArea.selectionStart).toBe(2)
            expect(textArea.selectionEnd).toBe(3)
            expect(textArea.selectionDirection).toBe('forward')
            subscription.unsubscribe()
        })
        test('backward selection', () => {
            const textArea = document.createElement('textarea')
            textArea.value = 'abc'
            const setValue = sinon.spy<(value: string) => void>(noop)
            const subscription = EditorTextFieldUtils.updateElementOnEditorOrModelChanges(
                {
                    observeViewer: () =>
                        of<CodeEditorData>({
                            type: 'CodeEditor',
                            resource: 'u',
                            selections: [
                                {
                                    anchor: { line: 0, character: 3 },
                                    active: { line: 0, character: 2 },
                                    start: { line: 0, character: 3 },
                                    end: { line: 0, character: 2 },
                                    isReversed: true,
                                },
                            ],
                            isActive: true,
                        }),
                },
                {
                    observeModel: () => of<TextModel>({ uri: 'u', languageId: 'l', text: 'xyz' }),
                },
                { viewerId: 'e' },
                setValue,
                { current: textArea }
            )
            expect(textArea.selectionStart).toBe(2)
            expect(textArea.selectionEnd).toBe(3)
            expect(textArea.selectionDirection).toBe('backward')
            subscription.unsubscribe()
        })
    })
})
