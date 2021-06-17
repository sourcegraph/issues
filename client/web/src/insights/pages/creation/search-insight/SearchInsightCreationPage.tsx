import classnames from 'classnames'
import React, { useCallback, useContext, useEffect } from 'react'
import { useHistory, useLocation } from 'react-router-dom'

import { PlatformContextProps } from '@sourcegraph/shared/src/platform/context'
import { SettingsCascadeProps } from '@sourcegraph/shared/src/settings/settings'
import { TelemetryProps } from '@sourcegraph/shared/src/telemetry/telemetryService'
import { asError } from '@sourcegraph/shared/src/util/errors'
import { useLocalStorage } from '@sourcegraph/shared/src/util/useLocalStorage'

import { AuthenticatedUser } from '../../../../auth'
import { Page } from '../../../../components/Page'
import { PageTitle } from '../../../../components/PageTitle'
import { FORM_ERROR, FormChangeEvent } from '../../../components/form/hooks/useForm'
import { InsightsApiContext } from '../../../core/backend/api-provider'
import { addInsightToCascadeSetting } from '../../../core/jsonc-operation'

import {
    SearchInsightCreationContent,
    SearchInsightCreationContentProps,
} from './components/search-insight-creation-content/SearchInsightCreationContent'
import styles from './SearchInsightCreationPage.module.scss'
import { CreateInsightFormFields } from './types'
import { getSanitizedSearchInsight } from './utils/insight-sanitizer'
import { getUrlQueryInsight } from './utils/use-url-query-insight/use-url-query-insight'

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
    const { search } = useLocation()
    const { updateSubjectSettings, getSubjectSettings } = useContext(InsightsApiContext)

    // Search insight creation UI form can take value from query param in order
    // to support 1-click insight creation from search result page.
    const queryParameterInsight = getUrlQueryInsight(search)

    // Creation UI saves all form values in local storage to be able restore these
    // values if page was fully refreshed or user came back from other page.
    const [localStorageFormValues, setInitialFormValues] = useLocalStorage<CreateInsightFormFields | undefined>(
        'insights.search-insight-creation',
        undefined
    )

    // Query param insight values have a higher priority that local storage values
    const initialFormValues = queryParameterInsight ?? localStorageFormValues

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
                const editedSettings = addInsightToCascadeSetting(settings.contents, insight)

                await updateSubjectSettings(platformContext, subjectID, editedSettings).toPromise()

                telemetryService.log('CodeInsightsSearchBasedCreationPageSubmitClick')

                // Clear initial values if user successfully created search insight
                setInitialFormValues(undefined)
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
            setInitialFormValues,
            history,
        ]
    )

    const handleChange = (event: FormChangeEvent<CreateInsightFormFields>): void => {
        setInitialFormValues(event.values)
    }

    const handleCancel = useCallback(() => {
        telemetryService.log('CodeInsightsSearchBasedCreationPageCancelClick')
        setInitialFormValues(undefined)
        history.push('/insights')
    }, [history, setInitialFormValues, telemetryService])

    const {
        organizations: { nodes: orgs },
    } = authenticatedUser

    return (
        <Page className={classnames('col-10', styles.creationPage)}>
            <PageTitle title="Create new code insight" />

            <div className="mb-5">
                <h2>Create new search-based insight</h2>

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
                initialValue={initialFormValues}
                organizations={orgs}
                onSubmit={handleSubmit}
                onCancel={handleCancel}
                onChange={handleChange}
            />
        </Page>
    )
}
