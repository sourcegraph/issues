import { asError } from '../../../../shared/src/util/errors'

export type ResourceDestructor = () => Promise<void>

interface Resource {
    /**
     * Resource type
     */
    type:
        | 'User'
        | 'External service'
        | 'Authentication provider'
        | 'Global setting'
        | 'Organization'
        | 'Configuration'
        | 'LSIF upload'

    /**
     * Name of the resource, printed upon creation and destruction. This should uniquely identify
     * the resource within the resource type. Only the last destructor for duplicate resources will
     * be applied.
     */
    name: string

    /**
     * Destroys the resource.
     */
    destroy: () => Promise<void>
}

/**
 * Tracks resources created by tests for easy resource cleanup at the end of tests. Resources are
 * destroyed in the reverse order in which they're added. Duplicate resources (as identified by type
 * and name) are only destroyed once (subsequent destructors passed to `add` are ignored).
 *
 * Prints which resources are created and destroyed in case tests are aborted midway through and
 * manual cleanup is required.
 */
export class TestResourceManager {
    private resources: Resource[] = []

    public add<T>(type: Resource['type'], name: string, value: { result: T; destroy: () => Promise<void> }): T
    public add(type: Resource['type'], name: string, destroy: () => Promise<void>): void
    public add(type: Resource['type'], name: string, value: any): any {
        if (value.destroy) {
            this.resources.push({ type, name, destroy: value.destroy })
            return value.result
        }
        this.resources.push({ type, name, destroy: value })
    }

    public async destroyAll(): Promise<void> {
        const seen: Record<string, Record<string, boolean>> = {}
        for (const resource of this.resources.reverse()) {
            if (!seen[resource.type]) {
                seen[resource.type] = {}
            }
            if (seen[resource.type][resource.name]) {
                continue
            }
            seen[resource.type][resource.name] = true

            try {
                await resource.destroy()
            } catch (error) {
                console.error(
                    `Error when destroying resource ${resource.type} ${JSON.stringify(resource.name)}: ${
                        asError(error).message
                    }`
                )
                continue
            }
            console.log(`Test resource destroyed: ${resource.type} ${JSON.stringify(resource.name)}`)
        }
    }
}
