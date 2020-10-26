import React, { useEffect } from 'react'
import * as GQL from '../../../../../shared/src/graphql/schema'
import { PageTitle } from '../../../components/PageTitle'
import { Timestamp } from '../../../components/time/Timestamp'
import { eventLogger } from '../../../tracking/eventLogger'
import * as H from 'history'
import { scheduleRepositoryPermissionsSync } from '../../../site-admin/backend'
import { ActionContainer } from '../../../repo/settings/components/ActionContainer'

export interface RepoSettingsPermissionsPageProps {
    repo: GQL.IRepository
    history: H.History
}

/**
 * The repository settings permissions page.
 */
export const RepoSettingsPermissionsPage: React.FunctionComponent<RepoSettingsPermissionsPageProps> = ({
    repo,
    history,
}) => {
    useEffect(() => eventLogger.logViewEvent('RepoSettingsPermissions'))

    return (
        <div className="repo-settings-permissions-page w-100">
            <PageTitle title="Permissions" />
            <h2>Permissions</h2>
            {!repo.isPrivate ? (
                <div className="alert alert-info">
                    Access to this repository is not restricted, all Sourcegraph users have access.
                </div>
            ) : !repo.permissionsInfo ? (
                <div className="alert alert-info">
                    This repository is queued to sync permissions, only site admins will have access to it until syncing
                    is finished.
                </div>
            ) : (
                <div>
                    <table className="table">
                        <tbody>
                            <tr>
                                <th>Last complete sync</th>
                                <td>
                                    {repo.permissionsInfo.syncedAt ? (
                                        <Timestamp date={repo.permissionsInfo.syncedAt} />
                                    ) : (
                                        'Never'
                                    )}
                                </td>
                                <td className="text-muted">Updated by repository permissions syncing</td>
                            </tr>
                            <tr>
                                <th>Last incremental sync</th>
                                <td>
                                    <Timestamp date={repo.permissionsInfo.updatedAt} />
                                </td>
                                <td className="text-muted">Updated by user permissions syncing</td>
                            </tr>
                        </tbody>
                    </table>
                    <ScheduleRepositoryPermissionsSyncActionContainer repo={repo} history={history} />
                </div>
            )}
            <a href="/help/admin/repo/permissions#background-permissions-syncing">
                Learn more about background permissions synching.
            </a>
        </div>
    )
}

interface ScheduleRepositoryPermissionsSyncActionContainerProps {
    repo: GQL.IRepository
    history: H.History
}

class ScheduleRepositoryPermissionsSyncActionContainer extends React.PureComponent<
    ScheduleRepositoryPermissionsSyncActionContainerProps
> {
    public render(): JSX.Element | null {
        return (
            <ActionContainer
                title="Manually schedule a permissions sync"
                description={
                    <div>Site admins are able to manually schedule a permissions sync for this repository.</div>
                }
                buttonLabel="Schedule now"
                flashText="Added to queue"
                run={this.scheduleRepositoryPermissions}
                history={this.props.history}
            />
        )
    }

    private scheduleRepositoryPermissions = async (): Promise<void> => {
        await scheduleRepositoryPermissionsSync({ repository: this.props.repo.id }).toPromise()
    }
}
