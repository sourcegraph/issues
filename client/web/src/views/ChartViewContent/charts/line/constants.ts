/**
 * Default value for line color in case if we didn't get color for line from content config.
 */
// TODO: Confirm usage, we want to deprecate this color
export const DEFAULT_LINE_STROKE = 'var(--color-bg-3)'

/**
 * Visx xy-chart supports data series with missing. To show the
 * points but not the very beginning of the chart we should use
 * this default empty value. See example below points that have
 * EMPTY_DATA_POINT_VALUE value haven't been rendered instead of
 * that we rendered non active background (``` area)
 *
 * ┌──────────────────────────────────┐     ┌──────────────────────────────────┐
 * │``````````````````                │ 10  │          ````````                │ 10
 * │``````````````````                │     │        ▼ ````````                │
 * │``````````````````              ▼ │ 9   │          ````````              ▼ │ 9
 * │``````````````````                │     │  ▼       ````````                │
 * │``````````````````      ▼         │ 8   │          ````````      ▼         │ 8
 * │``````````````````                │     │          ````````                │
 * │``````````````````          ▼     │ 7   │          ````````          ▼     │ 7
 * │`````````````````` ▼              │     │    ▼     ```````` ▼              │
 * │``````````````````                │ 6   │          ````````                │ 6
 * │``````````````````                │     │          ````````                │
 * │``````````````````                │ 5   │          ````````                │ 5
 * └──────────────────────────────────┘     └──────────────────────────────────┘
 *
 */
export const EMPTY_DATA_POINT_VALUE = null
