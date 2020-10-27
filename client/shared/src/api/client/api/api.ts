import { ClientContentAPI } from './content'
import { ClientContextAPI } from './context'
import { ClientViewsAPI } from './views'
import { ClientWindowsAPI } from './windows'
import { MainThreadAPI } from '../../contract'

/**
 * The API that is exposed from the client (main thread) to the extension host (worker)
 */
export interface ClientAPI extends MainThreadAPI {
    ping(): 'pong'

    context: ClientContextAPI
    windows: ClientWindowsAPI
    views: ClientViewsAPI
    content: ClientContentAPI
}
