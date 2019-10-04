import { createLogger as _createLogger, Logger, transports } from 'winston'
import { MESSAGE } from 'triple-beam'
import { TransformableInfo, format } from 'logform'
import { inspect } from 'util'

/**
 * The maximum level log message to output.
 */
const LOG_LEVEL = process.env.LOG_LEVEL || 'info'

/**
 * Create a structured logger.
 */
export function createLogger(service: string): Logger {
    const formatTransformer = (info: TransformableInfo): TransformableInfo => {
        const attributes: { [k: string]: any } = {}
        for (const [key, value] of Object.entries(info)) {
            if (key !== 'level' && key !== 'message') {
                attributes[key] = value
            }
        }

        info[MESSAGE] = `${info.level} ${info.message} ${inspect(attributes)}`
        return info
    }

    const uppercaseTransformer = (info: TransformableInfo): TransformableInfo => {
        info.level = info.level.toUpperCase()
        return info
    }

    const colors = {
        debug: 'dim',
        info: 'cyan',
        warn: 'yellow',
        error: 'red',
    }

    return _createLogger({
        level: LOG_LEVEL,
        // Need to upper case level before colorization or we destroy ANSI codes
        format: format.combine({ transform: uppercaseTransformer }, format.colorize({ level: true, colors }), {
            transform: formatTransformer,
        }),
        defaultMeta: { service },
        transports: [new transports.Console({})],
    })
}

/**
 * Log the beginning, end, and exception of an operation.
 *
 * @param name The log message to output.
 * @param logger The logger instance.
 * @param f The operation to perform.
 */
export async function log<T>(name: string, logger: Logger, f: () => Promise<T> | T): Promise<T> {
    const timer = logger.startTimer()
    logger.debug(name)

    try {
        const value = await f()
        timer.done({ message: `${name} finished`, level: 'debug' })
        return value
    } catch (error) {
        timer.done({ message: `${name} failed`, level: 'error', error })
        throw error
    }
}
