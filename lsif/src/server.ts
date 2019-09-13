import * as definitionsSchema from './lsif.schema.json'
import * as es from 'event-stream'
import * as fs from 'mz/fs'
import * as path from 'path'
import * as zlib from 'mz/zlib'
import Ajv, { ValidateFunction } from 'ajv'
import bodyParser from 'body-parser'
import exitHook from 'async-exit-hook'
import express from 'express'
import morgan from 'morgan'
import promBundle from 'express-prom-bundle'
import uuid from 'uuid'
import { ConnectionCache, DocumentCache, ResultChunkCache } from './cache'
import { createDirectory, hasErrorCode, logErrorAndExit, readEnvInt, createDatabaseFilename } from './util'
import { Queue, Scheduler, Job } from 'node-resque'
import { wrap } from 'async-middleware'
import {
    CONNECTION_CACHE_CAPACITY_GAUGE,
    DOCUMENT_CACHE_CAPACITY_GAUGE,
    RESULT_CHUNK_CACHE_CAPACITY_GAUGE,
} from './metrics'
import { XrepoDatabase } from './xrepo.js'
import { Database } from './database.js'

/**
 * Which port to run the LSIF server on. Defaults to 3186.
 */
const HTTP_PORT = readEnvInt('HTTP_PORT', 3186)

/**
 * The host running the redis instance containing work queues. Defaults to localhost.
 */
const REDIS_HOST = process.env.REDIS_HOST || 'localhost'

/**
 * The port of the redis instance containing work queues. Defaults to 6379.
 */
const REDIS_PORT = readEnvInt('REDIS_PORT', 6379)

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
 * Whether or not to log a message when the HTTP server is ready and listening.
 */
const LOG_READY = process.env.DEPLOY_TYPE === 'dev'

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
 */
async function main(): Promise<void> {
    // Update cache capacities on startup
    CONNECTION_CACHE_CAPACITY_GAUGE.set(CONNECTION_CACHE_CAPACITY)
    DOCUMENT_CACHE_CAPACITY_GAUGE.set(DOCUMENT_CACHE_CAPACITY)
    RESULT_CHUNK_CACHE_CAPACITY_GAUGE.set(RESULT_CHUNK_CACHE_CAPACITY)

    // Ensure storage roots exist
    await createDirectory(STORAGE_ROOT)
    await createDirectory(path.join(STORAGE_ROOT, 'tmp'))
    await createDirectory(path.join(STORAGE_ROOT, 'uploads'))

    // Create cross-repos database
    const connectionCache = new ConnectionCache(CONNECTION_CACHE_CAPACITY)
    const documentCache = new DocumentCache(DOCUMENT_CACHE_CAPACITY)
    const resultChunkCache = new ResultChunkCache(RESULT_CHUNK_CACHE_CAPACITY)
    const filename = path.join(STORAGE_ROOT, 'xrepo.db')
    const xrepoDatabase = new XrepoDatabase(connectionCache, filename)

    const createDatabase = async (repository: string, commit: string): Promise<Database | undefined> => {
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

    // Create queue to publish jobs for worker
    const queue = await setupQueue()

    // Compile the JSON schema used for validation
    const validator = createInputValidator()

    const app = express()
    app.use(morgan('tiny'))
    app.use(errorHandler)
    app.get('/healthz', (_, res) => res.send('ok'))
    app.use(promBundle({}))

    app.post(
        '/upload',
        wrap(
            async (req: express.Request, res: express.Response, next: express.NextFunction): Promise<void> => {
                const { repository, commit } = req.query
                checkRepository(repository)
                checkCommit(commit)

                let line = 0
                const filename = path.join(STORAGE_ROOT, 'uploads', uuid.v4())
                const writeStream = fs.createWriteStream(filename)

                const throwValidationError = (text: string, errorText: string): never => {
                    throw Object.assign(
                        new Error(`Failed to process line #${line + 1} (${JSON.stringify(text)}): ${errorText}.`),
                        { status: 422 }
                    )
                }

                const validateLine = (text: string): void => {
                    if (text === '' || DISABLE_VALIDATION) {
                        return
                    }

                    let data: any

                    try {
                        data = JSON.parse(text)
                    } catch (e) {
                        throwValidationError(text, 'Invalid JSON')
                    }

                    if (!validator(data)) {
                        throwValidationError(text, 'Does not match a known vertex or edge shape')
                    }

                    line++
                }

                await new Promise((resolve, reject) => {
                    req.pipe(zlib.createGunzip()) // unzip input
                        .pipe(es.split()) // split into lines
                        .pipe(
                            // Must check each line synchronously
                            // eslint-disable-next-line no-sync
                            es.mapSync((text: string) => {
                                validateLine(text) // validate seach line
                                return text
                            })
                        )
                        .on('error', reject) // catch validation error
                        .pipe(es.join('\n')) // join back into text
                        .pipe(zlib.createGzip()) // re-zip input
                        .pipe(writeStream) // write to temp file
                        .on('finish', resolve) // unblock promise when done
                })

                console.log(`Enqueueing conversion job for ${repository}@${commit}`)
                await queue.enqueue('lsif', 'convert', [repository, commit, filename])
                res.json(null)
            }
        )
    )

    app.post(
        '/exists',
        wrap(
            async (req: express.Request, res: express.Response): Promise<void> => {
                const { repository, commit, file } = req.query
                checkRepository(repository)
                checkCommit(commit)

                const db = await createDatabase(repository, commit)
                if (!db) {
                    res.json(false)
                    return
                }

                const result = !file || (await db.exists(file))
                res.json(result)
            }
        )
    )

    app.post(
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

                const db = await createDatabase(repository, commit)
                if (!db) {
                    throw Object.assign(new Error(`No LSIF data available for ${repository}@${commit}.`), {
                        status: 404,
                    })
                }

                res.json(await db[cleanMethod](path, position))
            }
        )
    )

    app.listen(HTTP_PORT, () => {
        if (LOG_READY) {
            console.log(`Listening for HTTP requests on port ${HTTP_PORT}`)
        }
    })
}

/**
 * Connect and start an active connection to the worker queue. We also run a
 * node-resque scheduler on each server instance, as these are guaranteed to
 * always be up with a responsive system. The schedulers will do their own
 * master election via a redis key and will check for dead workers attached
 * to the queue.
 */
async function setupQueue(): Promise<Queue> {
    const connectionOptions = {
        host: REDIS_HOST,
        port: REDIS_PORT,
        namespace: 'lsif',
    }

    const queue = new Queue({ connection: connectionOptions })
    queue.on('error', logErrorAndExit)
    await queue.connect()
    exitHook(() => queue.end())

    const inspectMetrics = async () => {
        // queued [
        //  {
        //    class: 'convert',
        //    queue: 'lsif',
        //    args: [
        //      'github.com/sourcegraph/codeintellify',
        //      '1234567890123456789012345678901234567890',
        //      '/Users/efritz/.sourcegraph/lsif-storage/uploads/aeb5bdf6-8dcd-4b70-b6bf-d78e447f01bb'
        //    ]
        //  }
        // ]

        // console.log('workers', await (queue as any).workers())
        // console.log('stats', await (queue as any).stats())
        // console.log('failedCount', await (queue as any).failedCount())

        // console.log('queued', await (queue as any).queued('lsif', 0, -1))
        // console.log('failed', await (queue as any).failed(0, -1))

        // console.log(await (queue as any).cleanOldWorkers(10))
        // console.log('allWorkingOn', await (queue as any).allWorkingOn())

        // let stats = await queue.stats()
        // let jobs = await queue.queued(q, start, stop)
        // let length = await queue.length(q)

        const data = await (queue as any).allWorkingOn()
        for (const i in Object.keys(data)) {
            const workerName = Object.keys(data)[i]
            console.log(data[workerName])
        }
    }

    setInterval(() => { inspectMetrics().catch(logErrorAndExit) }, 1000)

    const scheduler = new Scheduler({ connection: connectionOptions })
    scheduler.on('start', () => console.log('Scheduler started'))
    scheduler.on('end', () => console.log('Scheduler ended'))
    scheduler.on('poll', () => console.log('Scheduler polling'))
    scheduler.on('master', () => console.log('Scheduler has become master'))
    scheduler.on('cleanStuckWorker', (workerName: string) => console.log(`Cleaning stuck worker ${workerName}`))
    scheduler.on('transferredJob', (_: number, job: Job<any>) => console.log(`Transferring job ${JSON.stringify(job)}`))
    scheduler.on('error', logErrorAndExit)

    await scheduler.connect()
    exitHook(() => scheduler.end())
    scheduler.start().catch(logErrorAndExit)

    const inspectMetrics = async () => {
        // queued [
        //  {
        //    class: 'convert',
        //    queue: 'lsif',
        //    args: [
        //      'github.com/sourcegraph/codeintellify',
        //      '1234567890123456789012345678901234567890',
        //      '/Users/efritz/.sourcegraph/lsif-storage/uploads/aeb5bdf6-8dcd-4b70-b6bf-d78e447f01bb'
        //    ]
        //  }
        // ]

        // console.log('workers', await (queue as any).workers())
        // console.log('stats', await (queue as any).stats())
        // console.log('failedCount', await (queue as any).failedCount())

        // console.log('queued', await (queue as any).queued('lsif', 0, -1))
        // console.log('failed', await (queue as any).failed(0, -1))

        // console.log(await (queue as any).cleanOldWorkers(10))
        // console.log('allWorkingOn', await (queue as any).allWorkingOn())

        // let stats = await queue.stats()
        // let jobs = await queue.queued(q, start, stop)
        // let length = await queue.length(q)

        const data = await (queue as any).allWorkingOn()
        for (const workerName of Object.keys(data)) {
            console.log(data[workerName])
        }
    }

    setInterval(() => {
        inspectMetrics().catch(logErrorAndExit)
    }, 1000)

    return queue
}

/**
 * Create a json schema validation function that can validate each line of an
 * LSIF dump input.
 */
function createInputValidator(): ValidateFunction {
    // Compile schema with defs as a reference
    return new Ajv().addSchema({ $id: 'defs.json', ...definitionsSchema }).compile({
        oneOf: [{ $ref: 'defs.json#/definitions/Vertex' }, { $ref: 'defs.json#/definitions/Edge' }],
    })
}

/**
 * Middleware function used to convert uncaught exceptions into 500 responses.
 */
function errorHandler(err: any, req: express.Request, res: express.Response, next: express.NextFunction): void {
    if (err && err.status) {
        res.status(err.status).send({ message: err.message })
        return
    }

    console.error(err)
    res.status(500).send({ message: 'Unknown error' })
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

main().catch(logErrorAndExit)
