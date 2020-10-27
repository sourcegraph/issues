import * as sourcegraph from 'sourcegraph'
import { DirectoryViewerData, ViewerId } from '../../viewerTypes'

export class ExtensionDirectoryViewer implements sourcegraph.DirectoryViewer {
    public readonly viewerId: string
    public readonly type = 'DirectoryViewer'
    public isActive: boolean
    public directory: sourcegraph.Directory
    constructor(data: DirectoryViewerData & ViewerId) {
        this.isActive = data.isActive
        this.viewerId = data.viewerId
        // Since directories don't have any state beyond the immutable URI,
        // we can set the model to a static object for now and don't need to track directory models in a Map.
        this.directory = {
            uri: new URL(data.resource),
        }
    }
}
