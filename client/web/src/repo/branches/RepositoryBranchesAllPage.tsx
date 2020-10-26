import * as React from 'react'
import { RouteComponentProps } from 'react-router-dom'
import * as GQL from '../../../../shared/src/graphql/schema'
import { FilteredConnection, FilteredConnectionQueryArguments } from '../../components/FilteredConnection'
import { PageTitle } from '../../components/PageTitle'
import { eventLogger } from '../../tracking/eventLogger'
import { GitReferenceNode, queryGitReferences } from '../GitReference'
import { RepositoryBranchesAreaPageProps } from './RepositoryBranchesArea'
import { Observable } from 'rxjs'
import { GitRefType } from '../../graphql-operations'

interface Props extends RepositoryBranchesAreaPageProps, RouteComponentProps<{}> {}

/** A page that shows all of a repository's branches. */
export class RepositoryBranchesAllPage extends React.PureComponent<Props> {
    public componentDidMount(): void {
        eventLogger.logViewEvent('RepositoryBranchesAll')
    }

    public render(): JSX.Element | null {
        return (
            <div className="repository-branches-page">
                <PageTitle title="All branches" />
                <FilteredConnection<GQL.IGitRef>
                    className=""
                    listClassName="list-group list-group-flush"
                    noun="branch"
                    pluralNoun="branches"
                    queryConnection={this.queryBranches}
                    nodeComponent={GitReferenceNode}
                    defaultFirst={20}
                    autoFocus={true}
                    history={this.props.history}
                    location={this.props.location}
                />
            </div>
        )
    }

    private queryBranches = (args: FilteredConnectionQueryArguments): Observable<GQL.IGitRefConnection> =>
        queryGitReferences({ ...args, repo: this.props.repo.id, type: GitRefType.GIT_BRANCH })
}
