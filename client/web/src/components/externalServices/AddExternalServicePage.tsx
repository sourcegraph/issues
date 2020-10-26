import * as H from 'history'
import React, { useEffect, useCallback, useState } from 'react'
import { Markdown } from '../../../../shared/src/components/Markdown'
import { asError, isErrorLike } from '../../../../shared/src/util/errors'
import { renderMarkdown } from '../../../../shared/src/util/markdown'
import { PageTitle } from '../PageTitle'
import { refreshSiteFlags } from '../../site/backend'
import { ThemeProps } from '../../../../shared/src/theme'
import { ExternalServiceCard } from './ExternalServiceCard'
import { AddExternalServiceOptions } from './externalServices'
import { TelemetryProps } from '../../../../shared/src/telemetry/telemetryService'
import { ExternalServiceForm } from './ExternalServiceForm'
import { addExternalService } from './backend'
import { ExternalServiceFields, Scalars, AddExternalServiceInput } from '../../graphql-operations'

interface Props extends ThemeProps, TelemetryProps {
    history: H.History
    externalService: AddExternalServiceOptions
    routingPrefix: string
    afterCreateRoute: string
    userID?: Scalars['ID']

    /** For testing only. */
    autoFocusForm?: boolean
}

/**
 * Page for adding a single external service.
 */
export const AddExternalServicePage: React.FunctionComponent<Props> = ({
    afterCreateRoute,
    externalService,
    history,
    isLightTheme,
    routingPrefix,
    telemetryService,
    userID,
    autoFocusForm,
}) => {
    const [config, setConfig] = useState(externalService.defaultConfig)
    const [displayName, setDisplayName] = useState(externalService.defaultDisplayName)

    useEffect(() => {
        telemetryService.logViewEvent('AddExternalService')
    }, [telemetryService])

    const getExternalServiceInput = useCallback(
        (): AddExternalServiceInput => ({
            displayName,
            config,
            kind: externalService.kind,
        }),
        [displayName, config, externalService.kind]
    )

    const onChange = useCallback(
        (input: AddExternalServiceInput): void => {
            setDisplayName(input.displayName)
            setConfig(input.config)
        },
        [setDisplayName, setConfig]
    )

    const [isCreating, setIsCreating] = useState<boolean | Error>(false)
    const [createdExternalService, setCreatedExternalService] = useState<ExternalServiceFields>()
    const onSubmit = useCallback(
        async (event?: React.FormEvent<HTMLFormElement>): Promise<void> => {
            if (event) {
                event.preventDefault()
            }
            setIsCreating(true)
            try {
                const service = await addExternalService(
                    { input: { ...getExternalServiceInput(), namespace: userID ?? null } },
                    telemetryService
                )
                setIsCreating(false)
                setCreatedExternalService(service)
            } catch (error) {
                setIsCreating(asError(error))
            }
        },
        [getExternalServiceInput, telemetryService, userID]
    )

    useEffect(() => {
        if (createdExternalService && !isErrorLike(createdExternalService)) {
            // Refresh site flags so that global site alerts
            // reflect the latest configuration.
            // eslint-disable-next-line rxjs/no-ignored-subscription
            refreshSiteFlags().subscribe({ error: error => console.error(error) })
            history.push(afterCreateRoute)
        }
    }, [afterCreateRoute, createdExternalService, history])

    return (
        <div className="add-external-service-page mt-3">
            <PageTitle title="Add repositories" />
            <h2>Add repositories</h2>
            {createdExternalService?.warning ? (
                <div>
                    <div className="mb-3">
                        <ExternalServiceCard
                            {...externalService}
                            title={createdExternalService.displayName}
                            shortDescription="Update this external service configuration to manage repository mirroring."
                            to={`${routingPrefix}/external-services/${createdExternalService.id}`}
                        />
                    </div>
                    <div className="alert alert-warning">
                        <h4>Warning</h4>
                        <Markdown
                            dangerousInnerHTML={renderMarkdown(createdExternalService.warning)}
                            history={history}
                        />
                    </div>
                </div>
            ) : (
                <div>
                    <div className="mb-3">
                        <ExternalServiceCard {...externalService} />
                    </div>
                    <h3>Instructions:</h3>
                    <div className="mb-4">{externalService.instructions}</div>
                    <ExternalServiceForm
                        history={history}
                        isLightTheme={isLightTheme}
                        telemetryService={telemetryService}
                        error={isErrorLike(isCreating) ? isCreating : undefined}
                        input={getExternalServiceInput()}
                        editorActions={externalService.editorActions}
                        jsonSchema={externalService.jsonSchema}
                        mode="create"
                        onSubmit={onSubmit}
                        onChange={onChange}
                        loading={isCreating === true}
                        autoFocus={autoFocusForm}
                    />
                </div>
            )}
        </div>
    )
}
