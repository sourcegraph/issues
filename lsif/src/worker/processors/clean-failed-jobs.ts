import * as constants from '../../constants'
import * as fs from 'mz/fs'
import * as path from 'path'
import { FAILED_JOB_MAX_AGE, STORAGE_ROOT } from '../settings'
import { logAndTraceCall, TracingContext } from '../../shared/tracing'

/**
 * Create a job that removes upload and temp files that are older than `FAILED_JOB_MAX_AGE`.
 * This assumes that a conversion job's total duration (from enqueue to completion) is less
 * than this interval during healthy operation.
 */
export const createCleanFailedJobsProcessor = () => async (_: {}, ctx: TracingContext): Promise<void> => {
    await logAndTraceCall(ctx, 'cleaning failed jobs', async (ctx: TracingContext) => {
        const purgeFile = async (filename: string): Promise<void> => {
            const stat = await fs.stat(filename)
            if (Date.now() - stat.mtimeMs >= FAILED_JOB_MAX_AGE) {
                await fs.unlink(filename)
            }
        }

        for (const directory of [constants.TEMP_DIR, constants.UPLOADS_DIR]) {
            for (const filename of await fs.readdir(path.join(STORAGE_ROOT, directory))) {
                await purgeFile(path.join(STORAGE_ROOT, directory, filename))
            }
        }
    })
}
