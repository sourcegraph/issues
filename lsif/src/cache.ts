import { Connection, createConnection, EntityManager } from 'typeorm'
import { DocumentData, ResultChunkData } from './models.database'
import Yallist from 'yallist'

/**
 * A wrapper around a cache value promise.
 */
interface CacheEntry<K, V> {
    /**
     * The key that can retrieve this cache entry.
     */
    key: K

    /**
     * The promise that will resolve the cache value.
     */
    promise: Promise<V>

    /**
     * The size of the promise value, once resolved. This value is
     * initially zero and is updated once an appropriate can be
     * determined from the result of `promise`.
     */
    size: number

    /**
     * The number of active withValue calls referencing this entry.
     * If this value is non-zero, it should not be evict-able from the
     * cache.
     */
    readers: number
}

/**
 * A generic LRU cache. We use this instead of the `lru-cache` package
 * available in NPM so that we can handle async payloads in a more
 * first-class way as well as shedding some of the cruft around evictions.
 * We need to ensure database handles are closed when they are no longer
 * accessible, and we also do not want to evict any database handle while
 * it is actively being used.
 */
export class GenericCache<K, V> {
    /**
     * A map from from keys to nodes in `lruList`.
     */
    private cache = new Map<K, Yallist.Node<CacheEntry<K, V>>>()

    /**
     * A linked list of cache entires ordered by last-touch.
     */
    private lruList = new Yallist<CacheEntry<K, V>>()

    /**
     * The additive size of the items currently in the cache.
     */
    private size = 0

    /**
     * Create a new `GenericCache` with the given maximum (soft) size for
     * all items in the cache, a function that determine the size of a
     * cache item from its resolved value, and a function that is called
     * when an item falls out of the cache.
     */
    constructor(
        private max: number,
        private sizeFunction: (value: V) => number,
        private disposeFunction: (value: V) => void
    ) {}

    /**
     * Check if `key` exists in the cache. If it does not, create a value
     * from `factory`. Once the cache value resolves, invoke `callback` and
     * return its value. This method acts as a lock around the cache entry
     * so that it may not be removed while the factory or callback functions
     * are running.
     *
     * @param key The cache key.
     * @param factory The function used to create a new value.
     * @param callback The function to invoke with the resolved cache value.
     */
    public async withValue<T>(key: K, factory: () => Promise<V>, callback: (value: V) => Promise<T>): Promise<T> {
        const entry = this.getEntry(key, factory)
        entry.readers++

        try {
            return await callback(await entry.promise)
        } finally {
            entry.readers--
        }
    }

    /**
     * Check if `key` exists in the cache. If it does not, create a value
     * from `factory` and add it to the cache. In either case, update the
     * cache entry's place in `lruCache` and return the entry. If a new
     * value was created, then it may trigger a cache eviction once its
     * value resolves.
     *
     * @param key The cache key.
     * @param factory The function used to create a new value.
     */
    private getEntry(key: K, factory: () => Promise<V>): CacheEntry<K, V> {
        const node = this.cache.get(key)
        if (node) {
            this.lruList.unshiftNode(node)
            return node.value
        }

        const promise = factory()
        const newEntry = { key, promise, size: 0, readers: 0 }
        promise.then(value => this.resolved(newEntry, value), () => {})
        this.lruList.unshift(newEntry)
        const head = this.lruList.head
        if (head) {
            this.cache.set(key, head)
        }

        return newEntry
    }

    /**
     * Determine the size of the resolved value and update the size of the
     * entry as well as `size`. While the total cache size exceeds `max`,
     * try to evict the least recently used cache entries that do not have
     * a non-zero `readers` count.
     *
     * @param entry The cache entry.
     * @param value The cache entry's resolved value.
     */
    private resolved(entry: CacheEntry<K, V>, value: V): void {
        const size = this.sizeFunction(value)
        this.size += size
        entry.size = size

        let node = this.lruList.tail
        while (this.size > this.max && node) {
            const {
                prev,
                value: { promise, size, readers },
            } = node

            if (readers === 0) {
                this.size -= size
                this.lruList.removeNode(node)
                this.cache.delete(node.value.key)
                promise.then(value => this.disposeFunction(value), () => {})
            }

            node = prev
        }
    }
}

/**
 * A cache of SQLite database connections indexed by database filenames.
 */
export class ConnectionCache extends GenericCache<string, Connection> {
    /**
     * Create a new `ConnectionCache` with the given maximum (soft) size for
     * all items in the cache.
     */
    constructor(max: number) {
        super(
            max,
            // Each handle is roughly the same size.
            () => 1,
            // Close the underlying file handle on cache eviction.
            (connection: Connection) => connection.close()
        )
    }

    /**
     * Invoke `callback` with a SQLite connection object obtained from the
     * cache or created on cache miss. This connection is guaranteed not to
     * be disposed by cache eviction while the callback is active.
     *
     * @param database The database filename.
     * @param entities The set of expected entities present in this schema.
     * @param callback The function invoke with the SQLite connection.
     */
    public withConnection<T>(
        database: string,
        entities: Function[], // eslint-disable-line @typescript-eslint/ban-types
        callback: (connection: Connection) => Promise<T>
    ): Promise<T> {
        const factory = (): Promise<Connection> =>
            createConnection({
                database,
                entities,
                type: 'sqlite',
                name: database,
                synchronize: true,
                // logging: 'all',
            })

        return this.withValue(database, factory, callback)
    }

    /**
     * Like `withConnection`, but will open a transaction on the connection
     * before invoking the callback.
     *
     * @param database The database filename.
     * @param entities The set of expected entities present in this schema.
     * @param callback The function invoke with a SQLite transaction connection.
     */
    public withTransactionalEntityManager<T>(
        database: string,
        entities: Function[], // eslint-disable-line @typescript-eslint/ban-types
        callback: (entityManager: EntityManager) => Promise<T>
    ): Promise<T> {
        return this.withConnection(database, entities, async connection => {
            // TODO - enable with flag
            await connection.query('PRAGMA synchronous = OFF')
            await connection.query('PRAGMA journal_mode = OFF')

            return await connection.transaction(em => callback(em))
        })
    }
}

/**
 * A wrapper around a cache value that retains its encoded size. In order to keep
 * the in-memory limit of these decoded items, we use this value as the cache entry
 * size. This assumes that the size of the encoded text is a good proxy for the size
 * of the in-memory representation.
 */
export interface EncodedJsonCacheValue<T> {
    /**
     * The size of the encoded value.
     */
    size: number

    /**
     * The decoded value.
     */
    data: T
}

/**
 * A cache of decoded values encoded as JSON and gzipped in a SQLite database.
 */
class EncodedJsonCache<K, V> extends GenericCache<K, EncodedJsonCacheValue<V>> {
    /**
     * Create a new `EncodedJsonCache` with the given maximum (soft) size for
     * all items in the cache.
     */
    constructor(max: number) {
        super(
            max,
            // TODO - determine memory size
            v => v.size,
            // Let GC handle the cleanup of the object on cache eviction.
            (): void => {}
        )
    }
}

/**
 * A cache of deserialized `DocumentData` values indexed by a string containing
 * the database path and the path of the document.
 */
export class DocumentCache extends EncodedJsonCache<string, DocumentData> {
    /**
     * Create a new `DocumentCache` with the given maximum (soft) size for
     * all items in the cache.
     */
    constructor(max: number) {
        super(max)
    }
}

/**
 * A cache of deserialized `ResultChunkData` values indexed by a string containing
 * the database path and the chunk index.
 */
export class ResultChunkCache extends EncodedJsonCache<string, ResultChunkData> {
    /**
     * Create a new `ResultChunkCache` with the given maximum (soft) size for
     * all items in the cache.
     */
    constructor(max: number) {
        super(max)
    }
}
