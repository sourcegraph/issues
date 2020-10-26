import { RepogroupMetadata } from './types'
import * as React from 'react'

export const cncf: RepogroupMetadata = {
    title: 'Cloud Native Computing Foundation (CNCF)',
    name: 'cncf',
    url: '/cncf',
    description: (
        <>
            Search the{' '}
            <a href="https://landscape.cncf.io/project=hosted" target="_blank" rel="noopener noreferrer">
                CNCF projects
            </a>
        </>
    ),
    examples: [],
    homepageDescription: 'Search CNCF projects.',
    homepageIcon: 'https://github.com/cncf/artwork/blob/master/other/cncf/icon/color/cncf-icon-color.png?raw=true',
    lowProfile: true,
}
