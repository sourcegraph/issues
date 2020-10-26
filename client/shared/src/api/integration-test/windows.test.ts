import { from, of } from 'rxjs'
import { filter, map, switchMap, take, toArray, first } from 'rxjs/operators'
import { ViewComponent, Window } from 'sourcegraph'
import { isDefined } from '../../util/types'
import { TextModel } from '../client/services/modelService'
import { NotificationType } from '../client/services/notifications'
import { assertToJSON, collectSubscribableValues, integrationTestContext } from './testHelpers'

describe('Windows (integration)', () => {
    describe('app.activeWindow', () => {
        test('returns the active window', async () => {
            const { extensionAPI } = await integrationTestContext()
            const viewComponent: Pick<ViewComponent, 'type'> & {
                document: TextModel
            } = {
                type: 'CodeEditor' as const,
                document: { uri: 'file:///f', languageId: 'l', text: 't' },
            }
            assertToJSON(extensionAPI.app.activeWindow, {
                visibleViewComponents: [viewComponent],
                activeViewComponent: viewComponent,
            })
        })
    })

    describe('app.activeWindowChanges', () => {
        // Skipped, as sourcegraph.app.activeWindow is always defined.
        test.skip('reflects changes to the active window', async () => {
            const {
                services: { viewer: viewerService, model: modelService },
                extensionAPI,
            } = await integrationTestContext(undefined, {
                roots: [],
                viewers: [],
            })
            expect(extensionAPI.app.activeWindow).toBeUndefined()
            modelService.addModel({
                uri: 'u',
                languageId: 'l',
                text: 't',
            })
            viewerService.addViewer({
                type: 'CodeEditor',
                resource: 'u',
                selections: [],
                isActive: true,
            })
            await from(extensionAPI.app.activeWindowChanges).pipe(filter(isDefined), first()).toPromise()
            expect(extensionAPI.app.activeWindow).toBeTruthy()
        })
    })

    describe('app.windows', () => {
        test('lists windows', async () => {
            const { extensionAPI } = await integrationTestContext()
            const viewComponent: Pick<ViewComponent, 'type'> & {
                document: TextModel
            } = {
                type: 'CodeEditor' as const,
                document: { uri: 'file:///f', languageId: 'l', text: 't' },
            }
            assertToJSON(extensionAPI.app.windows, [
                {
                    visibleViewComponents: [viewComponent],
                    activeViewComponent: viewComponent,
                },
            ] as Window[])
        })

        test('adds new text documents', async () => {
            const {
                services: { viewer: viewerService, model: modelService },
                extensionAPI,
            } = await integrationTestContext(undefined, { viewers: [], roots: [] })

            modelService.addModel({ uri: 'file:///f2', languageId: 'l2', text: 't2' })
            viewerService.addViewer({
                type: 'CodeEditor',
                resource: 'file:///f2',
                selections: [],
                isActive: true,
            })
            await from(extensionAPI.app.activeWindowChanges)
                .pipe(
                    filter(isDefined),
                    switchMap(activeWindow => activeWindow.activeViewComponentChanges),
                    filter(isDefined),
                    take(1)
                )
                .toPromise()

            const viewComponent: Pick<ViewComponent, 'type'> & {
                document: TextModel
            } = {
                type: 'CodeEditor' as const,
                document: { uri: 'file:///f2', languageId: 'l2', text: 't2' },
            }
            assertToJSON(extensionAPI.app.windows, [
                {
                    visibleViewComponents: [viewComponent],
                    activeViewComponent: viewComponent,
                },
            ] as Window[])
        })
    })

    describe('Window', () => {
        test('Window#visibleViewComponents', async () => {
            const {
                services: { viewer: viewerService, model: modelService },
                extensionAPI,
            } = await integrationTestContext()

            modelService.addModel({
                uri: 'u2',
                languageId: 'l2',
                text: 't2',
            })
            viewerService.addViewer({
                type: 'CodeEditor',
                resource: 'u2',
                selections: [],
                isActive: true,
            })
            await from(extensionAPI.app.activeWindowChanges)
                .pipe(
                    filter(isDefined),
                    switchMap(activeWindow => activeWindow.activeViewComponentChanges),
                    filter(isDefined),
                    take(2)
                )
                .toPromise()

            assertToJSON(extensionAPI.app.windows[0].visibleViewComponents, [
                {
                    type: 'CodeEditor' as const,
                    document: { uri: 'file:///f', languageId: 'l', text: 't' },
                },
                {
                    type: 'CodeEditor' as const,
                    document: { uri: 'u2', languageId: 'l2', text: 't2' },
                },
            ] as ViewComponent[])
        })

        describe('Window#activeViewComponent', () => {
            test('ignores inactive components', async () => {
                const {
                    services: { viewer: viewerService, model: modelService },
                    extensionAPI,
                } = await integrationTestContext()

                modelService.addModel({
                    uri: 'file:///inactive',
                    languageId: 'inactive',
                    text: 'inactive',
                })
                viewerService.addViewer({
                    type: 'CodeEditor',
                    resource: 'file:///inactive',
                    selections: [],
                    isActive: false,
                })

                assertToJSON(extensionAPI.app.windows[0].activeViewComponent, {
                    type: 'CodeEditor' as const,
                    document: { uri: 'file:///f', languageId: 'l', text: 't' },
                })
            })
        })

        describe('Window#activeViewComponentChanges', () => {
            // Skipped, as sourcegraph.app.activeWindow is always defined.
            test.skip('reflects changes to the active window', async () => {
                const {
                    services: { viewer: viewerService, model: modelService },
                    extensionAPI,
                } = await integrationTestContext(undefined, {
                    roots: [],
                    viewers: [],
                })
                modelService.addModel({ uri: 'foo', languageId: 'l1', text: 't1' })
                modelService.addModel({ uri: 'bar', languageId: 'l2', text: 't2' })
                viewerService.addViewer({
                    type: 'CodeEditor',
                    resource: 'foo',
                    selections: [],
                    isActive: true,
                })
                viewerService.removeAllViewers()
                viewerService.addViewer({
                    type: 'CodeEditor',
                    resource: 'bar',
                    selections: [],
                    isActive: true,
                })
                const viewers = await from(extensionAPI.app.activeWindowChanges)
                    .pipe(
                        switchMap(activeWindow => (activeWindow ? activeWindow.activeViewComponentChanges : of(null))),
                        take(4),
                        toArray()
                    )
                    .toPromise()
                assertToJSON(
                    viewers.map(viewer => (viewer && viewer.type === 'CodeEditor' ? viewer.document.uri : null)),
                    [null, 'foo', null, 'bar']
                )
            })
        })

        test('Window#showNotification', async () => {
            const { extensionAPI, services } = await integrationTestContext()
            const values = collectSubscribableValues(services.notifications.showMessages)
            extensionAPI.app.activeWindow!.showNotification('a', NotificationType.Info)
            await extensionAPI.internal.sync()
            expect(values).toEqual([{ message: 'a', type: NotificationType.Info }] as typeof values)
        })

        test('Window#showMessage', async () => {
            const { extensionAPI, services } = await integrationTestContext()
            services.notifications.showMessageRequests.subscribe(({ resolve }) => resolve(Promise.resolve(null)))
            const values = collectSubscribableValues(
                services.notifications.showMessageRequests.pipe(map(({ message, type }) => ({ message, type })))
            )
            expect(await extensionAPI.app.activeWindow!.showMessage('a')).toBe(undefined)
            expect(values).toEqual([{ message: 'a', type: NotificationType.Info }] as typeof values)
        })

        test('Window#showInputBox', async () => {
            const { extensionAPI, services } = await integrationTestContext()
            services.notifications.showInputs.subscribe(({ resolve }) => resolve(Promise.resolve('c')))
            const values = collectSubscribableValues(
                services.notifications.showInputs.pipe(map(({ message, defaultValue }) => ({ message, defaultValue })))
            )
            expect(await extensionAPI.app.activeWindow!.showInputBox({ prompt: 'a', value: 'b' })).toBe('c')
            expect(values).toEqual([{ message: 'a', defaultValue: 'b' }] as typeof values)
        })
    })
})
