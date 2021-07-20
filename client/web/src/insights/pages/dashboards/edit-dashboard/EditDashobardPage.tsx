import classnames from 'classnames'
import MapSearchIcon from 'mdi-react/MapSearchIcon'
import React, { useMemo, useState } from 'react'
import { useHistory } from 'react-router'
import { Link } from 'react-router-dom'

import { PlatformContextProps } from '@sourcegraph/shared/src/platform/context'
import { SettingsCascadeProps } from '@sourcegraph/shared/src/settings/settings'
import { isErrorLike } from '@sourcegraph/shared/src/util/errors'
import { Container, PageHeader } from '@sourcegraph/wildcard'

import { AuthenticatedUser } from '../../../../auth'
import { HeroPage } from '../../../../components/HeroPage'
import { LoaderButton } from '../../../../components/LoaderButton'
import { Page } from '../../../../components/Page'
import { PageTitle } from '../../../../components/PageTitle'
import { Settings } from '../../../../schema/settings.schema'
import { CodeInsightsIcon } from '../../../components'
import { getSubjectDashboardByID } from '../../../hooks/use-dashboards/utils'
import { useInsightSubjects } from '../../../hooks/use-insight-subjects/use-insight-subjects'
import { InsightsDashboardCreationContent } from '../creation/components/insights-dashboard-creation-content/InsightsDashboardCreationContent'
import { useDashboardSettings } from '../creation/hooks/use-dashboard-settings'

import styles from './EditDashboardPage.module.scss'
import { useUpdateDashboardCallback } from './hooks/use-update-dashboard'

interface EditDashboardPageProps extends SettingsCascadeProps<Settings>, PlatformContextProps<'updateSettings'> {
    dashboardId: string

    authenticatedUser: Pick<AuthenticatedUser, 'id' | 'organizations' | 'username'>
}

/**
 * Displays the edit (configure) dashboard page.
 */
export const EditDashboardPage: React.FunctionComponent<EditDashboardPageProps> = props => {
    const { dashboardId, settingsCascade, authenticatedUser, platformContext } = props
    const history = useHistory()
    const subjects = useInsightSubjects({ settingsCascade })

    const [previousDashboard] = useState(() => {
        const subjects = settingsCascade.subjects
        const configureSubject = subjects?.find(
            ({ settings }) => settings && !isErrorLike(settings) && !!settings['insights.dashboards']?.[dashboardId]
        )

        if (!configureSubject || !configureSubject.settings || isErrorLike(configureSubject.settings)) {
            return undefined
        }

        const { subject, settings } = configureSubject

        return getSubjectDashboardByID(subject, settings, dashboardId)
    })

    const dashboardInitialValues = useMemo(() => {
        if (!previousDashboard) {
            return undefined
        }

        const dashboardOwnerID = previousDashboard.owner.id

        return {
            name: previousDashboard.title,
            visibility: dashboardOwnerID,
        }
    }, [previousDashboard])

    const finalDashboardSettings = useDashboardSettings({
        settingsCascade,

        // Final settings used below as a store of all existing dashboards
        // Usually we have a validation step for the title of dashboard because
        // users can't have two dashboards with the same name/id. In edit mode
        // we should allow users to have insight with id (camelCase(dashboard name))
        // which already exists in the settings. For turning off this id/title
        // validation we remove current dashboard from the final settings.
        excludeDashboardIds: [dashboardId],
    })

    const handleSubmit = useUpdateDashboardCallback({ authenticatedUser, platformContext, previousDashboard })
    const handleCancel = (): void => history.goBack()

    if (!previousDashboard) {
        return (
            <HeroPage
                icon={MapSearchIcon}
                title="Oops, we couldn't find the dashboard"
                subtitle={
                    <span>
                        We couldn't find that dashboard. Try to find the dashboard with ID:{' '}
                        <code className="badge badge-secondary">{dashboardId}</code> in your{' '}
                        <Link to={`/users/${authenticatedUser?.username}/settings`}>user or org settings</Link>
                    </span>
                }
            />
        )
    }

    return (
        <Page className={classnames('col-8', styles.page)}>
            <PageTitle title="Configure dashboard" />

            <PageHeader path={[{ icon: CodeInsightsIcon }, { text: 'Configure dashboard' }]} />

            <span className="text-muted d-block mt-2">
                Dashboards group your insights and let you share them with others.{' '}
                <a
                    href="https://docs.sourcegraph.com/code_insights/explanations/viewing_code_insights"
                    target="_blank"
                    rel="noopener"
                >
                    Learn more.
                </a>
            </span>

            <Container className="mt-4">
                <InsightsDashboardCreationContent
                    initialValues={dashboardInitialValues}
                    dashboardsSettings={finalDashboardSettings}
                    subjects={subjects}
                    onSubmit={handleSubmit}
                >
                    {formAPI => (
                        <>
                            <button type="button" className="btn btn-outline-secondary mb-2" onClick={handleCancel}>
                                Cancel
                            </button>

                            <LoaderButton
                                alwaysShowLabel={true}
                                data-testid="insight-save-button"
                                loading={formAPI.submitting}
                                label={formAPI.submitting ? 'Saving' : 'Save changes'}
                                spinnerClassName="mr-2"
                                type="submit"
                                disabled={formAPI.submitting}
                                className="d-flex btn btn-primary ml-2 mb-2"
                            />
                        </>
                    )}
                </InsightsDashboardCreationContent>
            </Container>
        </Page>
    )
}
