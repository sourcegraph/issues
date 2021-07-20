import classnames from 'classnames'
import React, { useCallback, useContext, useEffect } from 'react'
import { useHistory } from 'react-router-dom'

import { LoadingSpinner } from '@sourcegraph/react-loading-spinner'
import { PlatformContextProps } from '@sourcegraph/shared/src/platform/context'
import { SettingsCascadeProps } from '@sourcegraph/shared/src/settings/settings'
import { TelemetryProps } from '@sourcegraph/shared/src/telemetry/telemetryService'
import { asError } from '@sourcegraph/shared/src/util/errors'

import { Page } from '../../../../../components/Page'
import { PageTitle } from '../../../../../components/PageTitle'
import { FORM_ERROR, FormChangeEvent } from '../../../../components/form/hooks/useForm'
import { InsightsApiContext } from '../../../../core/backend/api-provider'
import { addInsightToSettings } from '../../../../core/settings-action/insights'
import { useInsightSubjects } from '../../../../hooks/use-insight-subjects/use-insight-subjects'

import {
    SearchInsightCreationContent,
    SearchInsightCreationContentProps,
} from './components/search-insight-creation-content/SearchInsightCreationContent'
import styles from './SearchInsightCreationPage.module.scss'
import { CreateInsightFormFields } from './types'
import { getSanitizedSearchInsight } from './utils/insight-sanitizer'
import { useSearchInsightInitialValues } from './utils/use-initial-values'

export interface SearchInsightCreationPageProps
    extends PlatformContextProps<'updateSettings'>,
        SettingsCascadeProps,
        TelemetryProps {}

/** Displays create insight page with creation form. */
export const SearchInsightCreationPage: React.FunctionComponent<SearchInsightCreationPageProps> = props => {
    const { platformContext, settingsCascade, telemetryService } = props

    const history = useHistory()
    const { updateSubjectSettings, getSubjectSettings } = useContext(InsightsApiContext)

    const { initialValues, loading, setLocalStorageFormValues } = useSearchInsightInitialValues()
    const subjects = useInsightSubjects({ settingsCascade })

    useEffect(() => {
        telemetryService.logViewEvent('CodeInsightsSearchBasedCreationPage')
    }, [telemetryService])

    const handleSubmit = useCallback<SearchInsightCreationContentProps['onSubmit']>(
        async values => {
            const subjectID = values.visibility

            try {
                const settings = await getSubjectSettings(subjectID).toPromise()
                const insight = getSanitizedSearchInsight(values)
                const editedSettings = addInsightToSettings(settings.contents, insight)

                await updateSubjectSettings(platformContext, subjectID, editedSettings).toPromise()

                telemetryService.log('CodeInsightsSearchBasedCreationPageSubmitClick')

                // Clear initial values if user successfully created search insight
                setLocalStorageFormValues(undefined)

                // Navigate user to the dashboard page with new created dashboard
                history.push(`/insights/dashboards/${insight.visibility}`)
            } catch (error) {
                return { [FORM_ERROR]: asError(error) }
            }

            return
        },
        [
            getSubjectSettings,
            updateSubjectSettings,
            platformContext,
            telemetryService,
            setLocalStorageFormValues,
            history,
        ]
    )

    const handleChange = (event: FormChangeEvent<CreateInsightFormFields>): void => {
        setLocalStorageFormValues(event.values)
    }

    const handleCancel = useCallback(() => {
        telemetryService.log('CodeInsightsSearchBasedCreationPageCancelClick')
        setLocalStorageFormValues(undefined)
        history.push('/insights/dashboards/all')
    }, [history, setLocalStorageFormValues, telemetryService])

    return (
        <Page className={classnames('col-10', styles.creationPage)}>
            <PageTitle title="Create new code insight" />

            {loading && (
                // loading state for 1 click creation insight values resolve operation
                <div>
                    <LoadingSpinner className="icon-inline" /> Resolving search query
                </div>
            )}

            {
                // If we have query in URL we should be sure that we have initial values
                // from URL query based insight. If we don't have query in URl we can render
                // page without resolving URL query based insight values.
                !loading && (
                    <>
                        <div className="mb-5">
                            <h2>Create new code insight</h2>

                            <p className="text-muted">
                                Search-based code insights analyze your code based on any search query.{' '}
                                <a href="https://docs.sourcegraph.com/code_insights" target="_blank" rel="noopener">
                                    Learn more.
                                </a>
                            </p>
                        </div>

                        <SearchInsightCreationContent
                            className="pb-5"
                            dataTestId="search-insight-create-page-content"
                            settings={settingsCascade.final}
                            initialValue={initialValues}
                            subjects={subjects}
                            onSubmit={handleSubmit}
                            onCancel={handleCancel}
                            onChange={handleChange}
                        />
                    </>
                )
            }
        </Page>
    )
}
