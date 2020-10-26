import React from 'react'
import { ThemeProps } from '../../../../shared/src/theme'
import classNames from 'classnames'

interface Props extends ThemeProps, Exclude<React.ImgHTMLAttributes<HTMLImageElement>, 'src'> {
    /**
     * The site configuration `branding` property. If not set, the global value from
     * `window.context.branding` is used.
     */
    branding?: typeof window.context.branding

    /**
     * The assets root path. If not set, the global value from `window.context.assetsRoot` is used.
     */
    assetsRoot?: typeof window.context.assetsRoot

    /** Whether to show the full logo (with text) or only the symbol icon. */
    variant: 'logo' | 'symbol'
}

/**
 * The Sourcegraph logo image. If a custom logo specified in the `branding` site configuration
 * property, it is used instead.
 */
export const BrandLogo: React.FunctionComponent<Props> = ({
    isLightTheme,
    branding,
    assetsRoot,
    variant,
    className = '',
    ...props
}) => {
    // Workaround: can't put this in optional parameter value because of https://github.com/babel/babel/issues/11166
    branding = branding ?? window.context?.branding
    assetsRoot = assetsRoot ?? (window.context?.assetsRoot || '')

    const sourcegraphLogoUrl =
        variant === 'symbol'
            ? `${assetsRoot}/img/sourcegraph-mark.svg`
            : `${assetsRoot}/img/sourcegraph${isLightTheme ? '-light' : ''}-head-logo.svg?v2`
    const customBrandingLogoUrl = branding?.[isLightTheme ? 'light' : 'dark']?.[variant]
    return (
        <img
            {...props}
            className={classNames('brand-logo', className, {
                'brand-logo--spin': variant === 'symbol' && !branding?.disableSymbolSpin,
            })}
            src={customBrandingLogoUrl || sourcegraphLogoUrl}
        />
    )
}
