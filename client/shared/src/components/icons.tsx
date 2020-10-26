import * as React from 'react'

export interface IconProps {
    className?: string
    size?: number
    'data-tooltip'?: string
}

function sizeProps(props: IconProps): { width: number; height: number; viewBox: string } {
    const defaultSize = 24
    const size = props.size || defaultSize
    return {
        width: size,
        height: size,
        viewBox: `0 0 ${size} ${size}`,
    }
}

export const ChatIcon: React.FunctionComponent<IconProps> = props => (
    <svg {...props} {...sizeProps(props)} className={`mdi-icon${props.className ? ' ' + props.className : ''}`}>
        <path d="M 2 11.636 A 10 8 0 0 0 4.75 17.146 A 9 9 0 0 1 2 21.636 A 10.4 10.4 0 0 0 8.5 19.13 A 10 8 0 0 0 12 19.636 A 10 8 0 0 0 22 11.636 A 10 8 0 0 0 12 3.636 A 10 8 0 0 0 2 11.636 Z" />
    </svg>
)

export const CircleChevronLeftIcon: React.FunctionComponent<IconProps> = props => (
    <svg {...props} {...sizeProps(props)} className={`mdi-icon${props.className ? ' ' + props.className : ''}`}>
        <path d="M22,12c0,5.5-4.5,10-10,10S2,17.5,2,12S6.5,2,12,2S22,6.5,22,12z M15.4,16.6L10.8,12l4.6-4.6L14,6l-6,6l6,6L15.4,16.6z" />
    </svg>
)

export const CircleChevronRightIcon: React.FunctionComponent<IconProps> = props => (
    <svg {...props} {...sizeProps(props)} className={`mdi-icon${props.className ? ' ' + props.className : ''}`}>
        <path d="M22,12c0,5.5-4.5,10-10,10S2,17.5,2,12S6.5,2,12,2S22,6.5,22,12z M10,18l6-6l-6-6L8.6,7.4l4.6,4.6l-4.6,4.6L10,18z" />
    </svg>
)

export const RepoQuestionIcon: React.FunctionComponent<IconProps> = props => (
    <svg
        {...props}
        {...sizeProps(props)}
        className={`mdi-icon${props.className ? ' ' + props.className : ''}`}
        viewBox="0 0 64 64"
    >
        <title>Icons 400</title>
        <g>
            <path
                d="M50,16h-1.5c-0.3,0-0.5,0.2-0.5,0.5v35c0,0.3-0.2,0.5-0.5,0.5h-27c-0.5,0-1-0.2-1.4-0.6l-0.6-0.6c-0.1-0.1-0.1-0.2-0.1-0.4
		c0-0.3,0.2-0.5,0.5-0.5H44c1.1,0,2-0.9,2-2V12c0-1.1-0.9-2-2-2H14c-1.1,0-2,0.9-2,2v36.3c0,1.1,0.4,2.1,1.2,2.8l3.1,3.1
		c1.1,1.1,2.7,1.8,4.2,1.8H50c1.1,0,2-0.9,2-2V18C52,16.9,51.1,16,50,16z M29,44c-1.7,0-3-1.3-3-3c0-1.7,1.3-3,3-3v0
		c1.7,0,3,1.3,3,3S30.7,44,29,44z M29,16c4.4,0,8,3.1,8,7c0,6.4-5.8,6.2-5.8,11l0,0c0,0.6-0.4,1-1,1H28c-0.6,0-1-0.4-1-1
		c0-6.6,5.6-6,5.6-11c0-1.7-1.9-3-3.8-3c-1.9,0-3.8,1.3-3.8,3c0,0.6,0.1,1.1,0.3,1.7c0.2,0.5-0.1,1.1-0.6,1.3
		c-0.1,0-0.2,0.1-0.3,0.1h-2c-0.5,0-0.8-0.3-1-0.7C21.1,24.5,21,23.8,21,23C21,19.1,24.6,16,29,16z"
            />
        </g>
    </svg>
)

export const FormatListBulletedIcon: React.FunctionComponent<IconProps> = props => (
    <svg {...props} {...sizeProps(props)} className={`mdi-icon${props.className ? ' ' + props.className : ''}`}>
        <path d="M7,5H21V7H7V5M7,13V11H21V13H7M4,4.5A1.5,1.5 0 0,1 5.5,6A1.5,1.5 0 0,1 4,7.5A1.5,1.5 0 0,1 2.5,6A1.5,1.5 0 0,1 4,4.5M4,10.5A1.5,1.5 0 0,1 5.5,12A1.5,1.5 0 0,1 4,13.5A1.5,1.5 0 0,1 2.5,12A1.5,1.5 0 0,1 4,10.5M7,19V17H21V19H7M4,16.5A1.5,1.5 0 0,1 5.5,18A1.5,1.5 0 0,1 4,19.5A1.5,1.5 0 0,1 2.5,18A1.5,1.5 0 0,1 4,16.5Z" />
    </svg>
)

export const PhabricatorIcon: React.FunctionComponent<IconProps> = props => (
    <svg
        {...props}
        {...sizeProps(props)}
        className={`phabricator-icon mdi-icon${props.className ? ' ' + props.className : ''}`}
        viewBox="0 0 64 64"
        fill="currentColor"
    >
        <g>
            <g id="Oval">
                <g transform="translate(-3426.45 1547.34)">
                    <path
                        id="path15_fill"
                        d="M32,36.4c2.3,0,4.2-1.9,4.2-4.2S34.3,28,32,28c-2.3,0-4.2,1.9-4.2,4.2S29.7,36.4,32,36.4z"
                    />
                </g>
            </g>
            <path
                d="M58.8,31.2L57.6,32v0L58.8,31.2C54.8,25,45,14.6,32,14.6C19,14.6,9.2,25,5.2,31.2L6.4,32v0l-1.3-0.8L4.6,32l0.5,0.8
		C9.2,39,19,49.4,32,49.4c13,0,22.8-10.4,26.8-16.6l0.5-0.8L58.8,31.2z M32,46.4c-10.9,0-19.6-8.5-23.7-14.4
		c4.2-5.9,12.9-14.4,23.7-14.4S51.6,26.1,55.7,32C51.6,37.9,42.9,46.4,32,46.4z"
            />
            <path
                d="M44.4,33.3v-2.2L42.7,31c-0.1-0.6-0.2-1.2-0.4-1.8l1.5-0.7l-0.8-2l-1.6,0.6c-0.3-0.5-0.6-1-1-1.5l1.2-1.3L40,22.6l-1.3,1.2
		c-0.5-0.4-1-0.7-1.5-1l0.6-1.7l-2-0.8L35,21.9c-0.6-0.2-1.2-0.3-1.8-0.3l-0.1-1.8h-2.2l-0.1,1.8c-0.6,0.1-1.2,0.2-1.8,0.3l-0.8-1.6
		l-2,0.8l0.6,1.7c-0.5,0.3-1,0.6-1.5,1L24,22.6l-1.6,1.6l1.2,1.3c-0.4,0.5-0.7,1-1,1.5l-1.6-0.6l-0.8,2l1.5,0.7
		c-0.2,0.6-0.3,1.2-0.4,1.8l-1.7,0.1v2.2l1.7,0.1c0.1,0.6,0.2,1.2,0.4,1.8l-1.5,0.7l0.8,2l1.6-0.6c0.3,0.5,0.6,1,1,1.5l-1.1,1.3
		l1.6,1.6l1.3-1.2c0.5,0.4,1,0.7,1.5,1l-0.6,1.7l2,0.8l0.8-1.6c0.6,0.2,1.2,0.3,1.8,0.3l0.1,1.8h2.2l0.1-1.8
		c0.6-0.1,1.2-0.2,1.8-0.3l0.8,1.6l2-0.8l-0.6-1.7c0.5-0.3,1-0.6,1.5-1l1.3,1.2l1.6-1.6l-1.2-1.3c0.4-0.5,0.7-1,1-1.5l1.6,0.6l0.8-2
		l-1.5-0.8c0.2-0.6,0.3-1.2,0.4-1.8L44.4,33.3z M38.7,32.2c0,3.7-3,6.6-6.7,6.6s-6.7-3-6.7-6.6c0-3.7,3-6.6,6.7-6.6
		S38.7,28.5,38.7,32.2z"
            />
        </g>
    </svg>
)

export const WrapDisabledIcon: React.FunctionComponent<IconProps> = props => (
    <svg {...props} {...sizeProps(props)} className={`mdi-icon${props.className ? ' ' + props.className : ''}`}>
        <path d="M16,7H3V5H16ZM3,19H16V17H3Zm19-7L18,9v2H3v2H18v2Z" />
    </svg>
)
