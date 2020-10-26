/**
 * The user preference for the theme.
 * These values are stored in local storage.
 */
export enum ThemePreference {
    Light = 'light',
    Dark = 'dark',
    System = 'system',
}

/**
 * Props that can be extended by any component's Props which needs to manipulate the theme preferences.
 */
export interface ThemePreferenceProps {
    themePreference: ThemePreference
    onThemePreferenceChange: (theme: ThemePreference) => void
}
