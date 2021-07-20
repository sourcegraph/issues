import { SettingsSiteSubject } from '@sourcegraph/shared/src/settings/settings'

/**
 * Returns tooltip text for global subject visibility option.
 */
export function getGlobalSubjectTooltipText(globalSubject: SettingsSiteSubject | undefined): string | undefined {
    if (!globalSubject) {
        return
    }

    const globalSubjectAdminCheckMessage = globalSubject.viewerCanAdminister
        ? undefined
        : "You don't have a permission to change global scope. Reach out your site admin"

    return globalSubject.allowSiteSettingsEdits
        ? globalSubjectAdminCheckMessage
        : 'The global subject cannot be edited since your Sourcegraph instance is using a separate settings file'
}
