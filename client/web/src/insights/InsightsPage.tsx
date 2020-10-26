import React, { useMemo } from 'react'
import { useObservable } from '../../../shared/src/util/useObservable'
import { getViewsForContainer } from '../../../shared/src/api/client/services/viewService'
import { ContributableViewContainer } from '../../../shared/src/api/protocol'
import { ExtensionsControllerProps } from '../../../shared/src/extensions/controller'
import { ViewGrid, ViewGridProps } from '../repo/tree/ViewGrid'
import { InsightsIcon } from './icon'
import PlusIcon from 'mdi-react/PlusIcon'
import { Link } from '../../../shared/src/components/Link'
import GearIcon from 'mdi-react/GearIcon'
import { LoadingSpinner } from '@sourcegraph/react-loading-spinner'
import { PageHeader } from '../components/PageHeader'
import { BreadcrumbsProps, BreadcrumbSetters, Breadcrumbs } from '../components/Breadcrumbs'

interface InsightsPageProps
    extends ExtensionsControllerProps,
        Omit<ViewGridProps, 'views'>,
        BreadcrumbsProps,
        BreadcrumbSetters {}
export const InsightsPage: React.FunctionComponent<InsightsPageProps> = props => {
    props.useBreadcrumb(
        useMemo(
            () => ({
                key: 'Insights',
                element: <>Insights</>,
            }),
            []
        )
    )
    const views = useObservable(
        useMemo(
            () =>
                getViewsForContainer(
                    ContributableViewContainer.InsightsPage,
                    {},
                    props.extensionsController.services.view
                ),
            [props.extensionsController.services.view]
        )
    )
    return (
        <div className="w-100">
            <Breadcrumbs breadcrumbs={props.breadcrumbs} location={props.location} />
            <div className="container mt-3 web-content">
                <PageHeader
                    icon={InsightsIcon}
                    title={
                        <>
                            Insights{' '}
                            <sup>
                                <span className="badge badge-info text-uppercase">Prototype</span>
                            </sup>
                        </>
                    }
                    actions={
                        <>
                            <Link to="/user/settings" className="btn btn-secondary mr-1">
                                <GearIcon className="icon-inline" /> Configure insights
                            </Link>
                            <Link to="/extensions?query=category:Insights" className="btn btn-secondary">
                                <PlusIcon className="icon-inline" /> Add more insights
                            </Link>
                        </>
                    }
                />
                {views === undefined ? (
                    <div className="d-flex w-100">
                        <LoadingSpinner className="my-4" />
                    </div>
                ) : (
                    <ViewGrid {...props} views={views} />
                )}
            </div>
        </div>
    )
}
