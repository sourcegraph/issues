import { Position, Range } from '@sourcegraph/extension-api-types'
import { upperFirst } from 'lodash'
import BitbucketIcon from 'mdi-react/BitbucketIcon'
import ExportIcon from 'mdi-react/ExportIcon'
import GithubIcon from 'mdi-react/GithubIcon'
import * as React from 'react'
import { merge, of, Subject, Subscription } from 'rxjs'
import { catchError, distinctUntilChanged, map, startWith, switchMap } from 'rxjs/operators'
import { PhabricatorIcon } from '../../../../shared/src/components/icons' // TODO: Switch mdi icon
import { LinkOrButton } from '../../../../shared/src/components/LinkOrButton'
import * as GQL from '../../../../shared/src/graphql/schema'
import { asError, ErrorLike, isErrorLike } from '../../../../shared/src/util/errors'
import { fetchFileExternalLinks, ExternalLink } from '../backend'
import { RevisionSpec, FileSpec } from '../../../../shared/src/util/url'

interface Props extends RevisionSpec, Partial<FileSpec> {
    repo?: GQL.Repository | null
    filePath?: string
    commitRange?: string
    position?: Position
    range?: Range

    externalLinks?: ExternalLink[]
}

interface State {
    /**
     * The external links for the current file/dir, or undefined while loading, null while not
     * needed (because not viewing a file/dir), or an error.
     */
    fileExternalLinksOrError?: ExternalLink[] | null | ErrorLike
}

/**
 * A repository header action that goes to the corresponding URL on an external code host.
 */
export class GoToCodeHostAction extends React.PureComponent<Props, State> {
    public state: State = { fileExternalLinksOrError: null }

    private componentUpdates = new Subject<Props>()
    private subscriptions = new Subscription()

    public componentDidMount(): void {
        this.subscriptions.add(
            this.componentUpdates
                .pipe(
                    startWith(this.props),
                    distinctUntilChanged(
                        (a, b) => a.repo === b.repo && a.revision === b.revision && a.filePath === b.filePath
                    ),
                    switchMap(({ repo, revision, filePath }) => {
                        if (!repo || !filePath) {
                            return of<Pick<State, 'fileExternalLinksOrError'>>({ fileExternalLinksOrError: null })
                        }
                        return merge(
                            of({ fileExternalLinksOrError: undefined }),
                            fetchFileExternalLinks({ repoName: repo.name, revision, filePath }).pipe(
                                catchError(error => [asError(error)]),
                                map(fileExternalLinksOrError => ({ fileExternalLinksOrError }))
                            )
                        )
                    })
                )
                .subscribe(
                    stateUpdate => this.setState(stateUpdate),
                    error => console.error(error)
                )
        )
    }

    public componentDidUpdate(): void {
        this.componentUpdates.next(this.props)
    }

    public componentWillUnmount(): void {
        this.subscriptions.unsubscribe()
    }

    public render(): JSX.Element | null {
        // If the default branch is undefined, set to HEAD
        const defaultBranch =
            (!isErrorLike(this.props.repo) &&
                this.props.repo &&
                this.props.repo.defaultBranch &&
                this.props.repo.defaultBranch.displayName) ||
            'HEAD'
        // If neither repo or file can be loaded, return null, which will hide all code host icons
        if (!this.props.repo || isErrorLike(this.state.fileExternalLinksOrError)) {
            return null
        }

        let externalURLs: GQL.ExternalLink[]
        if (this.props.externalLinks && this.props.externalLinks.length > 0) {
            externalURLs = this.props.externalLinks
        } else if (
            this.state.fileExternalLinksOrError === null ||
            this.state.fileExternalLinksOrError === undefined ||
            isErrorLike(this.state.fileExternalLinksOrError) ||
            this.state.fileExternalLinksOrError.length === 0
        ) {
            // If the external link for the more specific resource within the repository is loading or errored, use the
            // repository external link.
            externalURLs = this.props.repo.externalURLs
        } else {
            externalURLs = this.state.fileExternalLinksOrError
        }
        if (externalURLs.length === 0) {
            return null
        }

        // Only show the first external link for now.
        const externalURL = externalURLs[0]

        const { displayName, icon } = serviceTypeDisplayNameAndIcon(externalURL.serviceType)
        const Icon = icon || ExportIcon

        // Extract url to add branch, line numbers or commit range.
        let url = externalURL.url
        if (externalURL.serviceType === 'github' || externalURL.serviceType === 'gitlab') {
            // If in a branch, add branch path to the code host URL.
            if (this.props.revision && this.props.revision !== defaultBranch && !this.state.fileExternalLinksOrError) {
                url += `/tree/${this.props.revision}`
            }
            // If showing a comparison, add comparison specifier to the code host URL.
            if (this.props.commitRange) {
                url += `/compare/${this.props.commitRange.replace(/^\.{3}/, 'HEAD...').replace(/\.{3}$/, '...HEAD')}`
            }
            // Add range or position path to the code host URL.
            if (this.props.range) {
                url += `#L${this.props.range.start.line}-L${this.props.range.end.line}`
            } else if (this.props.position) {
                url += `#L${this.props.position.line}`
            }
        }

        return (
            <LinkOrButton
                className="nav-link test-go-to-code-host"
                to={url}
                target="_self"
                data-tooltip={`View on ${displayName}`}
            >
                <Icon className="icon-inline" />
            </LinkOrButton>
        )
    }
}

function serviceTypeDisplayNameAndIcon(
    serviceType: string | null
): { displayName: string; icon?: React.ComponentType<{ className?: string }> } {
    switch (serviceType) {
        case 'github':
            return { displayName: 'GitHub', icon: GithubIcon }
        case 'gitlab':
            return { displayName: 'GitLab' }
        case 'bitbucketServer':
            // TODO: Why is bitbucketServer (correctly) camelCase but
            // awscodecommit is (correctly) lowercase? Why is serviceType
            // not type-checked for validity?
            return { displayName: 'Bitbucket Server', icon: BitbucketIcon }
        case 'phabricator':
            return { displayName: 'Phabricator', icon: PhabricatorIcon }
        case 'awscodecommit':
            return { displayName: 'AWS CodeCommit' }
    }
    return { displayName: serviceType ? upperFirst(serviceType) : 'code host' }
}
