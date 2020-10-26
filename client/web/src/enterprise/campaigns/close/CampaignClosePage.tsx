import React, { useState, useMemo, useCallback } from 'react'
import * as H from 'history'
import { PageTitle } from '../../../components/PageTitle'
import { CampaignHeader } from '../detail/CampaignHeader'
import { CampaignCloseAlert } from './CampaignCloseAlert'
import { CampaignChangesetsResult, CampaignFields, Scalars } from '../../../graphql-operations'
import {
    queryExternalChangesetWithFileDiffs as _queryExternalChangesetWithFileDiffs,
    queryChangesets as _queryChangesets,
    fetchCampaignByNamespace as _fetchCampaignByNamespace,
} from '../detail/backend'
import { ThemeProps } from '../../../../../shared/src/theme'
import { PlatformContextProps } from '../../../../../shared/src/platform/context'
import { ExtensionsControllerProps } from '../../../../../shared/src/extensions/controller'
import { TelemetryProps } from '../../../../../shared/src/telemetry/telemetryService'
import { closeCampaign as _closeCampaign } from './backend'
import { CampaignCloseChangesetsList } from './CampaignCloseChangesetsList'
import { useObservable } from '../../../../../shared/src/util/useObservable'
import { LoadingSpinner } from '@sourcegraph/react-loading-spinner'
import { HeroPage } from '../../../components/HeroPage'
import AlertCircleIcon from 'mdi-react/AlertCircleIcon'
import { BreadcrumbSetters } from '../../../components/Breadcrumbs'
import { CampaignInfoByline } from '../detail/CampaignInfoByline'
import { ErrorLike, isErrorLike } from '../../../../../shared/src/util/errors'

export interface CampaignClosePageProps
    extends ThemeProps,
        TelemetryProps,
        PlatformContextProps,
        BreadcrumbSetters,
        ExtensionsControllerProps {
    /**
     * The namespace ID.
     */
    namespaceID: Scalars['ID']
    /**
     * The campaign name.
     */
    campaignName: CampaignFields['name']
    history: H.History
    location: H.Location

    /** For testing only. */
    fetchCampaignByNamespace?: typeof _fetchCampaignByNamespace
    /** For testing only. */
    queryChangesets?: typeof _queryChangesets
    /** For testing only. */
    queryExternalChangesetWithFileDiffs?: typeof _queryExternalChangesetWithFileDiffs
    /** For testing only. */
    closeCampaign?: typeof _closeCampaign
}

export const CampaignClosePage: React.FunctionComponent<CampaignClosePageProps> = ({
    namespaceID,
    campaignName,
    history,
    location,
    extensionsController,
    isLightTheme,
    platformContext,
    telemetryService,
    useBreadcrumb,
    fetchCampaignByNamespace = _fetchCampaignByNamespace,
    queryChangesets,
    queryExternalChangesetWithFileDiffs,
    closeCampaign,
}) => {
    const [closeChangesets, setCloseChangesets] = useState<boolean>(false)
    const campaign = useObservable(
        useMemo(() => fetchCampaignByNamespace(namespaceID, campaignName), [
            namespaceID,
            campaignName,
            fetchCampaignByNamespace,
        ])
    )

    const [totalCount, setTotalCount] = useState<number>()

    const onFetchChangesets = useCallback(
        (connection?: (CampaignChangesetsResult['node'] & { __typename: 'Campaign' })['changesets'] | ErrorLike) => {
            if (!connection || isErrorLike(connection)) {
                return
            }
            setTotalCount(connection.totalCount)
        },
        []
    )

    useBreadcrumb(
        useMemo(
            () =>
                campaign
                    ? {
                          link: { to: campaign.url, label: campaign.name },
                          key: 'CampaignClosePage',
                      }
                    : null,
            [campaign]
        )
    )

    // Is loading.
    if (campaign === undefined) {
        return (
            <div className="text-center">
                <LoadingSpinner className="icon-inline mx-auto my-4" />
            </div>
        )
    }

    // Campaign not found.
    if (campaign === null) {
        return <HeroPage icon={AlertCircleIcon} title="Campaign not found" />
    }

    return (
        <>
            <PageTitle title="Preview close" />
            <CampaignHeader name={campaign.name} namespace={campaign.namespace} className="test-campaign-close-page" />
            <CampaignInfoByline
                createdAt={campaign.createdAt}
                initialApplier={campaign.initialApplier}
                lastAppliedAt={campaign.lastAppliedAt}
                lastApplier={campaign.lastApplier}
                className="mb-3"
            />
            {totalCount !== undefined && (
                <CampaignCloseAlert
                    campaignID={campaign.id}
                    campaignURL={campaign.url}
                    closeChangesets={closeChangesets}
                    setCloseChangesets={setCloseChangesets}
                    history={history}
                    closeCampaign={closeCampaign}
                    viewerCanAdminister={campaign.viewerCanAdminister}
                    totalCount={totalCount}
                />
            )}
            <CampaignCloseChangesetsList
                campaignID={campaign.id}
                history={history}
                location={location}
                viewerCanAdminister={campaign.viewerCanAdminister}
                extensionsController={extensionsController}
                isLightTheme={isLightTheme}
                platformContext={platformContext}
                telemetryService={telemetryService}
                queryChangesets={queryChangesets}
                queryExternalChangesetWithFileDiffs={queryExternalChangesetWithFileDiffs}
                willClose={closeChangesets}
                onUpdate={onFetchChangesets}
            />
        </>
    )
}
