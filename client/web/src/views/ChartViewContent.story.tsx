import { storiesOf } from '@storybook/react'
import React from 'react'
import webStyles from '../SourcegraphWebApp.scss'
import { ChartViewContent } from './ChartViewContent'
import { createMemoryHistory } from 'history'
import isChromatic from 'chromatic/isChromatic'

const history = createMemoryHistory()

const commonProps = {
    history,
    animate: !isChromatic(),
    location: history.location,
}

const { add } = storiesOf('web/ChartViewContent', module).addDecorator(story => (
    <>
        <style>{webStyles}</style>
        {/* Chart will always fill the container, so we need to give the container an explicit size. */}
        <div className="theme-light" style={{ width: '32rem', height: '16rem' }}>
            {story()}
        </div>
    </>
))

add('Line chart', () => (
    <ChartViewContent
        {...commonProps}
        content={{
            chart: 'line',
            data: [
                { x: 1588965700286 - 4 * 24 * 60 * 60 * 1000, a: 110, b: 150 },
                { x: 1588965700286 - 3 * 24 * 60 * 60 * 1000, a: 145, b: 260 },
                { x: 1588965700286 - 2 * 24 * 60 * 60 * 1000, a: 94, b: 200 },
                { x: 1588965700286 - 1 * 24 * 60 * 60 * 1000, a: 134, b: 190 },
                { x: 1588965700286, a: 123, b: 170 },
            ],
            series: [
                {
                    dataKey: 'a',
                    name: 'A metric',
                    stroke: 'var(--warning)',
                    linkURLs: [
                        '#A:1st_data_point',
                        '#A:2nd_data_point',
                        '#A:3rd_data_point',
                        '#A:4th_data_point',
                        '#A:5th_data_point',
                    ],
                },
                {
                    dataKey: 'b',
                    name: 'B metric',
                    stroke: 'var(--warning)',
                    linkURLs: [
                        '#B:1st_data_point',
                        '#B:2nd_data_point',
                        '#B:3rd_data_point',
                        '#B:4th_data_point',
                        '#B:5th_data_point',
                    ],
                },
            ],
            xAxis: {
                dataKey: 'x',
                scale: 'time',
                type: 'number',
            },
        }}
    />
))

add('Bar chart', () => (
    <ChartViewContent
        {...commonProps}
        content={{
            chart: 'bar',
            data: [
                { name: 'A', value: 183 },
                { name: 'B', value: 145 },
                { name: 'C', value: 94 },
                { name: 'D', value: 134 },
                { name: 'E', value: 123 },
            ],
            series: [
                {
                    dataKey: 'value',
                    name: 'A metric',
                    fill: 'var(--oc-teal-7)',
                    linkURLs: [
                        '#1st_data_point',
                        '#2nd_data_point',
                        '#3rd_data_point',
                        '#4th_data_point',
                        '#5th_data_point',
                    ],
                },
            ],
            xAxis: {
                dataKey: 'name',
                type: 'category',
            },
        }}
    />
))

add('Pie chart', () => (
    <ChartViewContent
        {...commonProps}
        content={{
            chart: 'pie',
            pies: [
                {
                    dataKey: 'value',
                    nameKey: 'name',
                    fillKey: 'fill',
                    linkURLKey: 'linkURL',
                    data: [
                        {
                            name: 'Covered',
                            value: 0.3,
                            fill: 'var(--success)',
                            linkURL: '#Covered',
                        },
                        {
                            name: 'Not covered',
                            value: 0.7,
                            fill: 'var(--danger)',
                            linkURL: '#Not_covered',
                        },
                    ],
                },
            ],
        }}
    />
))
