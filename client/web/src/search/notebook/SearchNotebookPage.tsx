import classNames from 'classnames'
import React, { useEffect } from 'react'

import { TelemetryProps } from '@sourcegraph/shared/src/telemetry/telemetryService'
import { ThemeProps } from '@sourcegraph/shared/src/theme'
import { FeedbackBadge } from '@sourcegraph/web/src/components/FeedbackBadge'
import { Page } from '@sourcegraph/web/src/components/Page'
import { PageTitle } from '@sourcegraph/web/src/components/PageTitle'
import { PageHeader } from '@sourcegraph/wildcard'

import { SearchStreamingProps } from '..'
import { StreamingSearchResultsListProps } from '../results/StreamingSearchResultsList'

import styles from './SearchNotebookPage.module.scss'

interface SearchNotebookPageProps
    extends SearchStreamingProps,
        ThemeProps,
        TelemetryProps,
        Omit<StreamingSearchResultsListProps, 'allExpanded'> {
    globbing: boolean
    isMacPlatform: boolean
}

export const SearchNotebookPage: React.FunctionComponent<SearchNotebookPageProps> = props => {
    useEffect(() => props.telemetryService.logViewEvent('SearchNotebookPage'), [props.telemetryService])

    return (
        <div className={classNames('w-100 p-2', styles.searchNotebookPage)}>
            <PageTitle title="Search Notebook" />
            <Page>
                <PageHeader
                    className="mx-3"
                    annotation={<FeedbackBadge status="prototype" feedback={{ mailto: 'support@sourcegraph.com' }} />}
                    path={[{ text: 'Search Notebook' }]}
                />
                <hr className="mt-2 mb-1 mx-3" />
            </Page>
        </div>
    )
}
