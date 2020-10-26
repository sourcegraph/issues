import { Subscription } from 'rxjs'
import { CommandEntry, CommandRegistry, executeCommand } from './command'

describe('CommandRegistry', () => {
    test('is initially empty', () => {
        expect(new CommandRegistry().commandsSnapshot).toEqual([])
    })

    test('registers and unregisters commands', () => {
        const subscriptions = new Subscription()
        const registry = new CommandRegistry()
        const entry1: CommandEntry = { command: 'command1', run: () => Promise.resolve() }
        const entry2: CommandEntry = { command: 'command2', run: () => Promise.resolve() }

        const unregister1 = subscriptions.add(registry.registerCommand(entry1))
        expect(registry.commandsSnapshot).toEqual([entry1])

        const unregister2 = subscriptions.add(registry.registerCommand(entry2))
        expect(registry.commandsSnapshot).toEqual([entry1, entry2])

        unregister1.unsubscribe()
        expect(registry.commandsSnapshot).toEqual([entry2])

        unregister2.unsubscribe()
        expect(registry.commandsSnapshot).toEqual([])
    })

    test('refuses to register 2 commands with the same ID', () => {
        const registry = new CommandRegistry()
        registry.registerCommand({ command: 'c', run: () => Promise.resolve() })
        expect(() => {
            registry.registerCommand({ command: 'c', run: () => Promise.resolve() })
        }).toThrow()
    })

    test('runs the specified command', async () => {
        const registry = new CommandRegistry()
        registry.registerCommand({
            command: 'command1',
            // eslint-disable-next-line @typescript-eslint/require-await
            run: async argument => {
                expect(argument).toBe(123)
                return 456
            },
        })
        expect(await registry.executeCommand({ command: 'command1', arguments: [123] })).toBe(456)
    })
})

describe('executeCommand', () => {
    test('runs the specified command with args', async () => {
        const commands: CommandEntry[] = [
            {
                command: 'command1',
                // eslint-disable-next-line @typescript-eslint/require-await
                run: async argument => {
                    expect(argument).toBe(123)
                    return 456
                },
            },
        ]
        expect(await executeCommand(commands, { command: 'command1', arguments: [123] })).toBe(456)
    })

    test('runs the specified command with no args', async () => {
        // eslint-disable-next-line @typescript-eslint/require-await
        const commands: CommandEntry[] = [{ command: 'command1', run: async argument => undefined }]
        expect(await executeCommand(commands, { command: 'command1' })).toBe(undefined)
    })

    test('throws an exception if the command is not found', () => {
        expect(() => executeCommand([], { command: 'c' })).toThrow()
    })
})
