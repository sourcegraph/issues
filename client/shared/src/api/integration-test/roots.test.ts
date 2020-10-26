import { WorkspaceRoot } from 'sourcegraph'
import { collectSubscribableValues, integrationTestContext } from './testHelpers'

describe('Workspace roots (integration)', () => {
    describe('workspace.roots', () => {
        test('lists roots', async () => {
            const { extensionAPI } = await integrationTestContext()
            expect(extensionAPI.workspace.roots).toEqual([{ uri: new URL('file:///') }] as WorkspaceRoot[])
        })

        test('adds new text documents', async () => {
            const {
                services: { workspace },
                extensionAPI,
            } = await integrationTestContext()

            workspace.roots.next([{ uri: 'file:///a' }, { uri: 'file:///b' }])
            await extensionAPI.internal.sync()

            expect(extensionAPI.workspace.roots).toEqual([
                { uri: new URL('file:///a') },
                { uri: new URL('file:///b') },
            ] as WorkspaceRoot[])
        })
    })

    describe('workspace.rootChanges', () => {
        test('fires when a root is added or removed', async () => {
            const {
                services: { workspace },
                extensionAPI,
            } = await integrationTestContext()

            const values = collectSubscribableValues(extensionAPI.workspace.rootChanges)
            expect(values).toEqual([] as void[])

            workspace.roots.next([{ uri: 'file:///a' }])
            await extensionAPI.internal.sync()

            expect(values).toEqual([undefined])
        })
    })
})
