import { RepogroupMetadata } from './types'
import * as React from 'react'
import { SearchPatternType } from '../graphql-operations'

export const stanford: RepogroupMetadata = {
    title: 'Stanford University',
    name: 'stanford',
    url: '/stanford',
    description: 'Explore open-source code from Stanford students, faculty, research groups, and clubs.',
    examples: [
        {
            title: 'Find all mentions of "machine learning" in Stanford projects.',
            exampleQuery: (
                <>
                    <span className="repogroup-page__keyword-text">file:</span>README machine learning{' '}
                </>
            ),
            patternType: SearchPatternType.literal,
            rawQuery: 'machine learning',
        },
        {
            title:
                'Explore the code of specific research groups like Hazy Research, a group that investigates machine learning models and automated training set creation.',
            exampleQuery: (
                <>
                    <span className="repogroup-page__keyword-text">repo:</span>/HazyResearch/{' '}
                </>
            ),
            patternType: SearchPatternType.literal,
            rawQuery: 'repo:/HazyResearch/',
        },
        {
            title:
                'Explore the code of a specific user or organization such as Stanford University School of Medicine.',
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
            title: 'Find old-style string formatted print statements.',
            exampleQuery: (
                <>
                    <span className="repogroup-page__keyword-text">lang:</span>python print(:[args] % :[v]){' '}
                </>
            ),
            patternType: SearchPatternType.structural,
            rawQuery: 'lang:python print(:[args] % :[v])',
        },
    ],
    homepageDescription: 'Explore Stanford open-source code.',
    homepageIcon:
        'https://upload.wikimedia.org/wikipedia/commons/thumb/a/aa/Icons8_flat_graduation_cap.svg/120px-Icons8_flat_graduation_cap.svg.png',
}
