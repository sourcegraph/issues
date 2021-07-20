import {
    SettingsOrgSubject,
    SettingsSiteSubject,
    SettingsSubject,
    SettingsUserSubject,
} from '@sourcegraph/shared/src/settings/settings'

/**
 * Currently we support only two types of subject that can have insights dashboard.
 */
export type SupportedInsightSubject = SettingsUserSubject | SettingsOrgSubject | SettingsSiteSubject

/**
 * Supported insight subject types.
 *
 * Values of this enum are synced with settings subject __typename values.
 */
export enum SupportedInsightSubjectType {
    User = 'User',
    Organization = 'Org',
    Global = 'Site',
}

export const SUBJECT_SHARING_LEVELS: Record<string, number> = {
    [SupportedInsightSubjectType.User]: 1,
    [SupportedInsightSubjectType.Organization]: 2,
    [SupportedInsightSubjectType.Global]: 3,
}

export const SUPPORTED_TYPES_OF_SUBJECT = new Set<string>(Object.values(SupportedInsightSubjectType))

export const isSubjectInsightSupported = (subject: SettingsSubject): subject is SupportedInsightSubject =>
    SUPPORTED_TYPES_OF_SUBJECT.has(subject.__typename)

export const isGlobalSubject = (subject: SupportedInsightSubject): subject is SettingsSiteSubject =>
    subject.__typename === SupportedInsightSubjectType.Global

export const isOrganizationSubject = (subject: SupportedInsightSubject): subject is SettingsOrgSubject =>
    subject.__typename === SupportedInsightSubjectType.Organization

export const isUserSubject = (subject: SupportedInsightSubject): subject is SettingsUserSubject =>
    subject.__typename === SupportedInsightSubjectType.User
