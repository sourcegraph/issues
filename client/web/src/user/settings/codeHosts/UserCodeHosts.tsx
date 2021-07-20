import React, { useCallback } from 'react'

import { ErrorLike } from '@sourcegraph/shared/src/util/errors'
import { Container } from '@sourcegraph/wildcard'

import { codeHostExternalServices } from '../../../components/externalServices/externalServices'
import { ExternalServiceKind, ListExternalServiceFields, UserAreaUserFields } from '../../../graphql-operations'
import { SourcegraphContext } from '../../../jscontext'
import { useCodeHostScopeContext } from '../../../site/CodeHostScopeAlerts/CodeHostScopeProvider'
import { eventLogger } from '../../../tracking/eventLogger'
import { githubRepoScopeRequired, gitlabAPIScopeRequired } from '../cloud-ga'

import { CodeHostItem } from './CodeHostItem'

export interface UserCodeHosts {
    user: UserAreaUserFields
    externalServices: ListExternalServiceFields[]
    onDidError: (error: ErrorLike) => void
    onDidRemove: () => void
    onNavigation?: (called: boolean) => void
    context: Pick<SourcegraphContext, 'authProviders'>
}

type ServicesByKind = Partial<Record<ExternalServiceKind, ListExternalServiceFields>>
type AuthProvider = SourcegraphContext['authProviders'][0]
type AuthProvidersByKind = Partial<Record<ExternalServiceKind, AuthProvider>>

const cloudSupportedServices = {
    github: codeHostExternalServices.github,
    gitlabcom: codeHostExternalServices.gitlabcom,
}

export const UserCodeHosts: React.FunctionComponent<UserCodeHosts> = ({
    user,
    externalServices,
    context,
    onDidError,
    onDidRemove,
    onNavigation,
}) => {
    const { scopes, setScope } = useCodeHostScopeContext()

    const services: ServicesByKind = externalServices.reduce<ServicesByKind>((accumulator, service) => {
        // backend constraint - non-admin users have only one external service per ExternalServiceKind
        accumulator[service.kind] = service
        return accumulator
    }, {})

    // auth providers by service type
    const authProvidersByKind = context.authProviders.reduce((accumulator: AuthProvidersByKind, provider) => {
        if (provider.authenticationURL) {
            accumulator[provider.serviceType.toLocaleUpperCase() as ExternalServiceKind] = provider
        }
        return accumulator
    }, {})

    const isTokenUpdateRequired: Partial<Record<ExternalServiceKind, boolean | undefined>> = {
        [ExternalServiceKind.GITHUB]: githubRepoScopeRequired(user.tags, scopes.github),
        [ExternalServiceKind.GITLAB]: gitlabAPIScopeRequired(user.tags, scopes.gitlab),
    }

    const navigateToAuthProvider = useCallback(
        (kind: ExternalServiceKind): void => {
            const authProvider = authProvidersByKind[kind]

            if (authProvider) {
                onNavigation?.(true)
                eventLogger.log('UserAttemptConnectCodeHost', { kind })
                window.location.assign(
                    `${authProvider.authenticationURL as string}&redirect=${
                        window.location.href
                    }&op=createCodeHostConnection`
                )
            }
        },
        [authProvidersByKind, onNavigation]
    )

    const removeService = (kind: ExternalServiceKind) => (): void => {
        if (
            (kind === ExternalServiceKind.GITLAB || kind === ExternalServiceKind.GITHUB) &&
            isTokenUpdateRequired[kind]
        ) {
            setScope(kind, null)
        }

        onDidRemove()
    }

    return (
        <Container>
            <ul className="list-group">
                {Object.entries(cloudSupportedServices).map(([id, { kind, defaultDisplayName, icon }]) =>
                    authProvidersByKind[kind] ? (
                        <li key={id} className="list-group-item user-code-hosts-page__code-host-item">
                            <CodeHostItem
                                service={services[kind]}
                                kind={kind}
                                name={defaultDisplayName}
                                isTokenUpdateRequired={isTokenUpdateRequired[kind]}
                                navigateToAuthProvider={navigateToAuthProvider}
                                icon={icon}
                                onDidRemove={removeService(kind)}
                                onDidError={onDidError}
                            />
                        </li>
                    ) : null
                )}
            </ul>
        </Container>
    )
}
