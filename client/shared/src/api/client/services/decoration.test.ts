import { TextDocumentDecoration } from '@sourcegraph/extension-api-types'
import { TestScheduler } from 'rxjs/testing'
import { TextDocumentIdentifier } from '../types/textDocument'
import { decorationAttachmentStyleForTheme, decorationStyleForTheme } from './decoration'
import { FIXTURE as COMMON_FIXTURE } from './registry.test'

const FIXTURE = {
    ...COMMON_FIXTURE,
    TextDocumentIdentifier: { uri: 'file:///f' } as TextDocumentIdentifier,
}

const FIXTURE_RESULT: TextDocumentDecoration[] | null = [
    {
        range: { start: { line: 1, character: 2 }, end: { line: 3, character: 4 } },
        backgroundColor: 'red',
    },
]

const scheduler = (): TestScheduler => new TestScheduler((a, b) => expect(a).toEqual(b))

describe('decorationStyleForTheme', () => {
    const FIXTURE_RANGE = { start: { line: 1, character: 2 }, end: { line: 3, character: 4 } }

    test('supports no theme overrides', () =>
        expect(decorationStyleForTheme({ range: FIXTURE_RANGE, backgroundColor: 'red' }, true)).toEqual({
            backgroundColor: 'red',
        }))

    test('applies light theme overrides', () =>
        expect(
            decorationStyleForTheme(
                { range: FIXTURE_RANGE, backgroundColor: 'red', light: { backgroundColor: 'blue' } },
                true
            )
        ).toEqual({
            backgroundColor: 'blue',
        }))

    test('applies dark theme overrides', () =>
        expect(
            decorationStyleForTheme(
                {
                    range: FIXTURE_RANGE,
                    backgroundColor: 'red',
                    light: { backgroundColor: 'blue' },
                    dark: { backgroundColor: 'green' },
                },
                false
            )
        ).toEqual({
            backgroundColor: 'green',
        }))
})

describe('decorationAttachmentStyleForTheme', () => {
    test('supports no theme overrides', () =>
        expect(decorationAttachmentStyleForTheme({ color: 'red' }, true)).toEqual({ color: 'red' }))

    test('applies light theme overrides', () =>
        expect(decorationAttachmentStyleForTheme({ color: 'red', light: { color: 'blue' } }, true)).toEqual({
            color: 'blue',
        }))

    test('applies dark theme overrides', () =>
        expect(
            decorationAttachmentStyleForTheme(
                { color: 'red', light: { color: 'blue' }, dark: { color: 'green' } },
                false
            )
        ).toEqual({
            color: 'green',
        }))
})
