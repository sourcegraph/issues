import { SettingsCascadeOrError } from '../settings/settings'
import { isErrorLike } from '../util/errors'

/**
 * Determines if only default extensions are added
 */
export function onlyDefaultExtensionsAdded(settings: SettingsCascadeOrError): boolean {
    if (!isErrorLike(settings.subjects) && settings.subjects) {
        const userSettings = settings.subjects.find(subject => subject.subject.__typename === 'User')
        const defaultSettings = settings.subjects.find(subject => subject.subject.__typename === 'DefaultSettings')

        if (userSettings && defaultSettings) {
            const userExtensions = !isErrorLike(userSettings.settings) && userSettings.settings?.extensions
            const defaultExtensions = !isErrorLike(defaultSettings.settings) && defaultSettings.settings?.extensions

            if (userExtensions && defaultExtensions) {
                for (const key of Object.keys(userExtensions)) {
                    if (!(key in defaultExtensions)) {
                        return false
                    }
                }
            }
        }
    }

    return true
}
