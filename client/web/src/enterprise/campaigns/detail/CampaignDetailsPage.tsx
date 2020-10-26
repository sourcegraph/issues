import { LoadingSpinner } from '@sourcegraph/react-loading-spinner'
import AlertCircleIcon from 'mdi-react/AlertCircleIcon'
import React, { useEffect, useMemo } from 'react'
import { HeroPage } from '../../../components/HeroPage'
import { PageTitle } from '../../../components/PageTitle'
import { isEqual } from 'lodash'
import {
    fetchCampaignByNamespace as _fetchCampaignByNamespace,
    queryChangesets as _queryChangesets,
    queryExternalChangesetWithFileDiffs as _queryExternalChangesetWithFileDiffs,
    queryChangesetCountsOverTime as _queryChangesetCountsOverTime,
    deleteCampaign as _deleteCampaign,
} from './backend'
import { useObservable } from '../../../../../shared/src/util/useObservable'
import * as H from 'history'
import { delay, distinctUntilChanged, repeatWhen } from 'rxjs/operators'
import { ThemeProps } from '../../../../../shared/src/theme'
import { ExtensionsControllerProps } from '../../../../../shared/src/extensions/controller'
import { PlatformContextProps } from '../../../../../shared/src/platform/context'
import { TelemetryProps } from '../../../../../shared/src/telemetry/telemetryService'
import { CampaignFields, Scalars } from '../../../graphql-operations'
import { CampaignDescription } from './CampaignDescription'
import { CampaignStatsCard } from './CampaignStatsCard'
import { CampaignHeader } from './CampaignHeader'
import { CampaignTabs } from './CampaignTabs'
import { CampaignDetailsActionSection } from './CampaignDetailsActionSection'
import { BreadcrumbSetters } from '../../../components/Breadcrumbs'
import { CampaignInfoByline } from './CampaignInfoByline'
import { UnpublishedNotice } from './UnpublishedNotice'

export interface CampaignDetailsPageProps
    extends ThemeProps,
        ExtensionsControllerProps,
        PlatformContextProps,
        BreadcrumbSetters,
        TelemetryProps {
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
    queryChangesetCountsOverTime?: typeof _queryChangesetCountsOverTime
    /** For testing only. */
    deleteCampaign?: typeof _deleteCampaign
}

/**
 * The area for a single campaign.
 */
export const CampaignDetailsPage: React.FunctionComponent<CampaignDetailsPageProps> = ({
    namespaceID,
    campaignName,
    history,
    location,
    isLightTheme,
    extensionsController,
    platformContext,
    telemetryService,
    useBreadcrumb,
    fetchCampaignByNamespace = _fetchCampaignByNamespace,
    queryChangesets,
    queryExternalChangesetWithFileDiffs,
    queryChangesetCountsOverTime,
    deleteCampaign,
}) => {
    useEffect(() => {
        telemetryService.logViewEvent('CampaignDetailsPagePage')
    }, [telemetryService])

    const campaign: CampaignFields | null | undefined = useObservable(
        useMemo(
            () =>
                fetchCampaignByNamespace(namespaceID, campaignName).pipe(
                    repeatWhen(notifier => notifier.pipe(delay(5000))),
                    distinctUntilChanged((a, b) => isEqual(a, b))
                ),
            [namespaceID, campaignName, fetchCampaignByNamespace]
        )
    )

    useBreadcrumb(
        useMemo(
            () =>
                campaign
                    ? {
                          element: <>{campaign.name}</>,
                          key: 'CampaignDetailsPage',
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
    // Campaign was not found
    if (campaign === null) {
        return <HeroPage icon={AlertCircleIcon} title="Campaign not found" />
    }

    return (
        <>
            <PageTitle title={campaign.name} />
            <CampaignHeader
                name={campaign.name}
                namespace={campaign.namespace}
                actionSection={
                    <CampaignDetailsActionSection
                        campaignID={campaign.id}
                        campaignClosed={!!campaign.closedAt}
                        deleteCampaign={deleteCampaign}
                        campaignNamespaceURL={campaign.namespace.url}
                        history={history}
                    />
                }
                className="test-campaign-details-page"
            />
            <CampaignInfoByline
                createdAt={campaign.createdAt}
                initialApplier={campaign.initialApplier}
                lastAppliedAt={campaign.lastAppliedAt}
                lastApplier={campaign.lastApplier}
                className="mb-3"
            />
            <UnpublishedNotice
                unpublished={campaign.changesets.stats.unpublished}
                total={campaign.changesets.stats.total}
                className="mb-3"
            />
            <CampaignStatsCard closedAt={campaign.closedAt} stats={campaign.changesets.stats} className="mb-3" />
            <CampaignDescription history={history} description={campaign.description} />
            <CampaignTabs
                campaign={campaign}
                extensionsController={extensionsController}
                history={history}
                isLightTheme={isLightTheme}
                location={location}
                platformContext={platformContext}
                telemetryService={telemetryService}
                queryChangesets={queryChangesets}
                queryChangesetCountsOverTime={queryChangesetCountsOverTime}
                queryExternalChangesetWithFileDiffs={queryExternalChangesetWithFileDiffs}
            />
        </>
    )
}
