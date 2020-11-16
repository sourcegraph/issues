import * as sourcegraph from 'sourcegraph'

export class WorkspaceRoot implements sourcegraph.WorkspaceRoot {
    constructor(public uri: URL, public inputRevision?: string) {}
}
