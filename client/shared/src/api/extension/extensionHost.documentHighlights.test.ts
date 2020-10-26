import { DocumentHighlight } from 'sourcegraph'
import { Range } from '@sourcegraph/extension-api-classes'
import { initNewExtensionAPI, mergeDocumentHighlightResults } from './flatExtensionApi'
import { pretendRemote } from '../util'
import { MainThreadAPI } from '../contract'
import { SettingsCascade } from '../../settings/settings'
import { Observer } from 'rxjs'
import { ProxyMarked, proxyMarker, Remote } from 'comlink'
import { ExtensionDocuments } from './api/documents'
import { LOADING } from '@sourcegraph/codeintellify'

const range1 = new Range(1, 2, 3, 4)
const range2 = new Range(2, 3, 4, 5)
const range3 = new Range(3, 4, 5, 6)

describe('mergeDocumentHighlightResults', () => {
    it('merges non DocumentHighlight values into empty arrays', () => {
        expect(mergeDocumentHighlightResults([LOADING])).toStrictEqual([])
        expect(mergeDocumentHighlightResults([null])).toStrictEqual([])
        expect(mergeDocumentHighlightResults([undefined])).toStrictEqual([])
        // and yes, there can be several
        expect(mergeDocumentHighlightResults([null, LOADING])).toStrictEqual([])
    })

    it('merges a DocumentHighlight into result', () => {
        const highlight1: DocumentHighlight = { range: range1 }
        const highlight2: DocumentHighlight = { range: range2 }
        const highlight3: DocumentHighlight = { range: range3 }
        const merged: DocumentHighlight[] = [highlight1, highlight2, highlight3]
        expect(mergeDocumentHighlightResults([[highlight1], [highlight2, highlight3]])).toEqual(merged)
    })

    it('omits non DocumentHighlight values from document highlight result', () => {
        const highlight: DocumentHighlight = { range: range1 }
        const merged: DocumentHighlight[] = [highlight]
        expect(mergeDocumentHighlightResults([[highlight], null, LOADING, undefined])).toEqual(merged)
    })
})

describe('getDocumentHighlights from ExtensionHost API, it aims to have more e2e feel', () => {
    // integration(ish) tests for scenarios not covered by providers tests
    const noopMain = pretendRemote<MainThreadAPI>({})
    const emptySettings: SettingsCascade<object> = {
        subjects: [],
        final: {},
    }

    const observe = <T>(onValue: (value: T) => void): Remote<Observer<T> & ProxyMarked> =>
        pretendRemote({
            next: onValue,
            error: (error: any) => {
                throw error
            },
            complete: () => {},
            [proxyMarker]: Promise.resolve(true as const),
        })

    const documentHighlight = (value: number): DocumentHighlight => ({
        range: new Range(value, value, value, value),
    })

    it('restarts document highlights call if a provider was added or removed', () => {
        const typescriptFileUri = 'file:///f.ts'
        const documents = new ExtensionDocuments(() => Promise.resolve())
        documents.$acceptDocumentData([
            {
                type: 'added',
                languageId: 'ts',
                text: 'body',
                uri: typescriptFileUri,
            },
        ])

        const { exposedToMain, languages } = initNewExtensionAPI(noopMain, emptySettings, documents)

        let counter = 0
        languages.registerDocumentHighlightProvider([{ pattern: '*.ts' }], {
            provideDocumentHighlights: () => [documentHighlight(++counter)],
        })

        let results: DocumentHighlight[][] = []
        exposedToMain
            .getDocumentHighlights({
                position: { line: 1, character: 2 },
                textDocument: { uri: typescriptFileUri },
            })
            .subscribe(observe(value => results.push(value)))

        // first provider results
        expect(results).toEqual([[], [documentHighlight(1)]])
        results = []

        const subscription = languages.registerDocumentHighlightProvider([{ pattern: '*.ts' }], {
            provideDocumentHighlights: () => [documentHighlight(0)],
        })

        // second and first
        expect(results).toEqual([[], [2, 0].map(value => documentHighlight(value))])
        results = []

        subscription.unsubscribe()

        // just first was queried for the third time
        expect(results).toEqual([[], [documentHighlight(3)]])
    })
})
