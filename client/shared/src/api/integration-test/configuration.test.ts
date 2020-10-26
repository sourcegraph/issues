import { BehaviorSubject, of } from 'rxjs'
import { EMPTY_SETTINGS_CASCADE, SettingsCascadeOrError } from '../../settings/settings'
import { SettingsEdit } from '../client/services/settings'
import { assertToJSON, integrationTestContext } from './testHelpers'

describe('Configuration (integration)', () => {
    test('is synchronously available', async () => {
        const { extensionAPI } = await integrationTestContext({ settings: of(EMPTY_SETTINGS_CASCADE) })
        expect(() => extensionAPI.configuration.subscribe(() => undefined)).not.toThrow()
        expect(() => extensionAPI.configuration.get()).not.toThrow()
    })

    describe('Configuration#get', () => {
        test('gets configuration', async () => {
            const { extensionAPI } = await integrationTestContext({
                settings: of({ final: { a: 1 }, subjects: [] }),
            })
            assertToJSON(extensionAPI.configuration.get(), { a: 1 })
            expect(extensionAPI.configuration.get().value).toEqual({ a: 1 })
        })
    })

    describe('Configuration#update', () => {
        test('updates configuration', async () => {
            const calls: (SettingsEdit | string)[] = []
            const { extensionAPI } = await integrationTestContext({
                settings: of({ final: { a: 1 }, subjects: [{ subject: {} as any, lastID: null, settings: null }] }),
                // eslint-disable-next-line @typescript-eslint/require-await
                updateSettings: async (_subject, edit) => {
                    calls.push(edit)
                },
            })

            await extensionAPI.configuration.get().update('a', 2)
            await extensionAPI.internal.sync()
            expect(calls).toEqual([{ path: ['a'], value: 2 }] as SettingsEdit[])
            calls.length = 0 // clear

            await extensionAPI.configuration.get().update('a', 3)
            await extensionAPI.internal.sync()
            expect(calls).toEqual([{ path: ['a'], value: 3 }] as SettingsEdit[])
        })
    })

    describe('configuration.subscribe', () => {
        test('subscribes to changes', async () => {
            const mockSettings = new BehaviorSubject<SettingsCascadeOrError>(EMPTY_SETTINGS_CASCADE)
            const { extensionAPI } = await integrationTestContext({ settings: mockSettings })

            let calls = 0
            extensionAPI.configuration.subscribe(() => calls++)
            expect(calls).toBe(1) // called initially

            mockSettings.next(EMPTY_SETTINGS_CASCADE)
            await extensionAPI.internal.sync()
            expect(calls).toBe(2)
        })
    })
})
