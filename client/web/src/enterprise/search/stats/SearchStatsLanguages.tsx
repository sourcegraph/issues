import * as GQL from '../../../../../shared/src/graphql/schema'
import { PieChart, Pie, Tooltip, ResponsiveContainer, PieLabelRenderProps, Cell, TooltipFormatter } from 'recharts'
import React, { useCallback } from 'react'
import { Link } from 'react-router-dom'
import { numberWithCommas, pluralize } from '../../../../../shared/src/util/strings'
import { buildSearchURLQuery } from '../../../../../shared/src/util/url'
import { SearchPatternType } from '../../../graphql-operations'

const OTHER_LANGUAGE = 'Other'
const UNKNOWN_LANGUAGE = 'Unknown'

/**
 * Return a copy of the stats with all languages that make up less than `minFraction` of the total
 * grouped together as "" (which is displayed as "Other").
 */
export const summarizeSearchResultsStatsLanguages = (
    languages: GQL.ISearchResultsStats['languages'],
    minFraction: number
): GQL.ISearchResultsStats['languages'] => {
    const totalLines = languages.reduce((sum, language) => sum + language.totalLines, 0)
    const minLines = minFraction * totalLines
    const languagesAboveMin = languages.filter(language => language.totalLines >= minLines)
    const otherLines = totalLines - languagesAboveMin.reduce((sum, language) => sum + language.totalLines, 0)
    return [
        ...languagesAboveMin,
        { __typename: 'LanguageStatistics', name: OTHER_LANGUAGE, totalBytes: 0, totalLines: otherLines },
    ]
}

/** Nice-looking colors for the pie chart that have good contrast in both light and dark themes. */
const COLORS = ['#278389', '#f16321', '#753fff', '#0091ea', '#00c853', '#ffab00', '#ff3d00', '#ff7700']

const OTHER_COLOR = '#999999'
const UNKNOWN_COLOR = '#777777'

interface Props {
    query: string
    stats: GQL.ISearchResultsStats
}

/**
 * Shows language statistics about the results for a search query.
 */
export const SearchStatsLanguages: React.FunctionComponent<Props> = ({ query, stats }) => {
    const chartData = summarizeSearchResultsStatsLanguages(stats.languages, 0.02).map((language, index) => ({
        ...language,
        name: language.name || UNKNOWN_LANGUAGE,
        color: COLORS[index % COLORS.length],
    }))

    const totalLines = stats.languages.reduce((sum, language) => sum + language.totalLines, 0)

    const urlToSearchWithExtraQuery = useCallback(
        (extraQuery: string) =>
            `/search?${buildSearchURLQuery(`${query} ${extraQuery}`, SearchPatternType.literal, false)}`,
        [query]
    )

    const percent = useCallback(
        (lines: number) =>
            totalLines !== undefined && totalLines !== 0 ? `${Math.round((100 * lines) / totalLines)}%` : '',
        [totalLines]
    )
    const labelRenderer = useCallback(
        (props: PieLabelRenderProps): string =>
            `${props.name || UNKNOWN_LANGUAGE} ${
                props.percent !== undefined ? `(${Math.round(100 * props.percent)}%)` : ''
            }`,
        []
    )
    const tooltipFormatter = useCallback<TooltipFormatter>(
        value => (typeof value === 'number' ? `${numberWithCommas(value)} ${pluralize('line', value)}` : ''),
        []
    )

    return (
        <div className="card mb-3">
            <h4 className="card-header">Languages</h4>
            {stats.languages.length > 0 ? (
                <div className="d-flex">
                    <div className="flex-0 border-right">
                        <table className="search-stats-page__table table mb-0 border-top-0">
                            <thead>
                                <tr className="small">
                                    <th className="border-top-0">
                                        <span className="sr-only">Language</span>
                                    </th>
                                    <th className="border-top-0">Lines</th>
                                    <th className="border-top-0">
                                        <span className="sr-only">Percent</span>
                                    </th>
                                </tr>
                            </thead>
                            <tbody>
                                {stats.languages.map(({ name, totalLines: lines }, index) => (
                                    <tr key={name || index}>
                                        <td>
                                            {name ? (
                                                <Link to={urlToSearchWithExtraQuery(`lang:${name.toLowerCase()}`)}>
                                                    {name}
                                                </Link>
                                            ) : (
                                                UNKNOWN_LANGUAGE
                                            )}
                                        </td>
                                        <td>{numberWithCommas(lines)}</td>
                                        <td>{percent(lines)}</td>
                                    </tr>
                                ))}
                            </tbody>
                        </table>
                    </div>
                    <div className="flex-1 mx-4">
                        <ResponsiveContainer height={600}>
                            <PieChart margin={{ left: 60, right: 60 }}>
                                <Pie
                                    dataKey="totalLines"
                                    isAnimationActive={false}
                                    data={chartData}
                                    label={labelRenderer}
                                >
                                    {chartData.map((entry, index) => (
                                        <Cell
                                            key={entry.name}
                                            fill={
                                                entry.name === UNKNOWN_LANGUAGE
                                                    ? UNKNOWN_COLOR
                                                    : entry.name === OTHER_LANGUAGE
                                                    ? OTHER_COLOR
                                                    : COLORS[index % COLORS.length]
                                            }
                                        />
                                    ))}
                                </Pie>
                                <Tooltip animationDuration={0} formatter={tooltipFormatter} />
                            </PieChart>
                        </ResponsiveContainer>
                    </div>
                </div>
            ) : (
                <div className="card-body text-muted">No language statistics available.</div>
            )}
        </div>
    )
}
