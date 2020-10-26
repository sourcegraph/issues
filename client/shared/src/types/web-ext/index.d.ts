export namespace cmd {
    export interface RunParams {
        sourceDir: string
        firefox: string
        args?: string[]
    }
    export interface RunOptions {
        shouldExitProgram?: boolean
    }

    /**
     * Launches Firefox with an extension loaded.
     */
    export function run(params: RunParams, options: RunOptions): Promise<void>
}
