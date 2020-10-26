import classNames from 'classnames'
import * as React from 'react'
import { numberWithCommas, pluralize } from '../../../../shared/src/util/strings'

const NUM_SQUARES = 5

interface Props {
    /** Number of additions (added lines). */
    added: number

    /** Number of changes (changed lines). */
    changed: number

    /** Number of deletions (deleted lines). */
    deleted: number

    /* Show +/- numbers, not just the total change count. */
    expandedCounts?: boolean

    separateLines?: boolean

    className?: string
}

/** Displays a diff stat (visual representation of added, changed, and deleted lines in a diff). */
export const DiffStat: React.FunctionComponent<Props> = React.memo(function DiffStat({
    added,
    changed,
    deleted,
    expandedCounts = false,
    separateLines = false,
    className = '',
}) {
    const total = added + changed + deleted
    const numberOfSquares = Math.min(NUM_SQUARES, total)
    let addedSquares = allocateSquares(added, total)
    let changedSquares = allocateSquares(changed, total)
    let deletedSquares = allocateSquares(deleted, total)

    // Make sure we have exactly numSquares squares.
    const totalSquares = addedSquares + changedSquares + deletedSquares
    if (totalSquares < numberOfSquares) {
        const deficit = numberOfSquares - totalSquares
        if (deleted > changed && deleted > added) {
            deletedSquares += deficit
        } else if (changed > added && changed > deleted) {
            changedSquares += deficit
        } else {
            addedSquares += deficit
        }
    } else if (totalSquares > numberOfSquares) {
        const surplus = numberOfSquares - totalSquares
        if (deleted <= changed && deleted <= added) {
            deletedSquares -= surplus
        } else if (changed < added && changed < deleted) {
            changedSquares -= surplus
        } else {
            addedSquares -= surplus
        }
    }

    const squares = new Array<'bg-success' | 'bg-warning' | 'bg-danger' | 'diff-stat__empty'>(addedSquares)
        .fill('bg-success')
        .concat(
            new Array<'bg-warning'>(changedSquares).fill('bg-warning'),
            new Array<'bg-danger'>(deletedSquares).fill('bg-danger'),
            new Array<'diff-stat__empty'>(NUM_SQUARES - numberOfSquares).fill('diff-stat__empty')
        )

    const labels: string[] = []
    if (added > 0) {
        labels.push(`${numberWithCommas(added)} ${pluralize('addition', added)}`)
    }
    if (changed > 0) {
        labels.push(`${numberWithCommas(changed)} ${pluralize('change', changed)}`)
    }
    if (deleted > 0) {
        labels.push(`${numberWithCommas(deleted)} ${pluralize('deletion', deleted)}`)
    }
    return (
        <div
            className={classNames('diff-stat', separateLines && 'flex-column', className)}
            data-tooltip={labels.join(', ')}
        >
            {expandedCounts ? (
                <span className="diff-stat__total font-weight-bold">
                    <span className="text-success mr-1">+{numberWithCommas(added)}</span>
                    {changed > 0 && <span className="text-warning mr-1">&bull;{numberWithCommas(changed)}</span>}
                    <span className={classNames('text-danger', !separateLines && 'mr-1')}>
                        &minus;{numberWithCommas(deleted)}
                    </span>
                </span>
            ) : (
                <small className="diff-stat__total">{numberWithCommas(total + changed)}</small>
            )}
            <div>
                {squares.map((className, index) => (
                    <div key={index} className={`diff-stat__square ${className}`} />
                ))}
            </div>
        </div>
    )
})

function allocateSquares(number: number, total: number): number {
    if (total === 0) {
        return 0
    }
    return Math.max(Math.round(number / total), number > 0 ? 1 : 0)
}
