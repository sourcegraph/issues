import classNames from 'classnames'
import CheckCircleIcon from 'mdi-react/CheckCircleIcon'
import ErrorIcon from 'mdi-react/ErrorIcon'
import React from 'react'

import { pluralize } from '@sourcegraph/shared/src/util/strings'

import { Collapsible } from './Collapsible'
import { Timestamp } from './time/Timestamp'

interface ExecutionLogEntryProps extends React.PropsWithChildren<{}> {
    logEntry: {
        key: string
        command: string[]
        startTime: string
        exitCode: number
        out: string
        durationMilliseconds: number
    }
    now?: () => Date
}

export const ExecutionLogEntry: React.FunctionComponent<ExecutionLogEntryProps> = ({ logEntry, children, now }) => (
    <div className="card mb-3">
        <div className="card-body">
            <LogOutput text={logEntry.command.join(' ')} className="mb-3" />

            <div>
                {logEntry.exitCode === 0 ? (
                    <CheckCircleIcon className="text-success" />
                ) : (
                    <ErrorIcon className="text-danger" />
                )}

                <span className="ml-2">
                    <span className="text-muted">Started</span>{' '}
                    <Timestamp date={logEntry.startTime} now={now} noAbout={true} />
                    <span className="text-muted">, ran for</span> {formatMilliseconds(logEntry.durationMilliseconds)}
                </span>
            </div>

            {children}
        </div>

        <div className="p-2">
            {logEntry.out ? (
                <Collapsible title="Log output" titleAtStart={true} buttonClassName="p-2">
                    <LogOutput text={logEntry.out} />
                </Collapsible>
            ) : (
                <div className="p-2">
                    <span className="text-muted">No log output available.</span>
                </div>
            )}
        </div>
    </div>
)

interface LogOutputProps {
    text: string
    className?: string
}

const LogOutput: React.FunctionComponent<LogOutputProps> = React.memo(({ text, className }) => (
    <pre className={classNames('bg-code rounded p-3 mb-0', className)}>
        {
            // Use index as key because log lines may not be unique. This is OK
            // here because this list will not be updated during this component's
            // lifetime (note: it's also memoized).
            /* eslint-disable react/no-array-index-key */
            text.split('\n').map((line, index) => (
                <code key={index} className={classNames('d-block', line.startsWith('stderr:') ? 'text-danger' : '')}>
                    {line.replace(/^std(out|err): /, '')}
                </code>
            ))
        }
    </pre>
))

const timeOrders: [number, string][] = [
    [1000 * 60 * 60 * 24, 'day'],
    [1000 * 60 * 60, 'hour'],
    [1000 * 60, 'minute'],
    [1000, 'second'],
    [1, 'millisecond'],
]

/**
 * This is essentially to date-fns/formatDistance with support for milliseconds.
 * The output of this function has the following properties:
 *
 * - Consists of one unit (e.g. `x days`) or two units (e.g. `x days and y hours`).
 * - If there are more than one unit, they are adjacent (e.g. never `x days and y minutes`).
 * - If there is a greater unit, the value will not exceed the next threshold (e.g. `2 minutes and 5 seconds`, never `125 seconds`).
 *
 * @param milliseconds The number of milliseconds elapsed.
 */
const formatMilliseconds = (milliseconds: number): string => {
    const parts: string[] = []

    // Construct a list of parts like `1 day` or `7 hours` in descending
    // order. If the value is zero, an empty string is added to the list.`
    timeOrders.reduce((msRemaining, [denominator, suffix]) => {
        // Determine how many units can fit into the current value
        const part = Math.floor(msRemaining / denominator)
        // Format this part (pluralize if value is more than one)
        parts.push(part > 0 ? `${part} ${pluralize(suffix, part)}` : '')
        // Remove this order's contribution to the current value
        return msRemaining - part * denominator
    }, milliseconds)

    return (
        parts
            // Trim leading zero-valued parts
            .slice(parts.findIndex(part => part !== ''))
            // Keep only two consecutive non-zero parts
            .slice(0, 2)
            // Re-filter zero-valued parts
            .filter(part => part !== '')
            // If there are two parts, join them
            .join(' and ')
    )
}
