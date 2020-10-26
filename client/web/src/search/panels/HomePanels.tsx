import * as React from 'react'
import { AuthenticatedUser } from '../../auth'
import { HomePanelsProps, PatternTypeProps } from '..'
import { RecentFilesPanel } from './RecentFilesPanel'
import { RecentSearchesPanel } from './RecentSearchesPanel'
import { RepositoriesPanel } from './RepositoriesPanel'
import { SavedSearchesPanel } from './SavedSearchesPanel'
import { TelemetryProps } from '../../../../shared/src/telemetry/telemetryService'
import { RepogroupPanel } from './RepogroupPanel'

interface Props extends Pick<PatternTypeProps, 'patternType'>, TelemetryProps, HomePanelsProps {
    authenticatedUser: AuthenticatedUser | null
    isSourcegraphDotCom: boolean
}

export const HomePanels: React.FunctionComponent<Props> = (props: Props) => (
    <div className="home-panels container">
        <div className="row">
            <RepositoriesPanel {...props} className="home-panels__panel col-lg-4" />
            <RecentSearchesPanel {...props} className="home-panels__panel col-lg-8" />
        </div>
        <div className="row">
            <RecentFilesPanel {...props} className="home-panels__panel col-lg-7" />

            {props.isSourcegraphDotCom ? (
                <RepogroupPanel {...props} className="home-panels__panel col-lg-5" />
            ) : (
                <SavedSearchesPanel {...props} className="home-panels__panel col-lg-5" />
            )}
        </div>
    </div>
)
