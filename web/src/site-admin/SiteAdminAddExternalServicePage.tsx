import * as H from 'history'
import React, { useEffect, useCallback, useState } from 'react'
import { Observable, concat } from 'rxjs'
import { catchError, map, switchMap } from 'rxjs/operators'
import { Markdown } from '../../../shared/src/components/Markdown'
import { gql } from '../../../shared/src/graphql/graphql'
import * as GQL from '../../../shared/src/graphql/schema'
import { createAggregateError, ErrorLike, asError, isErrorLike } from '../../../shared/src/util/errors'
import { renderMarkdown } from '../../../shared/src/util/markdown'
import { mutateGraphQL } from '../backend/graphql'
import { PageTitle } from '../components/PageTitle'
import { refreshSiteFlags } from '../site/backend'
import { ThemeProps } from '../../../shared/src/theme'
import { ExternalServiceCard } from '../components/ExternalServiceCard'
import { SiteAdminExternalServiceForm } from './SiteAdminExternalServiceForm'
import { AddExternalServiceOptions } from './externalServices'
import { useEventObservable } from '../../../shared/src/util/useObservable'
import { TelemetryProps, TelemetryService } from '../../../shared/src/telemetry/telemetryService'
import { addExternalServiceResult } from '../graphql-operations'

interface Props extends ThemeProps, TelemetryProps {
    history: H.History
    externalService: AddExternalServiceOptions
}

const LOADING = 'loading' as const

/**
 * Page for adding a single external service
 */
export const SiteAdminAddExternalServicePage: React.FunctionComponent<Props> = props => {
    const [config, setConfig] = useState(props.externalService.defaultConfig)
    const [displayName, setDisplayName] = useState(props.externalService.defaultDisplayName)

    useEffect(() => {
        props.telemetryService.logViewEvent('AddExternalService')
    }, [props.telemetryService])

    const [nextSubmit, createdServiceOrError] = useEventObservable(
        useCallback(
            (
                submits: Observable<GQL.AddExternalServiceInput>
            ): Observable<typeof LOADING | ErrorLike | GQL.ExternalService> =>
                submits.pipe(
                    switchMap(input =>
                        concat(
                            [LOADING],
                            addExternalService(input, props.telemetryService).pipe(
                                catchError(error => [asError(error)])
                            )
                        )
                    )
                ),
            [props.telemetryService]
        )
    )

    useEffect(() => {
        if (createdServiceOrError && createdServiceOrError !== LOADING && !isErrorLike(createdServiceOrError)) {
            // Refresh site flags so that global site alerts
            // reflect the latest configuration.
            // eslint-disable-next-line rxjs/no-nested-subscribe, rxjs/no-ignored-subscription
            refreshSiteFlags().subscribe({ error: error => console.error(error) })
            props.history.push('/site-admin/repositories?repositoriesUpdated')
        }
    }, [createdServiceOrError, props.history])

    const getExternalServiceInput = useCallback(
        (): GQL.AddExternalServiceInput => ({
            displayName,
            config,
            kind: props.externalService.kind,
        }),
        [displayName, config, props.externalService.kind]
    )

    const onChange = useCallback(
        (input: GQL.AddExternalServiceInput): void => {
            setDisplayName(input.displayName)
            setConfig(input.config)
        },
        [setDisplayName, setConfig]
    )

    const onSubmit = useCallback(
        (event?: React.FormEvent<HTMLFormElement>): void => {
            if (event) {
                event.preventDefault()
            }
            nextSubmit(getExternalServiceInput())
        },
        [nextSubmit, getExternalServiceInput]
    )

    return (
        <div className="add-external-service-page mt-3">
            <PageTitle title="Add repositories" />
            <h2>Add repositories</h2>
            {createdServiceOrError &&
            createdServiceOrError !== LOADING &&
            !isErrorLike(createdServiceOrError) &&
            createdServiceOrError.warning ? (
                <div>
                    <div className="mb-3">
                        <ExternalServiceCard
                            {...props.externalService}
                            title={createdServiceOrError.displayName}
                            shortDescription="Update this external service configuration to manage repository mirroring."
                            to={`/site-admin/external-services/${createdServiceOrError.id}`}
                        />
                    </div>
                    <div className="alert alert-warning">
                        <h4>Warning</h4>
                        <Markdown
                            dangerousInnerHTML={renderMarkdown(createdServiceOrError.warning)}
                            history={props.history}
                        />
                    </div>
                </div>
            ) : (
                <div>
                    <div className="mb-3">
                        <ExternalServiceCard {...props.externalService} />
                    </div>
                    <h3>Instructions:</h3>
                    <div className="mb-4">{props.externalService.instructions}</div>
                    <SiteAdminExternalServiceForm
                        {...props}
                        error={isErrorLike(createdServiceOrError) ? createdServiceOrError : undefined}
                        input={getExternalServiceInput()}
                        editorActions={props.externalService.editorActions}
                        jsonSchema={props.externalService.jsonSchema}
                        mode="create"
                        onSubmit={onSubmit}
                        onChange={onChange}
                        loading={createdServiceOrError === LOADING}
                    />
                </div>
            )}
        </div>
    )
}

export function addExternalService(
    input: GQL.AddExternalServiceInput,
    eventLogger: TelemetryService
): Observable<GQL.ExternalService> {
    return mutateGraphQL<addExternalServiceResult>(
        gql`
            mutation addExternalService($input: AddExternalServiceInput!) {
                addExternalService(input: $input) {
                    id
                    kind
                    displayName
                    warning
                }
            }
        `,
        { input }
    ).pipe(
        map(({ data, errors }) => {
            if (!data || !data.addExternalService || (errors && errors.length > 0)) {
                eventLogger.log('AddExternalServiceFailed')
                throw createAggregateError(errors)
            }
            eventLogger.log('AddExternalServiceSucceeded')
            return data.addExternalService
        })
    )
}
