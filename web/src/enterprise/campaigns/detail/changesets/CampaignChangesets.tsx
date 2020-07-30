import React, { useState, useCallback, useMemo, useEffect } from 'react'
import * as H from 'history'
import * as GQL from '../../../../../../shared/src/graphql/schema'
import { ChangesetNodeProps, ChangesetNode } from './ChangesetNode'
import { ThemeProps } from '../../../../../../shared/src/theme'
import { FilteredConnection, FilteredConnectionQueryArgs } from '../../../../components/FilteredConnection'
import { Subject, merge, of } from 'rxjs'
import { upperFirst, lowerCase } from 'lodash'
import { queryChangesets as _queryChangesets } from '../backend'
import { repeatWhen, delay, withLatestFrom, map, filter, switchMap } from 'rxjs/operators'
import { ExtensionsControllerProps } from '../../../../../../shared/src/extensions/controller'
import { createHoverifier, HoveredToken } from '@sourcegraph/codeintellify'
import {
    RepoSpec,
    RevisionSpec,
    FileSpec,
    ResolvedRevisionSpec,
    UIPositionSpec,
    ModeSpec,
} from '../../../../../../shared/src/util/url'
import { HoverMerged } from '../../../../../../shared/src/api/client/types/hover'
import { ActionItemAction } from '../../../../../../shared/src/actions/ActionItem'
import { getHoverActions } from '../../../../../../shared/src/hover/actions'
import { WebHoverOverlay } from '../../../../components/shared'
import { getModeFromPath } from '../../../../../../shared/src/languages'
import { getHover, getDocumentHighlights } from '../../../../backend/features'
import { PlatformContextProps } from '../../../../../../shared/src/platform/context'
import { TelemetryProps } from '../../../../../../shared/src/telemetry/telemetryService'
import { property, isDefined } from '../../../../../../shared/src/util/types'
import { useObservable } from '../../../../../../shared/src/util/useObservable'
import { CampaignChangesetsResult } from '../../../../graphql-operations'

interface Props extends ThemeProps, PlatformContextProps, TelemetryProps, ExtensionsControllerProps {
    campaign: Pick<GQL.Campaign, 'id' | 'closedAt' | 'viewerCanAdminister'>
    history: H.History
    location: H.Location
    campaignUpdates: Subject<void>
    changesetUpdates: Subject<void>

    /** For testing only. */
    queryChangesets?: typeof _queryChangesets
}

function getLSPTextDocumentPositionParameters(
    hoveredToken: HoveredToken & RepoSpec & RevisionSpec & FileSpec & ResolvedRevisionSpec
): RepoSpec & RevisionSpec & ResolvedRevisionSpec & FileSpec & UIPositionSpec & ModeSpec {
    return {
        repoName: hoveredToken.repoName,
        revision: hoveredToken.revision,
        filePath: hoveredToken.filePath,
        commitID: hoveredToken.commitID,
        position: hoveredToken,
        mode: getModeFromPath(hoveredToken.filePath || ''),
    }
}

/**
 * A list of a campaign's changesets.
 */
export const CampaignChangesets: React.FunctionComponent<Props> = ({
    campaign,
    history,
    location,
    isLightTheme,
    changesetUpdates,
    campaignUpdates,
    extensionsController,
    platformContext,
    telemetryService,
    queryChangesets = _queryChangesets,
}) => {
    const [state, setState] = useState<GQL.ChangesetState | undefined>()
    const [reviewState, setReviewState] = useState<GQL.ChangesetReviewState | undefined>()
    const [checkState, setCheckState] = useState<GQL.ChangesetCheckState | undefined>()

    const queryChangesetsConnection = useCallback(
        (args: FilteredConnectionQueryArgs) =>
            merge(of(undefined), changesetUpdates).pipe(
                switchMap(() =>
                    queryChangesets({
                        ...args,
                        campaign: campaign.id,
                        state: state ?? null,
                        reviewState: reviewState ?? null,
                        checkState: checkState ?? null,
                    }).pipe(repeatWhen(notifier => notifier.pipe(delay(5000))))
                )
            ),
        [campaign.id, state, reviewState, checkState, queryChangesets, changesetUpdates]
    )

    const containerElements = useMemo(() => new Subject<HTMLElement | null>(), [])
    const nextContainerElement = useMemo(() => containerElements.next.bind(containerElements), [containerElements])

    const hoverOverlayElements = useMemo(() => new Subject<HTMLElement | null>(), [])
    const nextOverlayElement = useCallback((element: HTMLElement | null): void => hoverOverlayElements.next(element), [
        hoverOverlayElements,
    ])

    const closeButtonClicks = useMemo(() => new Subject<MouseEvent>(), [])
    const nextCloseButtonClick = useCallback((event: MouseEvent): void => closeButtonClicks.next(event), [
        closeButtonClicks,
    ])

    const componentRerenders = useMemo(() => new Subject<void>(), [])

    const hoverifier = useMemo(
        () =>
            createHoverifier<RepoSpec & RevisionSpec & FileSpec & ResolvedRevisionSpec, HoverMerged, ActionItemAction>({
                closeButtonClicks,
                hoverOverlayElements,
                hoverOverlayRerenders: componentRerenders.pipe(
                    withLatestFrom(hoverOverlayElements, containerElements),
                    map(([, hoverOverlayElement, relativeElement]) => ({
                        hoverOverlayElement,
                        // The root component element is guaranteed to be rendered after a componentDidUpdate
                        // eslint-disable-next-line @typescript-eslint/no-non-null-assertion
                        relativeElement: relativeElement!,
                    })),
                    // Can't reposition HoverOverlay if it wasn't rendered
                    filter(property('hoverOverlayElement', isDefined))
                ),
                getHover: hoveredToken =>
                    getHover(getLSPTextDocumentPositionParameters(hoveredToken), { extensionsController }),
                getDocumentHighlights: hoveredToken =>
                    getDocumentHighlights(getLSPTextDocumentPositionParameters(hoveredToken), { extensionsController }),
                getActions: context => getHoverActions({ extensionsController, platformContext }, context),
                pinningEnabled: true,
            }),
        [
            closeButtonClicks,
            containerElements,
            extensionsController,
            hoverOverlayElements,
            platformContext,
            componentRerenders,
        ]
    )
    useEffect(() => () => hoverifier.unsubscribe(), [hoverifier])

    const hoverState = useObservable(useMemo(() => hoverifier.hoverStateUpdates, [hoverifier]))
    useEffect(() => {
        componentRerenders.next()
    }, [componentRerenders, hoverState])

    const changesetFiltersRow = (
        <div className="form-inline mb-0 mt-2">
            <label htmlFor="changeset-state-filter">State</label>
            <select
                className="form-control mx-2"
                value={state}
                onChange={event => setState((event.target.value || undefined) as GQL.ChangesetState | undefined)}
                id="changeset-state-filter"
            >
                <option value="">All</option>
                {Object.values(GQL.ChangesetState).map(state => (
                    <option value={state} key={state}>
                        {upperFirst(lowerCase(state))}
                    </option>
                ))}
            </select>
            <label htmlFor="changeset-review-state-filter">Review state</label>
            <select
                className="form-control mx-2"
                value={reviewState}
                onChange={event =>
                    setReviewState((event.target.value || undefined) as GQL.ChangesetReviewState | undefined)
                }
                id="changeset-review-state-filter"
            >
                <option value="">All</option>
                {Object.values(GQL.ChangesetReviewState).map(state => (
                    <option value={state} key={state}>
                        {upperFirst(lowerCase(state))}
                    </option>
                ))}
            </select>
            <label htmlFor="changeset-check-state-filter">Check state</label>
            <select
                className="form-control mx-2"
                value={checkState}
                onChange={event =>
                    setCheckState((event.target.value || undefined) as GQL.ChangesetCheckState | undefined)
                }
                id="changeset-check-state-filter"
            >
                <option value="">All</option>
                {Object.values(GQL.ChangesetCheckState).map(state => (
                    <option value={state} key={state}>
                        {upperFirst(lowerCase(state))}
                    </option>
                ))}
            </select>
        </div>
    )

    return (
        <>
            {changesetFiltersRow}
            <div className="list-group position-relative" ref={nextContainerElement}>
                <FilteredConnection<
                    (CampaignChangesetsResult['node'] & { __typename: 'Campaign' })['changesets']['nodes'][0],
                    Omit<ChangesetNodeProps, 'node'>
                >
                    className="mt-2"
                    nodeComponent={ChangesetNode}
                    nodeComponentProps={{
                        isLightTheme,
                        viewerCanAdminister: campaign.viewerCanAdminister,
                        history,
                        location,
                        campaignUpdates,
                        extensionInfo: { extensionsController, hoverifier },
                    }}
                    queryConnection={queryChangesetsConnection}
                    hideSearch={true}
                    defaultFirst={15}
                    noun="changeset"
                    pluralNoun="changesets"
                    history={history}
                    location={location}
                    useURLQuery={false}
                />
                {hoverState?.hoverOverlayProps && (
                    <WebHoverOverlay
                        {...hoverState.hoverOverlayProps}
                        telemetryService={telemetryService}
                        extensionsController={extensionsController}
                        isLightTheme={isLightTheme}
                        location={location}
                        platformContext={platformContext}
                        hoverRef={nextOverlayElement}
                        onCloseButtonClick={nextCloseButtonClick}
                    />
                )}
            </div>
        </>
    )
}
