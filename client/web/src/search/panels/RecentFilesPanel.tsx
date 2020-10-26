import classNames from 'classnames'
import FileCodeIcon from 'mdi-react/FileCodeIcon'
import React, { useCallback, useEffect, useMemo, useState } from 'react'
import { AuthenticatedUser } from '../../auth'
import { EventLogResult } from '../backend'
import { Link } from '../../../../shared/src/components/Link'
import { LoadingPanelView } from './LoadingPanelView'
import { Observable } from 'rxjs'
import { PanelContainer } from './PanelContainer'
import { ShowMoreButton } from './ShowMoreButton'
import { TelemetryProps } from '../../../../shared/src/telemetry/telemetryService'
import { useObservable } from '../../../../shared/src/util/useObservable'

interface Props extends TelemetryProps {
    className?: string
    authenticatedUser: AuthenticatedUser | null
    fetchRecentFileViews: (userId: string, first: number) => Observable<EventLogResult | null>
}

export const RecentFilesPanel: React.FunctionComponent<Props> = ({
    className,
    authenticatedUser,
    fetchRecentFileViews,
    telemetryService,
}) => {
    const pageSize = 20

    const [itemsToLoad, setItemsToLoad] = useState(pageSize)
    const recentFiles = useObservable(
        useMemo(() => fetchRecentFileViews(authenticatedUser?.id || '', itemsToLoad), [
            authenticatedUser?.id,
            fetchRecentFileViews,
            itemsToLoad,
        ])
    )

    const [processedResults, setProcessedResults] = useState<RecentFile[] | null>(null)

    // Only update processed results when results are valid to prevent
    // flashing loading screen when "Show more" button is clicked
    useEffect(() => {
        if (recentFiles) {
            setProcessedResults(processRecentFiles(recentFiles))
        }
    }, [recentFiles])

    useEffect(() => {
        // Only log the first load (when items to load is equal to the page size)
        if (processedResults && itemsToLoad === pageSize) {
            telemetryService.log('RecentFilesPanelLoaded', { empty: processedResults.length === 0 })
        }
    }, [processedResults, telemetryService, itemsToLoad])

    const logFileClicked = useCallback(() => telemetryService.log('RecentFilesPanelFileClicked'), [telemetryService])

    const loadingDisplay = <LoadingPanelView text="Loading recent files" />

    const emptyDisplay = (
        <div className="panel-container__empty-container align-items-center text-muted">
            <FileCodeIcon className="mb-2" size="2rem" />
            <small className="mb-2">This panel will display your most recently viewed files.</small>
        </div>
    )

    function loadMoreItems(): void {
        setItemsToLoad(current => current + pageSize)
        telemetryService.log('RecentFilesPanelShowMoreClicked')
    }

    const contentDisplay = (
        <div>
            <div className="mb-1 mt-2">
                <small>File</small>
            </div>
            <dl className="list-group-flush">
                {processedResults?.map((recentFile, index) => (
                    <dd key={index} className="text-monospace test-recent-files-item">
                        <small>
                            <Link to={recentFile.url} onClick={logFileClicked}>
                                {recentFile.repoName} › {recentFile.filePath}
                            </Link>
                        </small>
                    </dd>
                ))}
            </dl>
            {recentFiles?.pageInfo.hasNextPage && (
                <div className="test-recent-files-show-more-container">
                    <ShowMoreButton onClick={loadMoreItems} className="test-recent-files-panel-show-more" />
                </div>
            )}
        </div>
    )

    return (
        <PanelContainer
            className={classNames(className, 'recent-files-panel')}
            title="Recent files"
            state={processedResults ? (processedResults.length > 0 ? 'populated' : 'empty') : 'loading'}
            loadingContent={loadingDisplay}
            populatedContent={contentDisplay}
            emptyContent={emptyDisplay}
        />
    )
}

interface RecentFile {
    repoName: string
    filePath: string
    timestamp: string
    url: string
}

function processRecentFiles(eventLogResult?: EventLogResult): RecentFile[] | null {
    if (!eventLogResult) {
        return null
    }

    const recentFiles: RecentFile[] = []

    for (const node of eventLogResult.nodes) {
        if (node.argument) {
            const parsedArguments = JSON.parse(node.argument)
            let repoName = parsedArguments?.repoName as string
            let filePath = parsedArguments?.filePath as string

            if (!repoName || !filePath) {
                ;({ repoName, filePath } = extractFileInfoFromUrl(node.url))
            }

            if (
                filePath &&
                repoName &&
                !recentFiles.some(file => file.repoName === repoName && file.filePath === filePath) // Don't show the same file twice
            ) {
                const parsedUrl = new URL(node.url)
                recentFiles.push({
                    url: parsedUrl.pathname + parsedUrl.search, // Strip domain from URL so clicking on it doesn't reload page
                    repoName,
                    filePath,
                    timestamp: node.timestamp,
                })
            }
        }
    }

    return recentFiles
}

function extractFileInfoFromUrl(url: string): { repoName: string; filePath: string } {
    const parsedUrl = new URL(url)

    // Remove first character as it's a '/'
    const [repoName, filePath] = parsedUrl.pathname.slice(1).split('/-/blob/')
    if (!repoName || !filePath) {
        return { repoName: '', filePath: '' }
    }

    return { repoName, filePath }
}
