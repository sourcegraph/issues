import React, { useEffect, useMemo, useState } from 'react'
import { ThemeProps } from '../../../shared/src/theme'

/**
 * Wrapper for the browser extension that listens to changes of the OS theme.
 */
export function ThemeWrapper({
    children,
}: {
    children: JSX.Element | null | ((props: ThemeProps) => JSX.Element | null)
}): JSX.Element | null {
    const darkThemeMediaList = useMemo(() => window.matchMedia('(prefers-color-scheme: dark)'), [])
    const [isLightTheme, setIsLightTheme] = useState(!darkThemeMediaList.matches)

    useEffect(() => {
        const listener = (event: MediaQueryListEvent): void => setIsLightTheme(!event.matches)
        darkThemeMediaList.addEventListener('change', listener)
        return () => darkThemeMediaList.removeEventListener('change', listener)
    }, [darkThemeMediaList])

    useEffect(() => {
        document.body.classList.toggle('theme-light', isLightTheme)
        document.body.classList.toggle('theme-dark', !isLightTheme)
    }, [isLightTheme])

    if (typeof children === 'function') {
        const Children = children
        return <Children isLightTheme={isLightTheme} />
    }
    return children
}
