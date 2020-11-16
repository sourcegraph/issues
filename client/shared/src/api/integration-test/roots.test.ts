import { WorkspaceRoot } from 'sourcegraph'
import { collectSubscribableValues, integrationTestContext } from './testHelpers'

describe('Workspace roots (integration)', () => {
    describe('workspace.roots', () => {
        test('lists roots', async () => {
            const { extensionAPI } = await integrationTestContext()
            expect(extensionAPI.workspace.roots).toEqual([{ uri: new URL('file:///') }] as WorkspaceRoot[])
        })

        test('adds new text documents', async () => {
            const { extensionHost, extensionAPI } = await integrationTestContext()

            await extensionHost.removeWorkspaceRoot('file:///')
            await extensionHost.addWorkspaceRoot({ uri: 'file:///a' })
            await extensionHost.addWorkspaceRoot({ uri: 'file:///b' })

            expect(extensionAPI.workspace.roots).toEqual([
                { uri: new URL('file:///a') },
                { uri: new URL('file:///b') },
            ] as WorkspaceRoot[])
        })
    })

    describe('workspace.rootChanges', () => {
        test('fires when a root is added or removed', async () => {
            const { extensionHost, extensionAPI } = await integrationTestContext()

            const values = collectSubscribableValues(extensionAPI.workspace.rootChanges)
            expect(values).toEqual([] as void[])

            await extensionHost.addWorkspaceRoot({ uri: 'file:///a' })
            // rootChanges lets us know when roots changed, but it doesn't emit the new value, it just emits undefined.
            expect(values).toEqual([undefined])
        })
    })
})
