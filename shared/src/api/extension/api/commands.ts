import * as comlink from 'comlink'
import { Unsubscribable } from 'rxjs'
import { ClientCommandsAPI } from '../../client/api/commands'
import { synchronousSubscription } from '../../util'

interface CommandEntry {
    command: string
    callback: (...args: any[]) => any
}

/** @internal */
export class ExtCommands {
    constructor(private proxy: comlink.Remote<ClientCommandsAPI>) {}

    /**
     * Extension API method invoked directly when an extension wants to execute a command. It calls to the client
     * to execute the command because the desired command might be implemented on the client (or otherwise not in
     * this extension host).
     */
    public executeCommand(command: string, args: any[]): Promise<any> {
        return this.proxy.$executeCommand(command, args)
    }

    public registerCommand(entry: CommandEntry): Unsubscribable {
        return synchronousSubscription(this.proxy.$registerCommand(entry.command, comlink.proxy(entry.callback)))
    }
}
