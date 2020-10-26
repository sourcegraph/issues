import * as React from 'react'
import { Subject, Subscription } from 'rxjs'
import { catchError, map, switchMap } from 'rxjs/operators'
import { IFileDiffConnection } from '../../../../shared/src/graphql/schema'
import { PlatformContextProps } from '../../../../shared/src/platform/context'
import { queryRepositoryComparisonFileDiffs } from '../backend/diffs'
import { OpenDiffInSourcegraphProps } from '../repo'
import { getPlatformName } from '../util/context'
import { SourcegraphIconButton, SourcegraphIconButtonProps } from './SourcegraphIconButton'
import classNames from 'classnames'

interface Props extends SourcegraphIconButtonProps, PlatformContextProps<'requestGraphQL'> {
    openProps: OpenDiffInSourcegraphProps
}

interface State {
    fileDiff: IFileDiffConnection | undefined
}

export class OpenDiffOnSourcegraph extends React.Component<Props, State> {
    private subscriptions = new Subscription()
    private componentUpdates = new Subject<Props>()

    constructor(props: Props) {
        super(props)
        this.state = { fileDiff: undefined }
    }

    public componentDidMount(): void {
        const { requestGraphQL } = this.props.platformContext
        this.subscriptions.add(
            // Fetch all fileDiffs in a given comparison. We rely on queryRepositoryComparisonFileDiffs
            // being memoized so that there is at most one network request when viewing
            // a commit/comparison on GitHub to get this information, despite this request occurring in
            // this component, which appears for each file in a diff.
            this.componentUpdates
                .pipe(
                    switchMap(props =>
                        queryRepositoryComparisonFileDiffs({
                            repo: this.props.openProps.repoName,
                            base: this.props.openProps.commit.baseRev,
                            head: this.props.openProps.commit.headRev,
                            requestGraphQL,
                        }).pipe(
                            map(fileDiff => ({
                                ...fileDiff,
                                // Only include the relevant file diff.
                                nodes: fileDiff.nodes.filter(node => node.oldPath === this.props.openProps.filePath),
                            })),
                            catchError(error => {
                                console.error(error)
                                return [undefined]
                            })
                        )
                    )
                )
                .subscribe(result => {
                    this.setState({ fileDiff: result })
                })
        )
        this.componentUpdates.next(this.props)
    }

    public componentWillUnmount(): void {
        this.subscriptions.unsubscribe()
    }

    public render(): JSX.Element {
        const url = this.getOpenInSourcegraphUrl(this.props.openProps)
        return (
            <SourcegraphIconButton
                {...this.props}
                className={classNames('open-on-sourcegraph', this.props.className)}
                href={url}
            />
        )
    }

    private getOpenInSourcegraphUrl(props: OpenDiffInSourcegraphProps): string {
        const baseUrl = props.sourcegraphURL
        const url = `${baseUrl}/${props.repoName}`
        const urlToCommit = `${url}/-/compare/${props.commit.baseRev}...${
            props.commit.headRev
        }?utm_source=${getPlatformName()}`

        if (this.state.fileDiff && this.state.fileDiff.nodes.length > 0) {
            // If the total number of files in the diff exceeds 25 (the default shown on commit pages),
            // make sure the commit page loads all files to make sure we can get to the file.
            const first =
                this.state.fileDiff.totalCount && this.state.fileDiff.totalCount > 25
                    ? `&first=${this.state.fileDiff.totalCount}`
                    : ''

            // Go to the specific file in the commit diff using the internalID of the matched file diff.
            return `${urlToCommit}${first}#diff-${this.state.fileDiff.nodes[0].internalID}`
        }
        // If the request for fileDiffs fails, and we can't get the internal ID, just go to the comparison page.
        return urlToCommit
    }
}
