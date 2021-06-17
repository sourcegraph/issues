import classnames from 'classnames'
import React, { useCallback, useContext, useEffect } from 'react'
import { useHistory } from 'react-router-dom'

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
    LangStatsInsightCreationContent,
    LangStatsInsightCreationContentProps,
} from './components/lang-stats-insight-creation-content/LangStatsInsightCreationContent'
import styles from './LangStatsInsightCreationPage.module.scss'
import { LangStatsCreationFormFields } from './types'
import { getSanitizedLangStatsInsight } from './utils/insight-sanitizer'

const DEFAULT_FINAL_SETTINGS = {}

export interface LangStatsInsightCreationPageProps
    extends PlatformContextProps<'updateSettings'>,
        SettingsCascadeProps,
        TelemetryProps {
    /**
     * Authenticated user info, Used to decide where code insight will appears
     * in personal dashboard (private) or in organization dashboard (public)
     * */
    authenticatedUser: Pick<AuthenticatedUser, 'id' | 'organizations'>
}

export const LangStatsInsightCreationPage: React.FunctionComponent<LangStatsInsightCreationPageProps> = props => {
    const { authenticatedUser, settingsCascade, platformContext, telemetryService } = props
    const { getSubjectSettings, updateSubjectSettings } = useContext(InsightsApiContext)
    const history = useHistory()

    const [initialFormValues, setInitialFormValues] = useLocalStorage<LangStatsCreationFormFields | undefined>(
        'insights.code-stats-creation',
        undefined
    )

    useEffect(() => {
        telemetryService.logViewEvent('CodeInsightsCodeStatsCreationPage')
    }, [telemetryService])

    const handleSubmit = useCallback<LangStatsInsightCreationContentProps['onSubmit']>(
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

                const insight = getSanitizedLangStatsInsight(values)
                const editedSettings = addInsightToCascadeSetting(settings.contents, insight)

                await updateSubjectSettings(platformContext, subjectID, editedSettings).toPromise()

                // Clear initial values if user successfully created search insight
                setInitialFormValues(undefined)
                telemetryService.log('CodeInsightsCodeStatsCreationPageSubmitClick')
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
            setInitialFormValues,
            telemetryService,
            history,
        ]
    )

    const handleCancel = useCallback(() => {
        // Clear initial values if user successfully created search insight
        setInitialFormValues(undefined)
        telemetryService.log('CodeInsightsCodeStatsCreationPageCancelClick')
        history.push('/insights')
    }, [history, setInitialFormValues, telemetryService])

    const handleChange = (event: FormChangeEvent<LangStatsCreationFormFields>): void => {
        setInitialFormValues(event.values)
    }

    const {
        organizations: { nodes: orgs },
    } = authenticatedUser

    return (
        <Page className={classnames(styles.creationPage, 'col-10')}>
            <PageTitle title="Create new code insight" />

            <div className="mb-5">
                <h2>Create new language usage insight</h2>

                <p className="text-muted">
                    Shows language usage in your repository based on number of lines of code.{' '}
                    <a href="https://docs.sourcegraph.com/code_insights" target="_blank" rel="noopener">
                        Learn more.
                    </a>
                </p>
            </div>

            <LangStatsInsightCreationContent
                className="pb-5"
                settings={settingsCascade.final ?? DEFAULT_FINAL_SETTINGS}
                initialValues={initialFormValues}
                organizations={orgs}
                onSubmit={handleSubmit}
                onCancel={handleCancel}
                onChange={handleChange}
            />
        </Page>
    )
}
