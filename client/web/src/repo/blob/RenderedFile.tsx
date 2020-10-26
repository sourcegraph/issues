import * as React from 'react'
import * as H from 'history'
import { Markdown } from '../../../../shared/src/components/Markdown'
import { useScrollToLocationHash } from '../../components/useScrollToLocationHash'

interface Props {
    /**
     * The rendered HTML contents of the file.
     */
    dangerousInnerHTML: string

    location: H.Location
    history: H.History
}

/**
 * Displays a file whose contents are rendered to HTML, such as a Markdown file.
 */
export const RenderedFile: React.FunctionComponent<Props> = ({ dangerousInnerHTML, location, history }) => {
    useScrollToLocationHash(location)
    return (
        <div className="rendered-file">
            <div className="rendered-file__container">
                <Markdown dangerousInnerHTML={dangerousInnerHTML} history={history} />
            </div>
        </div>
    )
}
