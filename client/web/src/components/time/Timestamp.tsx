import { parseISO } from 'date-fns'
import formatDistance from 'date-fns/formatDistance'
import formatDistanceStrict from 'date-fns/formatDistanceStrict'
import * as React from 'react'

interface Props {
    /** The date (if string, in ISO 8601 format). */
    date: string | Date | number

    /** Omit the "about". */
    noAbout?: boolean

    /** Function that returns the current time (for stability in visual tests). */
    now?: () => Date

    /** Whether to use exact timestamps (i.e. omit "less than", "about", etc.) */
    strict?: boolean
}

/**
 * Displays a date's relative time ("... ago") and shows the full date on hover. Re-renders
 * periodically to ensure the relative time string is up-to-date.
 */
export class Timestamp extends React.PureComponent<Props> {
    private static RERENDER_INTERVAL_MSEC = 7000

    private intervalHandle: number | null = null

    public componentDidMount(): void {
        this.intervalHandle = window.setInterval(() => this.forceUpdate(), Timestamp.RERENDER_INTERVAL_MSEC)
    }

    public componentWillUnmount(): void {
        if (this.intervalHandle !== null) {
            window.clearInterval(this.intervalHandle)
        }
    }

    public render(): JSX.Element {
        let label = formatDistance(
            typeof this.props.date === 'string' ? parseISO(this.props.date) : this.props.date,
            this.props.now ? this.props.now() : Date.now(),
            { addSuffix: true, includeSeconds: true }
        )
        if (this.props.strict) {
            label = formatDistanceStrict(
                typeof this.props.date === 'string' ? parseISO(this.props.date) : this.props.date,
                this.props.now ? this.props.now() : Date.now(),
                { addSuffix: true }
            )
        }
        if (this.props.noAbout) {
            label = label.replace('about ', '')
        }
        return (
            <span className="timestamp" data-tooltip={this.props.date}>
                {label}
            </span>
        )
    }
}
