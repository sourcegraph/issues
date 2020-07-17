import * as React from 'react'
import DownloadIcon from 'mdi-react/DownloadIcon'
import { Link } from '../../../../shared/src/components/Link'
import classNames from 'classnames'

interface Props {
    className?: string
}

export const PrivateCodeCta: React.FunctionComponent<Props> = props => {
    const logoSource = '/.assets/img/download-illus.svg'
    return (
        <div className={classNames('private-code-cta d-flex flex-row card', props.className)}>
            <div className="private-code-cta__icon-column d-flex flex-column align-items-center">
                <img src={logoSource} className="private-code-cta__logo" />
            </div>
            <div>
                <h3>Search your private code</h3>
                <p>
                    Set up a private Sourcegraph instance to search your private repositories on GitHub, GitLab,
                    Bitbucket and local installations of git, perforce, svn and other code repositories.
                </p>
                <div>
                    <Link to="https://docs.sourcegraph.com/">
                        <button className="btn btn-primary">Install now</button>
                    </Link>
                </div>
            </div>
        </div>
    )
}
