import { RepogroupMetadata } from './types'
import { SearchPatternType } from '../../../shared/src/graphql/schema'
import * as React from 'react'

export const stanford: RepogroupMetadata = {
    title: 'Stanford University',
    name: 'stanford',
    url: '/stanford',
    description: 'Explore open-source code from Stanford students, faculty, research groups, and clubs.',
    examples: [
        {
            title: 'Explore Stanford machine learning projects.',
            exampleQuery: <>machine learning</>,
            patternType: SearchPatternType.literal,
            rawQuery: 'machine learning',
        },
        {
            title: 'Explore the code of specific research groups.',
            exampleQuery: (
                <>
                    <span className="repogroup-page__keyword-text">repo:</span>/HazyResearch/{' '}
                </>
            ),
            patternType: SearchPatternType.literal,
            rawQuery: 'repo:/HazyResearch/',
        },
        {
            title: 'Search all Stanford University School of Medicine repositories.',
            exampleQuery: (
                <>
                    <span className="repogroup-page__keyword-text">repo:</span>/susom/{' '}
                </>
            ),
            patternType: SearchPatternType.literal,
            rawQuery: 'repo:/susom/',
        },
        {
            title: 'Search for repositories related to introductory programming concepts.',
            exampleQuery: (
                <>
                    <span className="repogroup-page__keyword-text">repo:</span>recursion{' '}
                </>
            ),
            patternType: SearchPatternType.literal,
            rawQuery: 'repo:recursion',
        },
        {
            title: 'Explore the README files of thousands of projects.',
            exampleQuery: (
                <>
                    <span className="repogroup-page__keyword-text">file:</span>README.txt{' '}
                </>
            ),
            patternType: SearchPatternType.literal,
            rawQuery: 'file:README.txt',
        },
        {
            title: "Perform a general search if you're unsure of an organization or project name.",
            exampleQuery: <>space initiative</>,
            patternType: SearchPatternType.literal,
            rawQuery: 'space initiative',
        },
        {
            title: 'Explore the code of a specific user.',
            exampleQuery: (
                <>
                    <span className="repogroup-page__keyword-text">repo:</span>/RedHeadedStranger99/{' '}
                </>
            ),
            patternType: SearchPatternType.literal,
            rawQuery: 'repo:/RedHeadedStranger99/',
        },
    ],
    homepageDescription: 'Explore Stanford open-source code.',
    homepageIcon:
        'https://upload.wikimedia.org/wikipedia/en/thumb/b/b7/Stanford_University_seal_2003.svg/1200px-Stanford_University_seal_2003.svg.png',
}
