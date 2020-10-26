import * as GQL from '../../../shared/src/graphql/schema'
import { SettingsCascadeOrError } from '../../../shared/src/settings/settings'
import { isErrorLike } from '../../../shared/src/util/errors'
import { LayoutProps } from '../Layout'
import { parseSearchURLPatternType } from '../search'
import { SettingsExperimentalFeatures } from '../schema/settings.schema'
import { AuthenticatedUser } from '../auth'
import { SearchPatternType } from '../../../shared/src/graphql-operations'

/** A fallback settings subject that can be constructed synchronously at initialization time. */
export const SITE_SUBJECT_NO_ADMIN: Pick<GQL.ISettingsSubject, 'id' | 'viewerCanAdminister'> = {
    id: window.context.siteGQLID,
    viewerCanAdminister: false,
}

export function viewerSubjectFromSettings(
    cascade: SettingsCascadeOrError,
    authenticatedUser: AuthenticatedUser | null
): LayoutProps['viewerSubject'] {
    if (authenticatedUser) {
        return authenticatedUser
    }
    if (cascade && !isErrorLike(cascade) && cascade.subjects && cascade.subjects.length > 0) {
        return cascade.subjects[0].subject
    }
    return SITE_SUBJECT_NO_ADMIN
}

export function defaultPatternTypeFromSettings(settingsCascade: SettingsCascadeOrError): SearchPatternType | undefined {
    // When the web app mounts, if the current page does not have a patternType URL
    // parameter, set the search pattern type to the defaultPatternType from settings
    // (if it is set), otherwise default to literal.
    //
    // For search result URLs that have no patternType= query parameter,
    // the `SearchResults` component will append &patternType=regexp
    // to the URL to ensure legacy search links continue to work.
    if (!parseSearchURLPatternType(window.location.search)) {
        const defaultPatternType =
            settingsCascade.final &&
            !isErrorLike(settingsCascade.final) &&
            settingsCascade.final['search.defaultPatternType']
        return defaultPatternType || 'literal'
    }
    return
}

export function experimentalFeaturesFromSettings(
    settingsCascade: SettingsCascadeOrError
): {
    splitSearchModes: boolean
    copyQueryButton: boolean
    searchStreaming: boolean
    showRepogroupHomepage: boolean
    showOnboardingTour: boolean
    showEnterpriseHomePanels: boolean
    showMultilineSearchConsole: boolean
    showQueryBuilder: boolean
} {
    const experimentalFeatures: SettingsExperimentalFeatures =
        (settingsCascade.final && !isErrorLike(settingsCascade.final) && settingsCascade.final.experimentalFeatures) ||
        {}

    const {
        splitSearchModes = false,
        copyQueryButton = false,
        searchStreaming = false,
        showRepogroupHomepage = false,
        showOnboardingTour = false,
        showEnterpriseHomePanels = true, // Default to true if not set
        showMultilineSearchConsole = false,
        showQueryBuilder = false,
    } = experimentalFeatures

    return {
        splitSearchModes,
        copyQueryButton,
        searchStreaming,
        showRepogroupHomepage,
        showOnboardingTour,
        showEnterpriseHomePanels,
        showMultilineSearchConsole,
        showQueryBuilder,
    }
}
