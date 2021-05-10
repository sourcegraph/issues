import * as H from 'history'
import React from 'react'

import { Markdown } from '@sourcegraph/shared/src/components/Markdown'
import { ResultContainer } from '@sourcegraph/shared/src/components/ResultContainer'
import * as GQL from '@sourcegraph/shared/src/graphql/schema'
import { renderMarkdown } from '@sourcegraph/shared/src/util/markdown'

import { SearchResultMatch } from './SearchResultMatch'

interface Props {
    result: Omit<GQL.IGenericSearchResultInterface, '__typename'>
    history: H.History
    icon: React.ComponentType<{ className?: string }>
}

export class SearchResult extends React.Component<Props> {
    private renderTitle = (): JSX.Element => (
        <div className="search-result__title">
            <Markdown
                className="test-search-result-label"
                dangerousInnerHTML={
                    this.props.result.label.html
                        ? this.props.result.label.html
                        : renderMarkdown(this.props.result.label.text)
                }
            />
            {this.props.result.detail && (
                <>
                    <span className="search-result__spacer" />
                    <Markdown
                        dangerousInnerHTML={
                            this.props.result.detail.html
                                ? this.props.result.detail.html
                                : renderMarkdown(this.props.result.detail.text)
                        }
                    />
                </>
            )}
        </div>
    )

    private renderBody = (): JSX.Element => (
        <>
            {this.props.result.matches.map(match => (
                <SearchResultMatch
                    key={match.url}
                    item={match}
                    highlightRanges={match.highlights}
                    history={this.props.history}
                />
            ))}
        </>
    )

    public render(): JSX.Element {
        return (
            <ResultContainer
                icon={this.props.icon}
                collapsible={this.props.result && this.props.result.matches.length > 0}
                defaultExpanded={true}
                title={this.renderTitle()}
                expandedChildren={this.renderBody()}
            />
        )
    }
}
