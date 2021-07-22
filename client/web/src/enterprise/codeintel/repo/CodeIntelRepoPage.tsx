import * as H from 'history'
import React from 'react'

import { ThemeProps } from '@sourcegraph/shared/src/theme'

import { RepositoryFields } from '../../../graphql-operations'

interface CodeIntelRepoPageProps extends ThemeProps {
    history: H.History
    location: H.Location
    repo: RepositoryFields
}

export const CodeIntelRepoPage: React.FunctionComponent<CodeIntelRepoPageProps> = () => <p>Placeholder content.</p>
