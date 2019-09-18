import * as definitionsSchema from './lsif.schema.json';
import * as fs from 'mz/fs';
import * as path from 'path';
import Ajv from 'ajv';
import bodyParser from 'body-parser';
import exitHook from 'async-exit-hook';
import express from 'express';
import onFinished from 'on-finished';
import onHeaders from 'on-headers';
import promClient from 'prom-client';
import uuid from 'uuid';
import { ConnectionCache, DocumentCache, ResultChunkCache } from './cache';
import {
    createDatabaseFilename,
    ensureDirectory,
    hasErrorCode,
    readEnvInt
} from './util';
import { createLogger } from './logger';
import { Database } from './database.js';
import { Logger } from 'winston';
import { Queue, rewriteJobMeta, WorkerMeta } from './queue';
import { Queue as ResqueQueue, Scheduler } from 'node-resque';
import { validateLsifInput } from './input';
import { wrap } from 'async-middleware';
import { XrepoDatabase } from './xrepo.js';
import {
    connectionCacheCapacityGauge,
    documentCacheCapacityGauge,
    httpQueryDurationHistogram,
    httpUploadDurationHistogram,
    resultChunkCacheCapacityGauge,
    queueSizeGauge,
} from './metrics'

/**
 * Which port to run the LSIF server on. Defaults to 3186.
 */
const HTTP_PORT = readEnvInt('HTTP_PORT', 3186)

/**
 * The host and port running the redis instance containing work queues.
 *
 * Set addresses. Prefer in this order:
 *   - Specific envvar REDIS_STORE_ENDPOINT
 *   - Fallback envvar REDIS_ENDPOINT
 *   - redis-store:6379
 *
 *  Additionally keep this logic in sync with pkg/redispool/redispool.go and cmd/server/redis.go
 */
const REDIS_ENDPOINT = process.env.REDIS_STORE_ENDPOINT || process.env.REDIS_ENDPOINT || 'redis-store:6379'

/**
 * The number of SQLite connections that can be opened at once. This
 * value may be exceeded for a short period if many handles are held
 * at once.
 */
const CONNECTION_CACHE_CAPACITY = readEnvInt('CONNECTION_CACHE_CAPACITY', 100)

/**
 * The maximum number of documents that can be held in memory at once.
 */
const DOCUMENT_CACHE_CAPACITY = readEnvInt('DOCUMENT_CACHE_CAPACITY', 1024 * 1024 * 1024)

/**
 * The maximum number of result chunks that can be held in memory at once.
 */
const RESULT_CHUNK_CACHE_CAPACITY = readEnvInt('RESULT_CHUNK_CACHE_CAPACITY', 1024 * 1024 * 1024)

/**
 * Where on the file system to store LSIF files.
 */
const STORAGE_ROOT = process.env.LSIF_STORAGE_ROOT || 'lsif-storage'

/**
 * Whether or not to disable input validation. Validation is enabled by default.
 */
const DISABLE_VALIDATION = process.env.DISABLE_VALIDATION === 'true'

/**
 * Runs the HTTP server which accepts LSIF dump uploads and responds to LSIF requests.
 *
 * @param logger The application logger instance.
 */
async function main(logger: Logger): Promise<void> {
    // Collect process metrics
    promClient.collectDefaultMetrics({ prefix: 'lsif_' })

    // Update cache capacities on startup
    connectionCacheCapacityGauge.set(CONNECTION_CACHE_CAPACITY)
    documentCacheCapacityGauge.set(DOCUMENT_CACHE_CAPACITY)
    resultChunkCacheCapacityGauge.set(RESULT_CHUNK_CACHE_CAPACITY)

    // Ensure storage roots exist
    await ensureDirectory(STORAGE_ROOT)
    await ensureDirectory(path.join(STORAGE_ROOT, 'tmp'))
    await ensureDirectory(path.join(STORAGE_ROOT, 'uploads'))

    // Create queue to publish jobs for worker
    const queue = await setupQueue(logger)

    const app = express()
    app.use(loggingMiddleware(logger))
    app.use(metricsMiddleware)

    // Register endpoints
    app.use(metaEndpoints())
    app.use(lsifEndpoints(queue, logger))
    app.use(queueEndpoints(queue, logger))

    // Error handler must be registered last
    app.use(errorHandler(logger))

    app.listen(HTTP_PORT, () => logger.debug('listening', { port: HTTP_PORT }))
}

/**
 * Connect and start an active connection to the worker queue. We also run a
 * node-resque scheduler on each server instance, as these are guaranteed to
 * always be up with a responsive system. The schedulers will do their own
 * master election via a redis key and will check for dead workers attached
 * to the queue.
 *
 * @param logger The server's logger instance.
 */
async function setupQueue(logger: Logger): Promise<Queue> {
    const [host, port] = REDIS_ENDPOINT.split(':', 2)

    const connectionOptions = {
        host,
        port: parseInt(port, 10),
        namespace: 'lsif',
    }

    // Create queue and log the interesting events
    const queue = new ResqueQueue({ connection: connectionOptions }) as Queue
    queue.on('error', e => logger.error('queue error', { error: e }))
    await queue.connect()
    exitHook(() => queue.end())

    const emitQueueSizeMetric = (): void => {
        queue
            .queued('lsif', 0, -1)
            .then(jobs => queueSizeGauge.set(jobs.length))
            .catch(e => logger.error('failed to get queued jobs', { error: e && e.message }))
    }

    // Update queue size metric on a timer
    setInterval(emitQueueSizeMetric, 1000)

    // Create scheduler log the interesting events
    const scheduler = new Scheduler({ connection: connectionOptions })
    scheduler.on('start', () => logger.debug('scheduler started'))
    scheduler.on('end', () => logger.debug('scheduler ended'))
    scheduler.on('poll', () => logger.debug('scheduler checking for stuck workers'))
    scheduler.on('master', () => logger.debug('scheduler became master'))
    scheduler.on('cleanStuckWorker', worker => logger.debug('scheduler cleaning stuck worker', { worker }))
    scheduler.on('transferredJob', (_, job) => logger.debug('scheduler transferring job', { job }))
    scheduler.on('error', e => logger.error('scheduler error', { error: e }))

    await scheduler.connect()
    exitHook(() => scheduler.end())
    await scheduler.start()

    return queue
}

/**
 * Create a router containing health and metrics endpoints.
 */
function metaEndpoints(): express.Router {
    const router = express.Router()
    router.get('/healthz', (_, res) => res.send('ok'))
    router.get('/metrics', (_, res) => {
        res.writeHead(200, { 'Content-Type': 'text/plain' })
        res.end(promClient.register.metrics())
    })

    return router
}

/**
 * Create a router containing the LSIF upload and query endpoints.
 *
 * @param queue The queue containing LSIF jobs.
 * @param logger The server's logger instance.
 */
function lsifEndpoints(queue: Queue, logger: Logger): express.Router {
    const router = express.Router()

    // Create cross-repo database
    const connectionCache = new ConnectionCache(CONNECTION_CACHE_CAPACITY)
    const documentCache = new DocumentCache(DOCUMENT_CACHE_CAPACITY)
    const resultChunkCache = new ResultChunkCache(RESULT_CHUNK_CACHE_CAPACITY)
    const filename = path.join(STORAGE_ROOT, 'xrepo.db')
    const xrepoDatabase = new XrepoDatabase(connectionCache, filename)

    // Compile the JSON schema used for validation
    const validator = new Ajv().addSchema({ $id: 'defs.json', ...definitionsSchema }).compile({
        oneOf: [{ $ref: 'defs.json#/definitions/Vertex' }, { $ref: 'defs.json#/definitions/Edge' }],
    })

    // Factory function to open a database for a given repository/commit
    const loadDatabase = async (repository: string, commit: string): Promise<Database | undefined> => {
        const file = createDatabaseFilename(STORAGE_ROOT, repository, commit)

        try {
            await fs.stat(file)
        } catch (e) {
            if (hasErrorCode(e, 'ENOENT')) {
                return undefined
            }

            throw e
        }

        return new Database(
            STORAGE_ROOT,
            xrepoDatabase,
            connectionCache,
            documentCache,
            resultChunkCache,
            repository,
            commit,
            file
        )
    }

    router.post(
        '/upload',
        wrap(
            async (req: express.Request, res: express.Response, next: express.NextFunction): Promise<void> => {
                const { repository, commit } = req.query
                checkRepository(repository)
                checkCommit(commit)

                const filename = path.join(STORAGE_ROOT, 'uploads', uuid.v4())
                const output = fs.createWriteStream(filename)

                try {
                    await validateLsifInput(req, output, DISABLE_VALIDATION ? undefined : validator)
                } catch (e) {
                    throw Object.assign(e, { status: 422 })
                }

                // Enqueue convert job
                logger.info('enqueueing conversion job', { repository, commit })
                await queue.enqueue('lsif', 'convert', [repository, commit, filename])
                res.status(204).send()
            }
        )
    )

    router.post(
        '/exists',
        wrap(
            async (req: express.Request, res: express.Response): Promise<void> => {
                const { repository, commit, file } = req.query
                checkRepository(repository)
                checkCommit(commit)

                const db = await loadDatabase(repository, commit)
                if (!db) {
                    res.json(false)
                    return
                }

                // If filename supplied, ensure we have data for it
                const result = file ? await db.exists(file) : true
                res.json(result)
            }
        )
    )

    router.post(
        '/request',
        bodyParser.json({ limit: '1mb' }),
        wrap(
            async (req: express.Request, res: express.Response): Promise<void> => {
                const { repository, commit } = req.query
                const { path, position, method } = req.body
                checkRepository(repository)
                checkCommit(commit)
                checkMethod(method, ['definitions', 'references', 'hover'])
                const cleanMethod = method as 'definitions' | 'references' | 'hover'

                const db = await loadDatabase(repository, commit)
                if (!db) {
                    throw Object.assign(new Error(`No LSIF data available for ${repository}@${commit}.`), {
                        status: 404,
                    })
                }

                res.json(await db[cleanMethod](path, position))
            }
        )
    )

    return router
}

/**
 * Add endpoints to the HTTP API to view/control the worker queue.
 *
 * @param queue The queue containing LSIF jobs.
 * @param logger The server's logger instance.
 */
function queueEndpoints(queue: Queue, logger: Logger): express.Router {
    const router = express.Router()

    router.get(
        '/queued',
        wrap(
            async (req: express.Request, res: express.Response): Promise<void> => {
                const queuedJobs = await queue.queued('lsif', 0, -1)

                res.send(
                    queuedJobs.map(job => ({
                        ...rewriteJobMeta(job),
                    }))
                )
            }
        )
    )

    router.get(
        '/failed',
        wrap(
            async (req: express.Request, res: express.Response): Promise<void> => {
                const failedJobs = await queue.failed(0, -1)
                failedJobs.sort((a, b) => a.failed_at.localeCompare(b.failed_at))

                res.send(
                    failedJobs.map(job => ({
                        error: job.error,
                        failed_at: new Date(job.failed_at).toISOString(),
                        ...rewriteJobMeta(job.payload),
                    }))
                )
            }
        )
    )

    router.get(
        '/active',
        wrap(
            async (req: express.Request, res: express.Response): Promise<void> => {
                const workerMeta = Array.from(Object.values(await queue.allWorkingOn())).filter(
                    (x): x is WorkerMeta => x !== 'started'
                )
                workerMeta.sort((a, b) => a.run_at.localeCompare(b.run_at))

                res.send(
                    workerMeta.map(job => ({
                        started_at: new Date(job.run_at).toISOString(),
                        ...rewriteJobMeta(job.payload),
                    }))
                )
            }
        )
    )

    return router
}

/**
 * A pair of seconds and nanoseconds representing the output of
 * the nodejs high-resolution timer.
 */
type HrTime = [number, number]

/**
 * Middleware function used to log requests and the corresponding
 * response status code and wall time taken to process the request
 * (to the point where headers are emitted).
 *
 * @param logger The server's logger instance.
 */
function loggingMiddleware(
    logger: Logger
): (req: express.Request, res: express.Response, next: express.NextFunction) => void {
    return (req: express.Request, res: express.Response, next: express.NextFunction): void => {
        const start = process.hrtime()
        let end: HrTime | undefined

        onHeaders(res, () => {
            end = process.hrtime()
        })

        onFinished(res, () => {
            const responseTime = end ? `${((end[0] - start[0]) * 1e3 + (end[1] - start[1]) * 1e-6).toFixed(3)}ms` : ''

            logger.debug('request', {
                method: req.method,
                path: req.path,
                statusCode: res.statusCode,
                responseTime,
            })
        })

        next()
    }
}

/**
 * Middleware function used to emit HTTP durations for LSIF functions. Originally
 * we used an express bundle, but that did not allow us to have different histogram
 * bucket for different endpoints, which makes half of the metrics useless in the
 * presence of large uploads.
 */
function metricsMiddleware(req: express.Request, res: express.Response, next: express.NextFunction): void {
    let histogram: promClient.Histogram | undefined
    switch (req.path) {
        case '/upload':
            histogram = httpUploadDurationHistogram
            break

        case '/exists':
        case '/request':
            histogram = httpQueryDurationHistogram
    }

    if (histogram !== undefined) {
        const labels = { code: 0 }
        const end = histogram.startTimer(labels)

        onFinished(res, () => {
            labels.code = res.statusCode
            end()
        })
    }

    next()
}

/**
 * Middleware function used to convert uncaught exceptions into 500 responses.
 *
 * @param logger The server's logger instance.
 */
function errorHandler(
    logger: Logger
): (e: any, req: express.Request, res: express.Response, next: express.NextFunction) => void {
    return (e: any, req: express.Request, res: express.Response, next: express.NextFunction): void => {
        if (res.headersSent) {
            return next(e)
        }

        if (e && e.status) {
            res.status(e.status).send({ message: e.message })
            return
        }

        logger.error('uncaught exception', { error: e })
        res.status(500).send({ message: 'Unknown error' })
    }
}

/**
 * Throws an error with status 400 if the repository string is invalid.
 */
export function checkRepository(repository: any): void {
    if (typeof repository !== 'string') {
        throw Object.assign(new Error('Must specify the repository (usually of the form github.com/user/repo)'), {
            status: 400,
        })
    }
}

/**
 * Throws an error with status 400 if the commit string is invalid.
 */
export function checkCommit(commit: any): void {
    if (typeof commit !== 'string' || commit.length !== 40 || !/^[0-9a-f]+$/.test(commit)) {
        throw Object.assign(new Error('Must specify the commit as a 40 character hash ' + commit), { status: 400 })
    }
}

/**
 * Throws an error with status 422 if the requested method is not supported.
 */
export function checkMethod(method: string, supportedMethods: string[]): void {
    if (!supportedMethods.includes(method)) {
        throw Object.assign(new Error(`Method must be one of ${Array.from(supportedMethods).join(', ')}`), {
            status: 422,
        })
    }
}

// Initialize logger
const appLogger = createLogger('lsif-server')

// Run app!
main(appLogger).catch(e => appLogger.error('failed to start process', { error: e }))
