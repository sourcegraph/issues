import React, { useMemo } from 'react'
import { useHistory } from 'react-router-dom'

import { Settings } from '@sourcegraph/shared/src/settings/settings'

import { SubmissionErrors } from '../../../../components/form/hooks/useForm'
import { InsightType, SearchBasedInsight } from '../../../../core/types'
import { SupportedInsightSubject } from '../../../../core/types/subjects'
import { createDefaultEditSeries } from '../../creation/search-insight/components/search-insight-creation-content/hooks/use-editable-series'
import { SearchInsightCreationContent } from '../../creation/search-insight/components/search-insight-creation-content/SearchInsightCreationContent'
import { CreateInsightFormFields, InsightStep } from '../../creation/search-insight/types'
import { getSanitizedSearchInsight } from '../../creation/search-insight/utils/insight-sanitizer'

interface EditSearchBasedInsightProps {
    insight: SearchBasedInsight
    onSubmit: (insight: SearchBasedInsight) => SubmissionErrors | Promise<SubmissionErrors> | void
    finalSettings: Settings
    subjects: SupportedInsightSubject[]
}

export const EditSearchBasedInsight: React.FunctionComponent<EditSearchBasedInsightProps> = props => {
    const { insight, finalSettings = {}, subjects, onSubmit } = props
    const history = useHistory()

    const insightFormValues = useMemo<CreateInsightFormFields>(
        () => ({
            title: insight.title,
            visibility: insight.visibility,
            repositories: insight.repositories.join(', '),
            series: insight.series.map(line => createDefaultEditSeries({ ...line, valid: true })),
            stepValue: Object.values(insight.step)[0]?.toString() ?? '3',
            step: Object.keys(insight.step)[0] as InsightStep,
            allRepos: insight.type === InsightType.Backend,
        }),
        [insight]
    )

    // Handlers
    const handleSubmit = (values: CreateInsightFormFields): SubmissionErrors | Promise<SubmissionErrors> | void => {
        const sanitizedInsight = getSanitizedSearchInsight(values)

        return onSubmit(sanitizedInsight)
    }

    const handleCancel = (): void => {
        history.push(`/insights/dashboards/${insight.visibility}`)
    }

    return (
        <SearchInsightCreationContent
            mode="edit"
            className="pb-5"
            initialValue={insightFormValues}
            settings={finalSettings}
            subjects={subjects}
            dataTestId="search-insight-edit-page-content"
            onSubmit={handleSubmit}
            onCancel={handleCancel}
        />
    )
}
