import React, { useMemo } from 'react'
import { ExtensionsControllerProps } from '../../../shared/src/extensions/controller'
import { PageTitle } from '../components/PageTitle'
import * as H from 'history'
import { ContributableViewContainer } from '../../../shared/src/api/protocol'
import { LoadingSpinner } from '@sourcegraph/react-loading-spinner'
import { getView } from '../../../shared/src/api/client/services/viewService'
import { useObservable } from '../../../shared/src/util/useObservable'
import { ViewContentProps, ViewContent } from './ViewContent'

interface Props extends ExtensionsControllerProps<'services'>, Omit<ViewContentProps, 'viewContent'> {
    viewID: string
    extraPath: string

    location: H.Location
    history: H.History

    /** For mocking in tests. */
    _getView?: typeof getView
}

/**
 * A page that displays a single view (contributed by an extension) as a standalone page.
 */
export const ViewPage: React.FunctionComponent<Props> = ({
    viewID,
    extraPath,
    location,
    extensionsController,
    _getView = getView,
    ...props
}) => {
    const queryParameters = useMemo<Record<string, string>>(
        () => ({ ...Object.fromEntries(new URLSearchParams(location.search).entries()), extraPath }),
        [extraPath, location.search]
    )

    const contributions = useMemo(() => extensionsController.services.contribution.getContributions(), [
        extensionsController.services.contribution,
    ])
    const view = useObservable(
        useMemo(
            () =>
                _getView(
                    viewID,
                    ContributableViewContainer.GlobalPage,
                    queryParameters,
                    contributions,
                    extensionsController.services.view
                ),
            [_getView, contributions, extensionsController.services.view, queryParameters, viewID]
        )
    )

    if (view === undefined) {
        return <LoadingSpinner className="icon-inline" />
    }

    if (view === null) {
        return (
            <div className="alert alert-danger">
                View not found: <code>{viewID}</code>
            </div>
        )
    }

    return (
        <div>
            <PageTitle title={view.title || 'View'} />
            {view.title && <h1>{view.title}</h1>}
            <ViewContent viewContent={view.content} location={location} {...props} />
        </div>
    )
}
