import { truncate } from 'lodash'
import * as React from 'react'
import { queryIndexOfScope } from './helpers'

interface Props {
    name?: string
    value: string
    query: string
    count?: number
    limitHit?: boolean
    showMore?: boolean
    onFilterChosen: (value: string) => void
}

export class FilterChip extends React.PureComponent<Props> {
    public render(): JSX.Element | null {
        const truncatedValue = truncate(this.props.value, { length: 50 })
        return (
            <button
                type="button"
                className={
                    `btn btn-sm text-nowrap filter-chip ${this.props.count ? 'filter-chip-repo' : ''}` +
                    (this.isScopeSelected(this.props.query, this.props.value) ? ' filter-chip--selected' : '')
                }
                data-testid="filter-chip"
                value={this.props.value}
                title={this.renderTooltip(this.props.value !== truncatedValue)}
                onMouseDown={this.onMouseDown}
                onClick={this.onClick}
            >
                <div>
                    {this.props.name || truncatedValue}
                    {!!this.props.count && (
                        <span
                            className={`filter-chip__count ${
                                this.isScopeSelected(this.props.query, this.props.value)
                                    ? ' filter-chip__count--selected'
                                    : ''
                            }`}
                        >
                            {this.props.count}
                            {this.props.limitHit ? '+' : ''}
                        </span>
                    )}
                </div>
            </button>
        )
    }

    private renderTooltip(valueIsTruncated: boolean): string | undefined {
        if (this.isScopeSelected(this.props.query, this.props.value)) {
            return 'Already added to query'
        }
        // Show filter value in tooltip if chip shows truncated value or scope name
        if (this.props.name || valueIsTruncated) {
            return this.props.value
        }
        return undefined
    }

    private isScopeSelected(query: string, scope: string): boolean {
        return queryIndexOfScope(query, scope) !== -1
    }

    private onMouseDown: React.MouseEventHandler<HTMLButtonElement> = event => {
        // prevent clicking on chips from taking focus away from the search input.
        event.preventDefault()
    }

    private onClick: React.MouseEventHandler<HTMLButtonElement> = event => {
        event.preventDefault()
        this.props.onFilterChosen(event.currentTarget.value)
    }
}
