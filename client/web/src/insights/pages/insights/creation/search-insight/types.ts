import { DataSeries } from '../../../../core/backend/types'
import { InsightVisibility } from '../../../../core/types'

export type InsightStep = 'hours' | 'days' | 'weeks' | 'months' | 'years'

export interface EditableDataSeries extends DataSeries {
    id: string
    valid: boolean
    edit: boolean
}

export interface CreateInsightFormFields {
    /**
     * Code Insight series setting (name of line, line query, color)
     */
    series: EditableDataSeries[]

    /**
     * Title of code insight
     */
    title: string

    /**
     * Repositories which to be used to get the info for code insights
     */
    repositories: string

    /**
     * Visibility setting which responsible for where insight will appear.
     * possible value 'personal' | '<org id 1> ... | ... <org id N>'
     */
    visibility: InsightVisibility

    /**
     * Setting for set chart step - how often do we collect data.
     */
    step: InsightStep

    /**
     * Value for insight step setting
     */
    stepValue: string

    /**
     * This settings stands for turn on/off all repos mode that means this insight
     * will be run over all repos on BE (BE insight)
     */
    allRepos: boolean
}
