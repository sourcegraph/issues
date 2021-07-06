import classnames from 'classnames'
import React, { useCallback, useContext, useEffect } from 'react'
import { useHistory } from 'react-router-dom'

import { Container } from '@sourcegraph/wildcard'
import { LoadingSpinner } from '@sourcegraph/react-loading-spinner'
import { PlatformContextProps } from '@sourcegraph/shared/src/platform/context'
import { SettingsCascadeProps } from '@sourcegraph/shared/src/settings/settings'
import { TelemetryProps } from '@sourcegraph/shared/src/telemetry/telemetryService'
import { asError } from '@sourcegraph/shared/src/util/errors'

import { AuthenticatedUser } from '../../../../auth'
import { Page } from '../../../../components/Page'
import { PageTitle } from '../../../../components/PageTitle'
import { FORM_ERROR, FormChangeEvent } from '../../../components/form/hooks/useForm'
import { InsightsApiContext } from '../../../core/backend/api-provider'
import { addInsightToSettings } from '../../../core/settings-action/insights'

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
        TelemetryProps {
    /**
     * Authenticated user info, Used to decide where code insight will appears
     * in personal dashboard (private) or in organisation dashboard (public)
     * */
    authenticatedUser: Pick<AuthenticatedUser, 'id' | 'organizations'>
}

/** Displays create insight page with creation form. */
export const SearchInsightCreationPage: React.FunctionComponent<SearchInsightCreationPageProps> = props => {
    const { platformContext, authenticatedUser, settingsCascade, telemetryService } = props

    const history = useHistory()
    const { updateSubjectSettings, getSubjectSettings } = useContext(InsightsApiContext)

    const { initialValues, loading, setLocalStorageFormValues } = useSearchInsightInitialValues()

    useEffect(() => {
        telemetryService.logViewEvent('CodeInsightsSearchBasedCreationPage')
    }, [telemetryService])

    const handleSubmit = useCallback<SearchInsightCreationContentProps['onSubmit']>(
        async values => {
            if (!authenticatedUser) {
                return
            }

            const { id: userID } = authenticatedUser

            const subjectID =
                values.visibility === 'personal'
                    ? userID
                    : // If this is not a 'personal' value than we are dealing with org id
                      values.visibility

            try {
                const settings = await getSubjectSettings(subjectID).toPromise()
                const insight = getSanitizedSearchInsight(values)
                const editedSettings = addInsightToSettings(settings.contents, insight)

                await updateSubjectSettings(platformContext, subjectID, editedSettings).toPromise()

                telemetryService.log('CodeInsightsSearchBasedCreationPageSubmitClick')

                // Clear initial values if user successfully created search insight
                setLocalStorageFormValues(undefined)
                history.push('/insights')
            } catch (error) {
                return { [FORM_ERROR]: asError(error) }
            }

            return
        },
        [
            authenticatedUser,
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
        history.push('/insights')
    }, [history, setLocalStorageFormValues, telemetryService])

    const {
        organizations: { nodes: orgs },
    } = authenticatedUser

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
                    <div className='pb-5'>
                        <header>
                            <h2>Create new code insight</h2>

                            <p className="text-muted">
                                Search-based code insights analyze your code based on any search query.{' '}
                                <a href="https://docs.sourcegraph.com/code_insights" target="_blank" rel="noopener">
                                    Learn more.
                                </a>
                            </p>
                        </header>

                        <Container>
                            <SearchInsightCreationContent
                                dataTestId="search-insight-create-page-content"
                                settings={settingsCascade.final}
                                initialValue={initialValues}
                                organizations={orgs}
                                onSubmit={handleSubmit}
                                onCancel={handleCancel}
                                onChange={handleChange}
                            />
                        </Container>
                    </div>
                )
            }
        </Page>
    )
}
