import { SettingsCascadeProps } from '@sourcegraph/shared/src/settings/settings'

import { isSubjectInsightSupported, SupportedInsightSubject } from '../../core/types/subjects'

/**
 * Returns all subjects that are supportable by insight logic.
 */
export function useInsightSubjects(props: SettingsCascadeProps): SupportedInsightSubject[] {
    const { settingsCascade } = props

    if (!settingsCascade.subjects) {
        return []
    }

    return settingsCascade.subjects
        .map(configureSubject => configureSubject.subject)
        .filter<SupportedInsightSubject>(isSubjectInsightSupported)
}
