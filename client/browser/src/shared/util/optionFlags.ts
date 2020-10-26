import { observeStorageKey } from '../../browser-extension/web-extension-api/storage'
import { map, distinctUntilChanged } from 'rxjs/operators'
import { Observable } from 'rxjs'
import { isDefaultSourcegraphUrl } from './context'

const OPTION_FLAGS_SYNC_STORAGE_KEY = 'featureFlags'

export type OptionFlagKey =
    | 'sendTelemetry'
    | 'allowErrorReporting'
    | 'experimentalLinkPreviews'
    | 'experimentalTextFieldCompletion'

export interface OptionFlagDefinition {
    label: string
    key: OptionFlagKey
    hidden?: boolean
}

export interface OptionFlagWithValue<T = boolean> extends OptionFlagDefinition {
    value: T
}

export type OptionFlagValues<T = boolean> = Record<OptionFlagKey, T>

export const optionFlagDefinitions: OptionFlagDefinition[] = [
    {
        key: 'sendTelemetry',
        label: 'Send telemetry',
    },
    {
        key: 'allowErrorReporting',
        label: 'Allow error reporting',
    },
    {
        key: 'experimentalLinkPreviews',
        label: 'Experimental link previews',
    },
    {
        key: 'experimentalTextFieldCompletion',
        label: 'Experimental text field completion',
    },
]

const optionFlagDefaults: OptionFlagValues = {
    sendTelemetry: false,
    allowErrorReporting: false,
    experimentalLinkPreviews: false,
    experimentalTextFieldCompletion: false,
}

export function assignOptionFlagValues(values: OptionFlagValues): OptionFlagWithValue[] {
    return optionFlagDefinitions.map(flag => ({ ...flag, value: values[flag.key] }))
}

/**
 * Apply default values to option flags, taking a partial option flag values
 * object and returning a complete option flag values object.
 */
export function applyOptionFlagDefaults(values: Partial<OptionFlagValues> | undefined): OptionFlagValues {
    return { ...optionFlagDefaults, ...values }
}
/**
 * Observe the option flags object, with default values already applied.
 */
export const observeOptionFlags = (): Observable<OptionFlagValues> =>
    observeStorageKey('sync', OPTION_FLAGS_SYNC_STORAGE_KEY).pipe(map(applyOptionFlagDefaults))

/**
 * Observe an option flag value, with default value already applied.
 */
export function observeOptionFlag(key: OptionFlagKey): Observable<boolean> {
    return observeOptionFlags().pipe(
        map(value => value[key]),
        distinctUntilChanged()
    )
}

/**
 * Determine if the sendTelemetry option flag should be overriden.
 *
 * This function encapsulates the logic of when telemetry should be overriden.
 */
export function shouldOverrideSendTelemetry(isFirefox: boolean, isExtension: boolean, sourcegraphUrl: string): boolean {
    const isFirefoxExtension = isFirefox && isExtension
    if (!isFirefoxExtension) {
        return true
    }

    if (!isDefaultSourcegraphUrl(sourcegraphUrl)) {
        return true
    }

    return false
}
