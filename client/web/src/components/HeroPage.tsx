import * as React from 'react'
import { Link } from 'react-router-dom'
import classNames from 'classnames'

interface HeroPageProps {
    icon?: React.ComponentType<any>
    iconLinkTo?: string
    iconClassName?: string
    className?: string
    title?: string | JSX.Element
    subtitle?: string | JSX.Element
    detail?: React.ReactNode
    body?: React.ReactNode
    cta?: JSX.Element
    lessPadding?: boolean
}

export const HeroPage: React.FunctionComponent<HeroPageProps> = props => (
    <div
        className={classNames(
            'hero-page',
            `hero-page__${props.lessPadding ? 'less' : 'default'}-padding`,
            props.className
        )}
    >
        {props.icon && (
            <div className={classNames('hero-page__icon', props.iconClassName)}>
                {props.iconLinkTo ? (
                    <Link to={props.iconLinkTo}>
                        <props.icon />
                    </Link>
                ) : (
                    <props.icon />
                )}
            </div>
        )}
        {props.title && <div className="hero-page__title">{props.title}</div>}
        {props.subtitle && <div className="hero-page__subtitle">{props.subtitle}</div>}
        {props.detail && <div className="hero-page__detail">{props.detail}</div>}
        {props.body}
        {props.cta && <div className="hero-page__cta">{props.cta}</div>}
    </div>
)
