// We want to polyfill first.
import '../../shared/polyfills'

import React, { useEffect, useState } from 'react'
import { render } from 'react-dom'
import { from, noop, Observable, combineLatest } from 'rxjs'
import { GraphQLResult } from '../../../../shared/src/graphql/graphql'
import { background } from '../web-extension-api/runtime'
import { observeStorageKey, storage } from '../web-extension-api/storage'
import { initSentry } from '../../shared/sentry'
import { fetchSite } from '../../shared/backend/server'
import { featureFlags } from '../../shared/util/featureFlags'
import {
    OptionFlagKey,
    OptionFlagWithValue,
    assignOptionFlagValues,
    observeOptionFlags,
    shouldOverrideSendTelemetry,
    optionFlagDefinitions,
} from '../../shared/util/optionFlags'
import { assertEnvironment } from '../environmentAssertion'
import {
    observeSourcegraphURL,
    isFirefox,
    getExtensionVersion,
    DEFAULT_SOURCEGRAPH_URL,
} from '../../shared/util/context'
import { catchError, map, mapTo } from 'rxjs/operators'
import { isExtension } from '../../shared/context'
import { OptionsPage, URL_AUTH_ERROR, URL_FETCH_ERROR } from '../options-menu/OptionsPage'
import { asError } from '../../../../shared/src/util/errors'
import { useObservable } from '../../../../shared/src/util/useObservable'
import { AnchorLink, setLinkComponent } from '../../../../shared/src/components/Link'
import { KnownCodeHost, knownCodeHosts } from '../knownCodeHosts'
import { Optional } from 'utility-types'
import { ThemeWrapper } from '../ThemeWrapper'

interface TabStatus {
    host: string
    protocol: string
    hasPermissions: boolean
    isPrivateRepository: boolean
}

assertEnvironment('OPTIONS')

initSentry('options')

const IS_EXTENSION = true

<<<<<<< HEAD:browser/src/browser-extension/scripts/optionsPage.main.tsx
type State = Pick<FeatureFlags, 'allowErrorReporting' | 'experimentalLinkPreviews'> & {
    sourcegraphURL: string | null
    isActivated: boolean
}
=======
/**
 * A list of protocols where we should *not* show the permissions notification.
 */
const PERMISSIONS_PROTOCOL_BLOCKLIST = new Set(['chrome:', 'about:'])
>>>>>>> main:client/browser/src/browser-extension/scripts/optionsPage.main.tsx

setLinkComponent(AnchorLink)

const isOptionFlagKey = (key: string): key is OptionFlagKey =>
    !!optionFlagDefinitions.find(definition => definition.key === key)

const fetchCurrentTabStatus = async (): Promise<TabStatus> => {
    const tabs = await browser.tabs.query({ active: true, currentWindow: true })
    if (tabs.length > 1) {
        throw new Error('Querying for the currently active tab returned more than one result')
    }
    const { url, id } = tabs[0]
    if (!url) {
        throw new Error('Currently active tab has no URL')
    }
    if (!id) {
        throw new Error('Currently active tab has no ID')
    }
    const isPrivateRepository = await background.checkPrivateRepository(id)
    const { host, protocol } = new URL(url)
    const hasPermissions = await browser.permissions.contains({
        origins: [`${protocol}//${host}/*`],
    })
    return { isPrivateRepository, host, protocol, hasPermissions }
}

// Make GraphQL requests from background page
function requestGraphQL<T, V = object>(options: {
    request: string
    variables: V
    sourcegraphURL?: string
}): Observable<GraphQLResult<T>> {
    return from(background.requestGraphQL<T, V>(options))
}

const version = getExtensionVersion()
const isFullPage = !new URLSearchParams(window.location.search).get('popup')

const validateSourcegraphUrl = (url: string): Observable<string | undefined> =>
    fetchSite(options => requestGraphQL({ ...options, sourcegraphURL: url })).pipe(
        mapTo(undefined),
        catchError(error => {
            const { message } = asError(error)
            // We lose Error type when communicating from the background page
            // to the options page, so we determine the error type from the message
            if (message.includes('Failed to fetch')) {
                return [URL_FETCH_ERROR]
            }
            if (message.includes('401')) {
                return [URL_AUTH_ERROR]
            }

            return [message]
        })
    )

const observeOptionFlagsWithValues = (): Observable<OptionFlagWithValue[]> => {
    const overrideSendTelemetry: Observable<boolean> = observeSourcegraphURL(IS_EXTENSION).pipe(
        map(sourcegraphUrl => shouldOverrideSendTelemetry(isFirefox(), isExtension, sourcegraphUrl))
    )

    return combineLatest([observeOptionFlags(), overrideSendTelemetry]).pipe(
        map(([flags, override]) => {
            const definitions = assignOptionFlagValues(flags)
            if (override) {
                return definitions.filter(flag => flag.key !== 'sendTelemetry')
            }
            return definitions
        })
    )
}

<<<<<<< HEAD:browser/src/browser-extension/scripts/optionsPage.main.tsx
class Options extends React.Component<{}, State> {
    public state: State = {
        sourcegraphURL: null,
        isActivated: true,
        allowErrorReporting: false,
        experimentalLinkPreviews: false,
    }
=======
const observingIsActivated = observeStorageKey('sync', 'disableExtension').pipe(map(isDisabled => !isDisabled))
const observingSourcegraphUrl = observeSourcegraphURL(true)
const observingOptionFlagsWithValues = observeOptionFlagsWithValues()
>>>>>>> main:client/browser/src/browser-extension/scripts/optionsPage.main.tsx

function handleToggleActivated(isActivated: boolean): void {
    storage.sync.set({ disableExtension: !isActivated }).catch(console.error)
}

<<<<<<< HEAD:browser/src/browser-extension/scripts/optionsPage.main.tsx
    public componentDidMount(): void {
        this.subscriptions.add(
            observeStorageKey('sync', 'featureFlags').subscribe(featureFlags => {
                const { allowErrorReporting, experimentalLinkPreviews } = {
                    ...featureFlagDefaults,
                    ...featureFlags,
                }
                this.setState({
                    allowErrorReporting,
                    experimentalLinkPreviews,
                })
            })
        )

        this.subscriptions.add(
            observeSourcegraphURL(IS_EXTENSION).subscribe(sourcegraphURL => {
                this.setState({ sourcegraphURL })
            })
        )

        this.subscriptions.add(
            observeStorageKey('sync', 'disableExtension').subscribe(disableExtension => {
                this.setState({
                    isActivated: !disableExtension,
                })
            })
        )
=======
function handleChangeOptionFlag(key: string, value: boolean): void {
    if (isOptionFlagKey(key)) {
        featureFlags.set(key, value).catch(noop)
>>>>>>> main:client/browser/src/browser-extension/scripts/optionsPage.main.tsx
    }
}

function handleChangeSourcegraphUrl(url: string): void {
    storage.sync.set({ sourcegraphURL: url }).catch(console.error)
}

function buildRequestPermissionsHandler({ protocol, host }: TabStatus) {
    return function requestPermissionsHandler(event: React.MouseEvent) {
        event.preventDefault()
        browser.permissions.request({ origins: [`${protocol}//${host}/*`] }).catch(error => {
            console.error('Error requesting permissions:', error)
        })
    }
}

<<<<<<< HEAD:browser/src/browser-extension/scripts/optionsPage.main.tsx
        const props: OptionsContainerProps = {
            sourcegraphURL: this.state.sourcegraphURL,
            isActivated: this.state.isActivated,

            ensureValidSite,
            fetchCurrentTabStatus,
            hasPermissions: url =>
                browser.permissions.contains({
                    origins: [`${url}/*`],
                }),
            requestPermissions: url =>
                browser.permissions.request({
                    origins: [`${url}/*`],
                }),

            setSourcegraphURL: (sourcegraphURL: string) => storage.sync.set({ sourcegraphURL }),
            toggleExtensionDisabled: (isActivated: boolean) => storage.sync.set({ disableExtension: !isActivated }),
            toggleFeatureFlag,
            featureFlags: [
                { key: 'allowErrorReporting', value: this.state.allowErrorReporting },
                { key: 'experimentalLinkPreviews', value: this.state.experimentalLinkPreviews },
            ],
=======
const Options: React.FunctionComponent = () => {
    const sourcegraphUrl = useObservable(observingSourcegraphUrl) || ''
    const isActivated = useObservable(observingIsActivated)
    const optionFlagsWithValues = useObservable(observingOptionFlagsWithValues) || []
    const [currentTabStatus, setCurrentTabStatus] = useState<
        { status: TabStatus; handler: React.MouseEventHandler } | undefined
    >()

    useEffect(() => {
        fetchCurrentTabStatus().then(tabStatus => {
            setCurrentTabStatus({ status: tabStatus, handler: buildRequestPermissionsHandler(tabStatus) })
        }, noop)
    }, [])

    const showSourcegraphCloudAlert = currentTabStatus?.status.host.endsWith('sourcegraph.com')

    let permissionAlert: Optional<KnownCodeHost, 'host' | 'icon'> | undefined
    if (
        currentTabStatus &&
        !currentTabStatus?.status.hasPermissions &&
        !showSourcegraphCloudAlert &&
        !PERMISSIONS_PROTOCOL_BLOCKLIST.has(currentTabStatus.status.protocol)
    ) {
        const knownCodeHost = knownCodeHosts.find(({ host }) => host === currentTabStatus.status.host)
        if (knownCodeHost) {
            permissionAlert = knownCodeHost
        } else {
            permissionAlert = { name: currentTabStatus.status.host }
>>>>>>> main:client/browser/src/browser-extension/scripts/optionsPage.main.tsx
        }
    }

    return (
        <ThemeWrapper>
            <OptionsPage
                isFullPage={isFullPage}
                sourcegraphUrl={sourcegraphUrl}
                onChangeSourcegraphUrl={handleChangeSourcegraphUrl}
                version={version}
                validateSourcegraphUrl={validateSourcegraphUrl}
                isActivated={!!isActivated}
                onToggleActivated={handleToggleActivated}
                optionFlags={optionFlagsWithValues}
                onChangeOptionFlag={handleChangeOptionFlag}
                showPrivateRepositoryAlert={
                    currentTabStatus?.status.isPrivateRepository && sourcegraphUrl === DEFAULT_SOURCEGRAPH_URL
                }
                showSourcegraphCloudAlert={showSourcegraphCloudAlert}
                permissionAlert={permissionAlert}
                currentHost={currentTabStatus?.status.host}
                requestPermissionsHandler={currentTabStatus?.handler}
            />
        </ThemeWrapper>
    )
}

const inject = (): void => {
    render(<Options />, document.body)
}

document.addEventListener('DOMContentLoaded', inject)
