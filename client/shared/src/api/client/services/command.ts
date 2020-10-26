import { BehaviorSubject, Observable, Unsubscribable } from 'rxjs'

/** A registered command in the command registry. */
export interface CommandEntry {
    /** The command ID (conventionally, e.g., "myextension.mycommand"). */
    command: string

    /** The function called to run the command and return an async value. */
    run: (...args: any[]) => Promise<any>
}

export interface ExecuteCommandParameters {
    /**
     * The identifier of the actual command handler.
     */
    command: string

    /**
     * Arguments that the command should be invoked with.
     */
    arguments?: any[]
}

/** Manages and executes commands from all extensions. */
export class CommandRegistry {
    private entries = new BehaviorSubject<CommandEntry[]>([])

    public registerCommand(entryToRegister: CommandEntry): Unsubscribable {
        // Enforce uniqueness of command IDs.
        for (const entry of this.entries.value) {
            if (entry.command === entryToRegister.command) {
                throw new Error(`command is already registered: ${JSON.stringify(entry.command)}`)
            }
        }

        this.entries.next([...this.entries.value, entryToRegister])
        return {
            unsubscribe: () => {
                this.entries.next(this.entries.value.filter(entry => entry !== entryToRegister))
            },
        }
    }

    public executeCommand(parameters: ExecuteCommandParameters): Promise<any> {
        return executeCommand(this.commandsSnapshot, parameters)
    }

    /** All commands, emitted whenever the set of registered commands changed. */
    public readonly commands: Observable<CommandEntry[]> = this.entries

    /**
     * The current set of commands. Used by callers that do not need to react to commands being registered or
     * unregistered.
     */
    public get commandsSnapshot(): CommandEntry[] {
        return this.entries.value
    }
}

/**
 * Executes the command (in the commands list) specified in params.
 *
 * Most callers should use CommandRegistry's executeCommand method, which uses the registered commands.
 */
export function executeCommand(commands: CommandEntry[], parameters: ExecuteCommandParameters): Promise<any> {
    const command = commands.find(entry => entry.command === parameters.command)
    if (!command) {
        throw new Error(`command not found: ${JSON.stringify(parameters.command)}`)
    }
    return Promise.resolve(command.run(...(parameters.arguments || [])))
}
