import * as H from 'history'
import React from 'react'
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
} from 'recharts'
import { ChangesetCounts } from '../../../../../shared/src/graphql/schema'
import { CampaignFields } from '../../../graphql-operations'

interface Props extends Pick<CampaignFields, 'changesetCountsOverTime'> {
    history: H.History
    width?: string | number
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
    ChangesetCounts,
    'openPending' | 'openChangesRequested' | 'openApproved' | 'closed' | 'merged'
>

const states: Record<keyof DisplayableChangesetCounts, StateDefinition> = {
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
export const CampaignBurndownChart: React.FunctionComponent<Props> = ({ changesetCountsOverTime, width = '100%' }) => {
    if (changesetCountsOverTime.length <= 1) {
        return (
            <p>
                <em>Burndown chart will be shown when there is more than 1 day of data.</em>
            </p>
        )
    }
    const hasEntries = changesetCountsOverTime.some(counts => counts.total > 0)
    if (!hasEntries) {
        return (
            <p>
                <em>Burndown chart will be shown when data is available.</em>
            </p>
        )
    }
    return (
        <ResponsiveContainer width={width} height={300}>
            <ComposedChart
                data={changesetCountsOverTime.map(snapshot => ({ ...snapshot, date: Date.parse(snapshot.date) }))}
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
                    wrapperStyle={{ border: '1px solid var(--color-border)' }}
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
            </ComposedChart>
        </ResponsiveContainer>
    )
}
