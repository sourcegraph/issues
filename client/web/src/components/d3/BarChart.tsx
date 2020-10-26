import { axisBottom, AxisContainerElement } from 'd3-axis'
import { scaleBand, scaleLinear, scaleOrdinal } from 'd3-scale'
import { select, Selection } from 'd3-selection'
import { stack } from 'd3-shape'
import { isEqual } from 'lodash'
import * as React from 'react'
import { ThemeProps } from '../../../../shared/src/theme'

interface BarChartSeries {
    [key: string]: null
}

interface BarChartDatum<T extends BarChartSeries> {
    xLabel: string
    yValues: { [key in keyof T]: number }
}

interface Props<T extends BarChartSeries> extends ThemeProps {
    /**
     * Bar chart data.
     * One datum for each column, with each datum containing values for each series in the given column.
     */
    data: BarChartDatum<T>[]
    /**
     * Initial width (chart will be automatically resized to fit container).
     */
    width: number
    /**
     * Initial height (chart will be automatically resized to fit container).
     */
    height: number
    /**
     * Display column totals labels.
     */
    showLabels?: boolean
    /**
     * Display legend.
     */
    showLegend?: boolean
    className?: string
}

export class BarChart<T extends BarChartSeries> extends React.Component<Props<T>> {
    private svgRef: SVGSVGElement | null = null

    public componentDidMount(): void {
        this.drawChart()
    }

    public componentDidUpdate(): void {
        this.drawChart()
    }

    public shouldComponentUpdate(nextProps: Props<T>): boolean {
        return !isEqual(this.props, nextProps)
    }

    private drawChart = (): void => {
        if (!this.svgRef) {
            return
        }
        const { width, height } = this.props

        const data = this.props.data.reverse()
        const barColors = this.props.isLightTheme ? ['#a2b0cd', '#cad2e2'] : ['#566e9f', '#a2b0cd']
        const series = Object.keys(data[0].yValues)
        const xLabels = data.map(({ xLabel }) => xLabel)
        const yValues = data.map(({ yValues }) => yValues)
        const yHeights = data.map(({ yValues }) =>
            Object.keys(yValues).reduce((accumulator, key) => accumulator + yValues[key], 0)
        )

        if (data.length === 0) {
            return
        }

        const columns = xLabels.length

        const xScaleBand = scaleBand().domain(xLabels).rangeRound([0, width])
        const yScaleBand = scaleLinear()
            .domain([0, Math.max(...yHeights)])
            .range([height, 0])
        const zScaleOrdinal = scaleOrdinal<string, string>().domain(series).range(barColors)
        const xAxis = axisBottom(xScaleBand)

        const svg = select(this.svgRef)
        svg.selectAll('*').remove()

        const barWidth = width / columns - 2

        const barHolder = svg
            .classed(`d3-bar-chart ${this.props.className || ''}`, true)
            .attr('preserveAspectRatio', 'xMinYMin')
            .append('g')
            .classed('bar-holder', true)

        const stackData = stack()
            .keys(series)
            .value((data, key) => data[key])(yValues, series)

        // Generate bars.
        barHolder
            .append('g')
            .selectAll('g')
            .data(stackData)
            .enter()
            .append('g')
            .attr('fill', data => zScaleOrdinal(data.key))
            .selectAll('rect')
            .data(data => data)
            .enter()
            .append('rect')
            .classed('bar', true)
            .attr('x', (data, index) => xScaleBand(xLabels[index]) || 0 + 1)
            .attr('y', data => yScaleBand(data[1]))
            .attr('width', barWidth)
            .attr('height', data => yScaleBand(data[0]) - yScaleBand(data[1]))
            .attr('data-tooltip', data => `${data[1] - data[0]} users`)

        if (this.props.showLabels) {
            // Generate value labels on top of each column.
            barHolder
                .append('g')
                .selectAll('text')
                .data(data)
                .enter()
                .append('text')
                .attr('text-anchor', 'middle')
                .attr('x', data => xScaleBand(data.xLabel) || 0)
                .attr('dx', barWidth / 2)
                .attr('y', (data, index) => yScaleBand(yHeights[index]))
                .attr('dy', '-0.5em')
                .text((data, index) => yHeights[index])
        }

        // Generate x-axis and labels.
        barHolder
            .append<AxisContainerElement>('g')
            .classed('axis', true)
            .attr('transform', `translate(0, ${height})`)
            .call(xAxis)
            .selectAll('.tick text')
            .call(wrapLabel, barWidth)

        if (this.props.showLegend) {
            // Generate a legend.
            const legend = barHolder
                .append('svg')
                .attr('y', '-5em')
                .append('g')
                .attr('text-anchor', 'end')
                .selectAll('g')
                .data(series.slice().reverse())
                .enter()
                .append('g')
                .attr('transform', (data, index) => `translate(0,${index * 20})`)
            legend
                .append('rect')
                .attr('x', width - 19)
                .attr('width', 19)
                .attr('height', 19)
                .attr('fill', zScaleOrdinal)
            legend
                .append('text')
                .attr('x', width - 24)
                .attr('y', 9.5)
                .attr('dy', '0.32em')
                .text(data => data)
        }
    }

    public render(): JSX.Element | null {
        const { width, height } = this.props
        return <svg viewBox={`0 0 ${width} ${height}`} ref={reference => (this.svgRef = reference)} />
    }
}

// Source: Mike Bostock's "Wrapping Long Labels": https://bl.ocks.org/mbostock/7555321
function wrapLabel(text: Selection<any, any, any, any>, width: number): void {
    text.each(function (): void {
        const text = select(this)
        const words = text.text().split(/\s+/).reverse()
        const lineHeight = 1.1
        const yAttribute = text.attr('y')
        const dyAttribute = parseFloat(text.attr('dy'))
        let lineNumber = 0
        let currentWord
        // currentLine holds the line as it grows, until it overflows.
        let currentLine: string[] = []
        // tspan holds the current <tspan> element as it grows, until it overflows.
        let tspan = text.text(null).append('tspan').attr('x', 0).attr('y', yAttribute).attr('dy', `${dyAttribute}em`)

        while (words.length) {
            currentWord = words.pop() || ''
            currentLine.push(currentWord)
            tspan.text(currentLine.join(' '))
            if ((tspan.node() as SVGTextContentElement).getComputedTextLength() > width) {
                currentLine.pop()
                tspan.text(currentLine.join(' '))
                // Start a new line and generate a new tspan element.
                currentLine = [currentWord]
                tspan = text
                    .append('tspan')
                    .attr('x', 0)
                    .attr('y', yAttribute)
                    .attr('dy', `${++lineNumber * lineHeight + dyAttribute}em`)
                    .text(currentWord)
            }
        }
    })
}
