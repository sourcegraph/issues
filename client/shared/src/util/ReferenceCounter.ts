/**
 * A Map that tracks reference counts for keys.
 */
export class ReferenceCounter<T> implements Pick<Set<T>, 'keys' | 'delete'> {
    private refCount = new Map<T, number>()

    /**
     * Increment the refCount for the given key.
     *
     * @returns true if the given key is new.
     */
    public increment(key: T): boolean {
        const current = this.refCount.get(key)
        if (current === undefined) {
            this.refCount.set(key, 1)
            return true
        }
        this.refCount.set(key, current + 1)
        return false
    }

    /**
     * Decrements the refCount for the given key.
     *
     * @returns true if this was the given key's last reference.
     */
    public decrement(key: T): boolean {
        const current = this.refCount.get(key)
        if (current === undefined) {
            throw new Error(`No refCount for key: ${String(key)}`)
        } else if (current === 1) {
            this.refCount.delete(key)
            return true
        } else {
            this.refCount.set(key, current - 1)
            return false
        }
    }

    /**
     * Returns an iterable of registered keys.
     */
    public keys(): IterableIterator<T> {
        return this.refCount.keys()
    }

    public delete(key: T): boolean {
        return this.refCount.delete(key)
    }
}
