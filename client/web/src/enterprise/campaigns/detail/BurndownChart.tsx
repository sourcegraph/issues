import * as H from 'history'
import React, { useCallback, useMemo, useState } from 'react'
import {
    Area,
    ComposedChart,
    LabelFormatter,
    Legend,
    ResponsiveContainer,
    Tooltip,
    XAxis,
    YAxis,
    TooltipPayload,
    ReferenceArea,
    RechartsFunction,
} from 'recharts'
import { ChangesetCountsOverTimeFields, Scalars } from '../../../graphql-operations'
import { LoadingSpinner } from '@sourcegraph/react-loading-spinner'
import { useObservable } from '../../../../../shared/src/util/useObservable'
import { queryChangesetCountsOverTime as _queryChangesetCountsOverTime } from './backend'
import { subDays } from 'date-fns'

interface Props {
    campaignID: Scalars['ID']
    history: H.History
    width?: string | number

    /** For testing only. */
    queryChangesetCountsOverTime?: typeof _queryChangesetCountsOverTime
}

const dateTickFormat = new Intl.DateTimeFormat(undefined, { month: 'long', day: 'numeric' })
const dateTickFormatter = (timestamp: number): string => dateTickFormat.format(timestamp)

// const tooltipLabelFormat = new Intl.DateTimeFormat(undefined, { dateStyle: 'medium' })
const tooltipLabelFormat = new Intl.DateTimeFormat(undefined, { month: 'short', day: 'numeric', year: 'numeric' })
const tooltipLabelFormatter = (date: number): string => tooltipLabelFormat.format(date)

const toLocaleString = (value: number): string => value.toLocaleString()

const tooltipStyle: React.CSSProperties = {
    color: 'var(--body-color)',
    border: 'none',
    background: 'var(--body-bg)',
}

const commonAreaProps = {
    isAnimationActive: false,
    strokeWidth: 0,
    stackId: 'stack',
    type: 'stepBefore',
} as const

interface StateDefinition {
    fill: string
    label: string
    sortOrder: number
}

type DisplayableChangesetCounts = Pick<
    ChangesetCountsOverTimeFields,
    'openPending' | 'openChangesRequested' | 'openApproved' | 'closed' | 'merged' | 'draft'
>

const states: Record<keyof DisplayableChangesetCounts, StateDefinition> = {
    draft: { fill: 'var(--text-muted)', label: 'Draft', sortOrder: 5 },
    openPending: { fill: 'var(--warning)', label: 'Open & awaiting review', sortOrder: 4 },
    openChangesRequested: { fill: 'var(--danger)', label: 'Open & changes requested', sortOrder: 3 },
    openApproved: { fill: 'var(--success)', label: 'Open & approved', sortOrder: 2 },
    closed: { fill: 'var(--secondary)', label: 'Closed', sortOrder: 1 },
    merged: { fill: 'var(--merged)', label: 'Merged', sortOrder: 0 },
}

const tooltipItemSorter = ({ dataKey }: TooltipPayload): number =>
    states[dataKey as keyof DisplayableChangesetCounts].sortOrder

/**
 * A burndown chart showing progress of the campaigns changesets.
 */
export const CampaignBurndownChart: React.FunctionComponent<Props> = ({
    campaignID,
    queryChangesetCountsOverTime = _queryChangesetCountsOverTime,
    width = '100%',
}) => {
    const [left, setLeft] = useState<number>()
    const [right, setRight] = useState<number>()
    const changesetCountsOverTime: ChangesetCountsOverTimeFields[] | undefined = useObservable(
        useMemo(
            () =>
                queryChangesetCountsOverTime({
                    campaign: campaignID,
                    from: left ? new Date(left).toISOString() : null,
                    to: right ? new Date(right).toISOString() : null,
                }),
            [campaignID, queryChangesetCountsOverTime, left, right]
        )
    )

    const [referenceAreaLeft, setReferenceAreaLeft] = useState<number>()
    const [referenceAreaRight, setReferenceAreaRight] = useState<number>()

    const onMouseDown = useCallback<RechartsFunction>(e => {
        if (!e) {
            return
        }
        setReferenceAreaLeft(e.activeLabel)
    }, [])
    const onMouseMove = useCallback<RechartsFunction>(
        e => {
            if (!e) {
                return
            }
            if (referenceAreaLeft !== undefined) {
                setReferenceAreaRight(e.activeLabel)
            }
        },
        [referenceAreaLeft]
    )
    const onMouseUp = useCallback<RechartsFunction>(
        e => {
            if (
                referenceAreaLeft === referenceAreaRight ||
                referenceAreaLeft === undefined ||
                referenceAreaRight === undefined
            ) {
                setReferenceAreaLeft(undefined)
                setReferenceAreaRight(undefined)
                return
            }

            setReferenceAreaLeft(undefined)
            setReferenceAreaRight(undefined)

            let left = referenceAreaLeft
            let right = referenceAreaRight
            // xAxis domain
            if (left > right) {
                ;[left, right] = [right, left]
            }
            setLeft(left)
            setRight(right)

            // yAxis domain
            // const [bottom, top] = getAxisYDomain(referenceAreaLeft, referenceAreaRight, 'cost', 1)
            // const [bottom2, top2] = getAxisYDomain(referenceAreaLeft, referenceAreaRight, 'impression', 50)

            // this.setState(() => ({
            //     left: refAreaLeft,
            //     right: refAreaRight,
            //     bottom,
            //     top,
            //     bottom2,
            //     top2,
            // }))
        },
        [referenceAreaLeft, referenceAreaRight]
    )
    const resetZoom = useCallback<React.MouseEventHandler>(() => {
        setLeft(undefined)
        setRight(undefined)
    }, [])

    // Is loading.
    if (changesetCountsOverTime === undefined) {
        return (
            <div className="text-center">
                <LoadingSpinner className="icon-inline mx-auto my-4" />
            </div>
        )
    }

    if (changesetCountsOverTime.length <= 1) {
        return (
            <div className="col-md-8 offset-md-2 col-sm-12 card mt-5">
                <div className="card-body p-5">
                    <h2 className="text-center mb-4">The burndown chart requires 2 days of data</h2>
                    <p>
                        Come back in a few days and we'll be able to show you data on how your campaign is progressing!
                    </p>
                </div>
            </div>
        )
    }
    const hasEntries = changesetCountsOverTime.some(counts => counts.total > 0)
    if (!hasEntries) {
        return (
            <div className="col-md-8 offset-md-2 col-sm-12 card mt-5">
                <div className="card-body p-5">
                    <h2 className="text-center mb-4">Burndown chart is not available</h2>
                    <p>The burndown chart will be shown when data is available.</p>
                </div>
            </div>
        )
    }
    return (
        <>
            {left && right && (
                <button type="button" onClick={resetZoom}>
                    Reset zoom
                </button>
            )}
            <ResponsiveContainer width={width} height={300} className="test-campaigns-chart">
                <ComposedChart
                    data={changesetCountsOverTime.map(snapshot => ({ ...snapshot, date: Date.parse(snapshot.date) }))}
                    onMouseDown={onMouseDown}
                    onMouseMove={onMouseMove}
                    onMouseUp={onMouseUp}
                >
                    <Legend verticalAlign="bottom" iconType="square" />
                    <XAxis
                        dataKey="date"
                        domain={[
                            changesetCountsOverTime[0].date,
                            changesetCountsOverTime[changesetCountsOverTime.length - 1].date,
                        ]}
                        name="Time"
                        tickFormatter={dateTickFormatter}
                        type="number"
                        stroke="var(--text-muted)"
                        scale="time"
                    />
                    <YAxis
                        tickFormatter={toLocaleString}
                        stroke="var(--text-muted)"
                        type="number"
                        allowDecimals={false}
                        domain={[0, 'dataMax']}
                    />
                    <Tooltip
                        labelFormatter={tooltipLabelFormatter as LabelFormatter}
                        isAnimationActive={false}
                        wrapperStyle={{ border: '1px solid var(--border-color)' }}
                        contentStyle={tooltipStyle}
                        labelStyle={{ fontWeight: 'bold' }}
                        itemStyle={tooltipStyle}
                        itemSorter={tooltipItemSorter}
                    />

                    {Object.entries(states)
                        .sort(([, a], [, b]) => b.sortOrder - a.sortOrder)
                        .map(([dataKey, state]) => (
                            <Area
                                key={state.sortOrder}
                                dataKey={dataKey}
                                name={state.label}
                                fill={state.fill}
                                // The stroke is used to color the legend, which we
                                // want to match the fill color for each area.
                                stroke={state.fill}
                                {...commonAreaProps}
                            />
                        ))}

                    {referenceAreaLeft && referenceAreaRight && (
                        <ReferenceArea x1={referenceAreaLeft} x2={referenceAreaRight} strokeOpacity={0.3} />
                    )}
                </ComposedChart>
            </ResponsiveContainer>
        </>
    )
}
