import React, { useCallback, useState, useMemo } from 'react'
import * as H from 'history'
import { LineChartContent, BarChartContent, ChartContent, PieChartContent } from 'sourcegraph'
import {
    LineChart,
    ResponsiveContainer,
    CartesianGrid,
    XAxis,
    YAxis,
    Tooltip,
    Legend,
    Line,
    Dot,
    LabelFormatter,
    Bar,
    Cell,
    Rectangle,
    PieChart,
    Pie,
    BarChart,
    Sector,
    ContentRenderer,
    PieLabelRenderProps,
} from 'recharts'
import { createLinkClickHandler } from '../../../shared/src/components/linkClickHandler'
import niceTicks from 'nice-ticks'

/** Wraps the children in a link if an href is passed. */
const MaybeLink: React.FunctionComponent<React.AnchorHTMLAttributes<unknown>> = ({ children, ...props }) =>
    props.href ? <a {...props}>{children}</a> : (children as React.ReactElement)

const animationDuration = 600

const strokeWidth = 2
const activeDotStrokeWidth = 3
const dotRadius = 4
const activeDotRadius = 5

const dateTickFormat = new Intl.DateTimeFormat(undefined, { month: 'numeric', day: 'numeric' })
const dateTickFormatter = dateTickFormat.format.bind(dateTickFormat)

const tooltipLabelFormat = new Intl.DateTimeFormat(undefined, { month: 'short', day: 'numeric', year: 'numeric' })
const dateTooltipLabelFormatter = tooltipLabelFormat.format.bind(tooltipLabelFormat)

const toLocaleString = (value: {}): string => value.toLocaleString()

/**
 * Displays a line or bar chart view content.
 */
export const CartesianChartViewContent: React.FunctionComponent<{
    content: LineChartContent<any, string> | BarChartContent<any, string>
    animate?: boolean
    history: H.History
}> = ({ content, animate, history }) => {
    const linkHandler = useMemo(() => createLinkClickHandler(history), [history])

    // Unwrap union type
    const series: typeof content.series[number][] = content.series

    // rechart's default ticks are not as nice
    const yAxisTicks = useMemo(() => {
        const allValues = series.flatMap(series => content.data.map(datum => datum[series.dataKey]))
        return niceTicks(Math.min(0, ...allValues), Math.max(...allValues), 6)
    }, [content.data, series])

    const ChartComponent = content.chart === 'line' ? LineChart : BarChart
    return (
        <ResponsiveContainer width="100%">
            <ChartComponent
                className="cartesian-chart-view-content"
                data={content.data}
                // Chart is weirdly shifted to the right by default
                margin={{ top: 5, bottom: 5, left: -20, right: 20 }}
            >
                <CartesianGrid vertical={false} stroke="var(--border-color)" />
                <XAxis
                    dataKey={content.xAxis.dataKey}
                    scale={content.xAxis.scale}
                    type={content.xAxis.type}
                    domain={['dataMin', 'dataMax']}
                    tickLine={true}
                    stroke="" // prefer CSS to style
                    tickFormatter={content.xAxis.scale === 'time' ? dateTickFormatter : toLocaleString}
                />
                <YAxis
                    domain={[yAxisTicks[0], yAxisTicks[yAxisTicks.length - 1]]}
                    ticks={yAxisTicks}
                    type="number"
                    axisLine={false}
                    tickLine={false}
                    tick={{ stroke: null }} // prefer CSS to style
                    tickFormatter={toLocaleString}
                />
                <Tooltip
                    isAnimationActive={false}
                    labelFormatter={
                        content.xAxis.scale === 'time' ? (dateTooltipLabelFormatter as LabelFormatter) : undefined
                    }
                    allowEscapeViewBox={{ x: false, y: false }} // TODO assign z-indexes to grid items to allow overflow
                />
                {series.length > 1 && (
                    // Ensure the legend is centered despite the margin passed to ChartComponent
                    <Legend wrapperStyle={{ width: null, left: 0, right: 0 }} />
                )}
                {content.chart === 'line'
                    ? content.series.map(series => (
                          <Line
                              isAnimationActive={animate}
                              key={series.dataKey as string}
                              name={series.name}
                              dataKey={series.dataKey as string}
                              stroke={series.stroke}
                              strokeWidth={strokeWidth}
                              label={false}
                              animationDuration={animationDuration}
                              animationEasing="ease-in"
                              // recharts types are wrong
                              // eslint-disable-next-line react/jsx-no-bind
                              dot={(props: any) => (
                                  <MaybeLink
                                      href={series.linkURLs?.[props.index]}
                                      key={props.key}
                                      onClick={linkHandler}
                                      className="d-block p-1"
                                  >
                                      <Dot {...props} r={dotRadius} strokeWidth={strokeWidth} fill="" />
                                  </MaybeLink>
                              )}
                              // recharts types are wrong
                              // eslint-disable-next-line react/jsx-no-bind
                              activeDot={({ key, ...props }: any) => (
                                  // Add links to dots
                                  <MaybeLink
                                      href={series.linkURLs?.[props.index]}
                                      key={key}
                                      onClick={linkHandler}
                                      className="d-block p-1"
                                  >
                                      <Dot
                                          {...props}
                                          r={activeDotRadius}
                                          strokeWidth={activeDotStrokeWidth}
                                          fill="" // prefer CSS for styling
                                          style={{ stroke: series.stroke }}
                                      />
                                  </MaybeLink>
                              )}
                          />
                      ))
                    : content.series.map(series => (
                          <Bar
                              key={series.dataKey as string}
                              name={series.name}
                              dataKey={series.dataKey as string}
                              isAnimationActive={animate}
                              fill={series.fill}
                              label={false}
                              stackId={series.stackId}
                              // eslint-disable-next-line react/jsx-no-bind
                              shape={({ key, ...props }: any) => (
                                  // Add links to bars
                                  <MaybeLink href={series.linkURLs?.[props.index]} key={key} onClick={linkHandler}>
                                      <Rectangle {...props} />
                                  </MaybeLink>
                              )}
                          />
                      ))}
            </ChartComponent>
        </ResponsiveContainer>
    )
}

const percentageLabel: ContentRenderer<PieLabelRenderProps> = props =>
    props.name + (props.percent ? ': ' + (props.percent * 100).toFixed(0) + '%' : '')

export const PieChartViewContent: React.FunctionComponent<{
    content: PieChartContent<any>
    animate?: boolean
    history: H.History
}> = ({ content, history, animate }) => {
    const linkHandler = useMemo(() => createLinkClickHandler(history), [history])

    // Track hovered element to wrap it with a link
    const [activeIndex, setActiveIndex] = useState<number>()
    const onMouseEnter = useCallback((data, index) => setActiveIndex(index), [])

    return (
        <ResponsiveContainer className="pie-chart-view-content">
            <PieChart>
                {content.pies.map(({ fillKey, dataKey, nameKey, linkURLKey, data }) => (
                    <Pie
                        isAnimationActive={animate}
                        key={dataKey as string}
                        animationDuration={animationDuration}
                        data={data}
                        dataKey={dataKey as string}
                        nameKey={nameKey as string}
                        outerRadius="70%"
                        // Ensure first sector is at 12 o'clock
                        startAngle={90}
                        endAngle={360 + 90}
                        label={percentageLabel}
                        labelLine={{ stroke: '' }}
                        activeIndex={activeIndex}
                        // eslint-disable-next-line react/jsx-no-bind
                        activeShape={({ key, ...props }) => (
                            <MaybeLink key={key} href={linkURLKey && props.payload[linkURLKey]} onClick={linkHandler}>
                                <Sector {...props} />
                            </MaybeLink>
                        )}
                        onMouseEnter={onMouseEnter}
                    >
                        {fillKey && data.map(entry => <Cell key={entry[nameKey]} fill={entry[fillKey]} stroke="" />)}
                    </Pie>
                ))}
            </PieChart>
        </ResponsiveContainer>
    )
}

/**
 * Displays chart view content.
 */
export const ChartViewContent: React.FunctionComponent<{
    content: ChartContent
    animate?: boolean
    history: H.History
}> = ({ content, ...props }) => (
    <>
        {content.chart === 'line' || content.chart === 'bar' ? (
            <CartesianChartViewContent {...props} content={content} />
        ) : content.chart === 'pie' ? (
            <PieChartViewContent {...props} content={content} />
        ) : null}
    </>
)
