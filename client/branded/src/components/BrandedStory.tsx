import { radios } from '@storybook/addon-knobs'
import React from 'react'
import { MemoryRouter, MemoryRouterProps } from 'react-router'
import { ThemeProps } from '../../../shared/src/theme'
import brandedStyles from '../global-styles/index.scss'
import { Tooltip } from './tooltip/Tooltip'

export interface WebStoryProps extends MemoryRouterProps {
    children: React.FunctionComponent<ThemeProps>
}

/**
 * Wrapper component for webapp Storybook stories that provides light theme and react-router props.
 * Takes a render function as children that gets called with the props.
 */
export const BrandedStory: React.FunctionComponent<
    WebStoryProps & {
        styles?: string
    }
> = ({ children, styles = brandedStyles, ...memoryRouterProps }) => {
    const theme = radios('Theme', { Light: 'light', Dark: 'dark' }, 'light')
    document.body.classList.toggle('theme-light', theme === 'light')
    document.body.classList.toggle('theme-dark', theme === 'dark')
    const Children = children
    return (
        <MemoryRouter {...memoryRouterProps}>
            <Tooltip />
            <Children isLightTheme={theme === 'light'} />
            <style title="Webapp CSS">{styles}</style>
        </MemoryRouter>
    )
}
