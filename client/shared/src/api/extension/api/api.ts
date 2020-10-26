import { ProxyMarked } from 'comlink'
import { InitData } from '../extensionHost'
<<<<<<< HEAD:shared/src/api/extension/api/api.ts
=======
import { ExtensionDocumentsAPI } from './documents'
import { ExtensionExtensionsAPI } from './extensions'
>>>>>>> main:client/shared/src/api/extension/api/api.ts
import { ExtensionWindowsAPI } from './windows'
import { FlatExtensionHostAPI } from '../../contract'

export type ExtensionHostAPIFactory = (initData: InitData) => ExtensionHostAPI

export interface ExtensionHostAPI extends ProxyMarked, FlatExtensionHostAPI {
    ping(): 'pong'

<<<<<<< HEAD:shared/src/api/extension/api/api.ts
=======
    documents: ExtensionDocumentsAPI
    extensions: ExtensionExtensionsAPI
>>>>>>> main:client/shared/src/api/extension/api/api.ts
    windows: ExtensionWindowsAPI
}
