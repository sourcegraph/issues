import { USER_DISPLAY_NAME_MAX_LENGTH, USERNAME_MAX_LENGTH, VALID_USERNAME_REGEXP } from '../user'

/**
 * Regular expression to identify valid organization names.
 */
export const VALID_ORG_NAME_REGEXP = VALID_USERNAME_REGEXP

/** Maximum allowed length for an organization name. */
export const ORG_NAME_MAX_LENGTH = USERNAME_MAX_LENGTH

/** Maximum allowed length for an organization display name. */
export const ORG_DISPLAY_NAME_MAX_LENGTH = USER_DISPLAY_NAME_MAX_LENGTH

/** Returns the URL path to an organization's public profile */
export function orgURL(name: string): string {
    return `/organizations/${name}`
}
