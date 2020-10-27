import { Selection } from '@sourcegraph/extension-api-types'
import * as sinon from 'sinon'
import { from, Observable } from 'rxjs'
import { first, tap, bufferCount, map } from 'rxjs/operators'
import { TestScheduler } from 'rxjs/testing'
import {
    createViewerService,
    ViewerService,
    getActiveCodeEditorPosition,
    CodeEditorData,
    ViewerUpdate,
    ViewerData,
} from './viewerService'
import { ModelService } from './modelService'

const FIXTURE_MODEL_SERVICE: Pick<ModelService, 'removeModel'> = {
    removeModel: sinon.spy(),
}

export function createTestViewerService({
    modelService,
    viewers,
    updates,
}: {
    modelService?: ModelService
    viewers?: ViewerData[]
    updates?: Observable<ViewerUpdate[]>
}): ViewerService {
    const viewerService = createViewerService(modelService || FIXTURE_MODEL_SERVICE)
    if (viewers) {
        for (const viewer of viewers) {
            viewerService.addViewer(viewer)
        }
    }
    const viewerUpdates = updates
        ? from(updates).pipe(
              tap(updates => {
                  for (const update of updates) {
                      switch (update.type) {
                          case 'added':
                              viewerService.addViewer(update.viewerData)
                              break
                          case 'updated':
                              viewerService.setSelections(update, update.viewerData.selections)
                              break
                          case 'deleted':
                              viewerService.removeViewer(update)
                              break
                      }
                  }
              })
          )
        : viewerService.viewerUpdates
    return { ...viewerService, viewerUpdates }
}

const scheduler = (): TestScheduler => new TestScheduler((a, b) => expect(a).toEqual(b))

const SELECTIONS: Selection[] = [
    {
        start: { line: 3, character: -1 },
        end: { line: 3, character: -1 },
        anchor: { line: 3, character: -1 },
        active: { line: 3, character: -1 },
        isReversed: false,
    },
]

describe('viewerService', () => {
    test('addViewer', async () => {
        const viewerService = createViewerService(FIXTURE_MODEL_SERVICE)
        const viewerData: CodeEditorData = {
            type: 'CodeEditor',
            resource: 'u',
            selections: [],
            isActive: true,
        }
        const editorAdded = from(viewerService.viewerUpdates).pipe(first()).toPromise()
        const { viewerId } = viewerService.addViewer(viewerData)
        expect(viewerId).toEqual('viewer#0')
        expect(await editorAdded).toEqual([
            {
                type: 'added',
                viewerId,
                viewerData,
            },
        ] as ViewerUpdate[])
    })

    describe('observeViewer', () => {
        test('emits error if editor does not exist', () => {
            scheduler().run(({ expectObservable }) => {
                const viewerService = createViewerService(FIXTURE_MODEL_SERVICE)
                expectObservable(from(viewerService.observeViewer({ viewerId: 'x' }))).toBe(
                    '#',
                    {},
                    new Error('Viewer not found: x')
                )
            })
        })

        test('emits on selections changes', async () => {
            const viewerService = createViewerService(FIXTURE_MODEL_SERVICE)
            const viewerId = viewerService.addViewer({
                type: 'CodeEditor',
                resource: 'r',
                selections: [],
                isActive: true,
            })
            const changes = (viewerService.observeViewer(viewerId) as Observable<CodeEditorData>)
                .pipe(
                    map(({ selections }) => selections),
                    bufferCount(2),
                    first()
                )
                .toPromise()
            viewerService.setSelections(viewerId, SELECTIONS)
            expect(await changes).toMatchObject([[], SELECTIONS])
        })

        test('completes when the editor is removed', async () => {
            const viewerService = createViewerService(FIXTURE_MODEL_SERVICE)
            const viewerId = viewerService.addViewer({
                type: 'CodeEditor',
                resource: 'u',
                selections: [],
                isActive: true,
            })
            const removed = viewerService.observeViewer(viewerId).toPromise()
            viewerService.removeViewer(viewerId)
            await removed
        })
    })

    describe('setSelections', () => {
        test('ok', async () => {
            const viewerService = createViewerService(FIXTURE_MODEL_SERVICE)
            const editor: CodeEditorData = {
                type: 'CodeEditor',
                resource: 'u',
                selections: [],
                isActive: true,
            }
            const { viewerId } = viewerService.addViewer(editor)
            const selectionsSet = from(viewerService.viewerUpdates).pipe(first()).toPromise()
            viewerService.setSelections({ viewerId }, SELECTIONS)
            expect(await selectionsSet).toMatchObject([
                { type: 'updated', viewerId, viewerData: { selections: SELECTIONS } },
            ] as ViewerUpdate[])
        })
        test('not found', () => {
            const viewerService = createViewerService(FIXTURE_MODEL_SERVICE)
            expect(() => viewerService.setSelections({ viewerId: 'x' }, [])).toThrowError('Viewer not found: x')
        })
    })

    describe('removeViewer', () => {
        test('ok', () => {
            const viewerService = createViewerService(FIXTURE_MODEL_SERVICE)
            const editor = viewerService.addViewer({
                type: 'CodeEditor',
                resource: 'u',
                selections: [],
                isActive: true,
            })
            viewerService.removeViewer(editor)
            expect(viewerService.viewers.size).toBe(0)
        })
        test('not found', () => {
            const viewerService = createViewerService(FIXTURE_MODEL_SERVICE)
            expect(() => viewerService.removeViewer({ viewerId: 'x' })).toThrowError('Viewer not found: x')
        })

        it('calls removeModel() when removing an editor', () => {
            const removeModel = sinon.spy((uri: string) => {
                /* noop */
            })
            const viewerService = createViewerService({
                removeModel,
            })
            const editor1 = viewerService.addViewer({
                type: 'CodeEditor',
                resource: 'u',
                selections: [],
                isActive: true,
            })
            const editor2 = viewerService.addViewer({
                type: 'CodeEditor',
                resource: 'u',
                selections: [],
                isActive: true,
            })
            viewerService.removeViewer(editor1)
            viewerService.removeViewer(editor2)
            sinon.assert.calledOnce(removeModel)
            sinon.assert.calledWith(removeModel, 'u')
        })
    })

    test('removeAllViewers', async () => {
        const viewerService = createViewerService(FIXTURE_MODEL_SERVICE)
        const editor: CodeEditorData = { type: 'CodeEditor', resource: 'u', selections: [], isActive: true }
        viewerService.addViewer(editor)
        viewerService.addViewer(editor)
        viewerService.addViewer(editor)
        const editorsRemoved = from(viewerService.viewerUpdates).pipe(first()).toPromise()
        viewerService.removeAllViewers()
        expect(await editorsRemoved).toMatchObject([
            { type: 'deleted', viewerId: 'viewer#0' },
            { type: 'deleted', viewerId: 'viewer#1' },
            { type: 'deleted', viewerId: 'viewer#2' },
        ])
    })
})

describe('getActiveCodeEditorPosition', () => {
    test('null if code editor is empty', () => {
        expect(getActiveCodeEditorPosition(undefined)).toBeNull()
    })

    test('null if active code editor has no selection', () => {
        expect(
            getActiveCodeEditorPosition({
                viewerId: '1',
                type: 'CodeEditor',
                isActive: true,
                selections: [],
                resource: 'u',
            })
        ).toBeNull()
    })

    test('null if active code editor has empty selection', () => {
        expect(
            getActiveCodeEditorPosition({
                viewerId: '1',
                type: 'CodeEditor',
                isActive: true,
                selections: [
                    {
                        start: { line: 3, character: -1 },
                        end: { line: 3, character: -1 },
                        anchor: { line: 3, character: -1 },
                        active: { line: 3, character: -1 },
                        isReversed: false,
                    },
                ],
                resource: 'u',
            })
        ).toBeNull()
    })

    test('equivalent params', () => {
        expect(
            getActiveCodeEditorPosition({
                viewerId: '1',
                type: 'CodeEditor',
                isActive: true,
                selections: [
                    {
                        start: { line: 3, character: 2 },
                        end: { line: 3, character: 5 },
                        anchor: { line: 3, character: 2 },
                        active: { line: 3, character: 5 },
                        isReversed: false,
                    },
                ],
                resource: 'u',
            })
        ).toEqual({ textDocument: { uri: 'u' }, position: { line: 3, character: 2 } })
    })
})
