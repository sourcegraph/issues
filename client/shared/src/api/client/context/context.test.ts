import { EMPTY_SETTINGS_CASCADE, SettingsCascadeOrError } from '../../../settings/settings'
import { CodeEditorWithPartialModel } from '../services/viewerService'
import { computeContext } from './context'

describe('computeContext', () => {
    test('provides config', () => {
        const settings: SettingsCascadeOrError = {
            final: {
                a: 1,
                'a.b': 2,
                'c.d': 3,
            },
            subjects: [],
        }
        expect(computeContext(undefined, settings, {})).toEqual({
            'config.a': 1,
            'config.a.b': 2,
            'config.c.d': 3,
        })
    })

    test('with code editor', () => {
        const editor: CodeEditorWithPartialModel = {
            viewerId: 'editor2',
            type: 'CodeEditor',
            resource: 'file:///a/b.c',
            model: { languageId: 'l' },
            selections: [
                {
                    start: { line: 1, character: 2 },
                    end: { line: 3, character: 4 },
                    anchor: { line: 1, character: 2 },
                    active: { line: 3, character: 4 },
                    isReversed: false,
                },
            ],
            isActive: true,
        }

        expect(computeContext(editor, EMPTY_SETTINGS_CASCADE, {})).toEqual({
            resource: true,
            'resource.uri': 'file:///a/b.c',
            'resource.basename': 'b.c',
            'resource.dirname': 'file:///a',
            'resource.extname': '.c',
            'resource.language': 'l',
            'resource.type': 'textDocument',
            component: true,
            'component.type': 'CodeEditor',
            'component.selection': {
                start: { line: 1, character: 2 },
                end: { line: 3, character: 4 },
                anchor: { line: 1, character: 2 },
                active: { line: 3, character: 4 },
                isReversed: false,
            },
            'component.selections': [
                {
                    start: { line: 1, character: 2 },
                    end: { line: 3, character: 4 },
                    anchor: { line: 1, character: 2 },
                    active: { line: 3, character: 4 },
                    isReversed: false,
                },
            ],
            'component.selection.start': { line: 1, character: 2 },
            'component.selection.start.line': 1,
            'component.selection.start.character': 2,
            'component.selection.end': { line: 3, character: 4 },
            'component.selection.end.line': 3,
            'component.selection.end.character': 4,
        })
    })

    test('without code editor', () => {
        expect(computeContext(undefined, EMPTY_SETTINGS_CASCADE, {})).toEqual({})
    })

    test('code editor with no selection', () => {
        const editorWithNoSelection: CodeEditorWithPartialModel = {
            viewerId: 'editor1',
            type: 'CodeEditor' as const,
            resource: 'file:///a/b.c',
            model: { languageId: 'l' },
            selections: [],
            isActive: true,
        }
        expect(computeContext(editorWithNoSelection, EMPTY_SETTINGS_CASCADE, {})).toEqual({
            resource: true,
            'resource.uri': 'file:///a/b.c',
            'resource.basename': 'b.c',
            'resource.dirname': 'file:///a',
            'resource.extname': '.c',
            'resource.language': 'l',
            'resource.type': 'textDocument',
            component: true,
            'component.type': 'CodeEditor',
            'component.selections': [],
        })
    })

    test('panel', () => {
        expect(
            computeContext(
                undefined,
                EMPTY_SETTINGS_CASCADE,
                {},
                {
                    type: 'panelView',
                    id: 'x',
                    hasLocations: true,
                }
            )
        ).toEqual({
            component: true,
            'panel.activeView.id': 'x',
            'panel.activeView.hasLocations': true,
        })
    })

    test('context fallback', () => {
        expect(computeContext(undefined, EMPTY_SETTINGS_CASCADE, { x: 1 })).toEqual({ x: 1 })
    })
})
